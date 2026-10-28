//go:build linux

package ipc

import (
	"net"
	"os"
	"testing"

	"golang.org/x/sys/unix"
)

func TestSharedBuffer(t *testing.T) {
	buffer, err := CreateSharedBuffer(4096)
	if err != nil {
		t.Fatal(err)
	}
	defer buffer.Close()
	copy(buffer.Bytes(), "hello shared memory")
	if string(buffer.Bytes()[:19]) != "hello shared memory" {
		t.Errorf("buffer = %q", buffer.Bytes()[:19])
	}
	if buffer.Fd() < 0 {
		t.Error("invalid file descriptor")
	}
}

func unixPair(t *testing.T) (*net.UnixConn, *net.UnixConn) {
	t.Helper()
	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_DGRAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	firstFile := os.NewFile(uintptr(fds[0]), "ipc-first")
	secondFile := os.NewFile(uintptr(fds[1]), "ipc-second")
	firstConnection, err := net.FileConn(firstFile)
	if err != nil {
		t.Fatal(err)
	}
	secondConnection, err := net.FileConn(secondFile)
	if err != nil {
		t.Fatal(err)
	}
	firstFile.Close()
	secondFile.Close()
	return firstConnection.(*net.UnixConn), secondConnection.(*net.UnixConn)
}

func TestSendReceiveFD(t *testing.T) {
	first, second := unixPair(t)
	defer first.Close()
	defer second.Close()
	buffer, err := CreateSharedBuffer(128)
	if err != nil {
		t.Fatal(err)
	}
	defer buffer.Close()
	copy(buffer.Bytes(), "shared payload")
	if err := SendFD(first, []byte("fd follows"), buffer.Fd()); err != nil {
		t.Fatal(err)
	}
	data, receivedFD, err := ReceiveFD(second, make([]byte, 64))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "fd follows" {
		t.Errorf("data = %q", data)
	}
	receivedFile := os.NewFile(uintptr(receivedFD), "received-shm")
	defer receivedFile.Close()
	content := make([]byte, 64)
	if _, err := receivedFile.ReadAt(content, 0); err != nil {
		t.Fatal(err)
	}
	if string(content[:14]) != "shared payload" {
		t.Errorf("received content = %q", content[:14])
	}
}
