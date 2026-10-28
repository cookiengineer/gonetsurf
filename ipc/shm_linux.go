//go:build linux

package ipc

import (
	"fmt"
	"net"
	"os"

	"golang.org/x/sys/unix"
)

type memfdBuffer struct {
	file *os.File
	data []byte
}

func createSharedBuffer(size int) (SharedBuffer, error) {
	if size <= 0 {
		return nil, fmt.Errorf("ipc: invalid shared buffer size")
	}
	fd, err := unix.MemfdCreate("gonetsurf-shm", unix.MFD_CLOEXEC)
	if err != nil {
		return nil, err
	}
	if err := unix.Ftruncate(fd, int64(size)); err != nil {
		unix.Close(fd)
		return nil, err
	}
	data, err := unix.Mmap(fd, 0, size, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		unix.Close(fd)
		return nil, err
	}
	file := os.NewFile(uintptr(fd), "gonetsurf-shm")
	return &memfdBuffer{file: file, data: data}, nil
}

func (buffer *memfdBuffer) Bytes() []byte {
	return buffer.data
}

func (buffer *memfdBuffer) Fd() int {
	return int(buffer.file.Fd())
}

func (buffer *memfdBuffer) Close() error {
	var firstError error
	if buffer.data != nil {
		if err := unix.Munmap(buffer.data); err != nil {
			firstError = err
		}
		buffer.data = nil
	}
	if buffer.file != nil {
		if err := buffer.file.Close(); err != nil && firstError == nil {
			firstError = err
		}
		buffer.file = nil
	}
	return firstError
}

// SendFD sends data and fileDescriptor over connection.
func SendFD(connection *net.UnixConn, data []byte, fileDescriptor int) error {
	raw, err := connection.SyscallConn()
	if err != nil {
		return err
	}
	control := unix.UnixRights(fileDescriptor)
	var sendError error
	if err := raw.Write(func(fd uintptr) bool {
		sendError = unix.Sendmsg(int(fd), data, control, nil, 0)
		return sendError != unix.EAGAIN
	}); err != nil {
		return err
	}
	return sendError
}

// ReceiveFD receives data and a file descriptor over connection.
func ReceiveFD(connection *net.UnixConn, buffer []byte) ([]byte, int, error) {
	raw, err := connection.SyscallConn()
	if err != nil {
		return nil, -1, err
	}
	control := make([]byte, unix.CmsgSpace(4))
	var received int
	var controlLength int
	var receiveError error
	if err := raw.Read(func(fd uintptr) bool {
		received, controlLength, _, _, receiveError = unix.Recvmsg(int(fd), buffer, control, 0)
		return receiveError != unix.EAGAIN
	}); err != nil {
		return nil, -1, err
	}
	if receiveError != nil {
		return nil, -1, receiveError
	}
	messages, err := unix.ParseSocketControlMessage(control[:controlLength])
	if err != nil {
		return nil, -1, err
	}
	for index := range messages {
		rights, err := unix.ParseUnixRights(&messages[index])
		if err != nil {
			continue
		}
		if len(rights) > 0 {
			return buffer[:received], rights[0], nil
		}
	}
	return buffer[:received], -1, fmt.Errorf("ipc: no file descriptor received")
}
