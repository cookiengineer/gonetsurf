//go:build !linux

package ipc

import (
	"fmt"
	"net"
)

func createSharedBuffer(size int) (SharedBuffer, error) {
	return nil, fmt.Errorf("ipc: shared memory is only supported on Linux")
}

// SendFD is unsupported on this platform.
func SendFD(connection *net.UnixConn, data []byte, fileDescriptor int) error {
	return fmt.Errorf("ipc: file-descriptor passing is only supported on Linux")
}

// ReceiveFD is unsupported on this platform.
func ReceiveFD(connection *net.UnixConn, buffer []byte) ([]byte, int, error) {
	return nil, -1, fmt.Errorf("ipc: file-descriptor passing is only supported on Linux")
}
