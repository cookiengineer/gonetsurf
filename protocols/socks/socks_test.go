package socks

import (
	"context"
	"io"
	"net"
	"testing"
	"time"
)

func TestSocks5Connect(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	go func() {
		connection, err := listener.Accept()
		if err != nil {
			return
		}
		defer connection.Close()
		// Greeting.
		greeting := make([]byte, 2)
		io.ReadFull(connection, greeting)
		methods := make([]byte, greeting[1])
		io.ReadFull(connection, methods)
		connection.Write([]byte{0x05, 0x00})
		// Request.
		header := make([]byte, 4)
		io.ReadFull(connection, header)
		switch header[3] {
		case 0x03:
			length := make([]byte, 1)
			io.ReadFull(connection, length)
			host := make([]byte, length[0])
			io.ReadFull(connection, host)
		case 0x01:
			io.ReadFull(connection, make([]byte, 4))
		case 0x04:
			io.ReadFull(connection, make([]byte, 16))
		}
		io.ReadFull(connection, make([]byte, 2))
		// Reply: success, IPv4 0.0.0.0:0.
		connection.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		// Echo.
		io.Copy(connection, connection)
	}()

	dialer := &Dialer{ProxyAddress: listener.Addr().String()}
	connection, err := dialer.DialContext(context.Background(), "tcp", "example.com:80")
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	connection.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := connection.Write([]byte("ping")); err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, 4)
	if _, err := io.ReadFull(connection, buffer); err != nil {
		t.Fatal(err)
	}
	if string(buffer) != "ping" {
		t.Errorf("echo = %q", string(buffer))
	}
}
