// Package socks implements a SOCKS5 client (RFC 1928).
package socks

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
)

// Dialer forwards connections through a SOCKS5 proxy.
type Dialer struct {
	ProxyAddress string
	Username     string
	Password     string
	Forward      BaseDialer
}

// BaseDialer establishes network connections.
type BaseDialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// Errors returned by the SOCKS5 handshake.
var (
	ErrHandshake = errors.New("socks: handshake failed")
	ErrReply     = errors.New("socks: connection request rejected")
)

// DialContext establishes a connection to address through the proxy.
func (dialer *Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	forward := dialer.Forward
	if forward == nil {
		forward = &net.Dialer{}
	}
	connection, err := forward.DialContext(ctx, "tcp", dialer.ProxyAddress)
	if err != nil {
		return nil, err
	}
	if err := dialer.handshake(connection); err != nil {
		connection.Close()
		return nil, err
	}
	if err := dialer.connect(connection, address); err != nil {
		connection.Close()
		return nil, err
	}
	return connection, nil
}

func (dialer *Dialer) handshake(connection net.Conn) error {
	methods := []byte{0x00}
	if dialer.Username != "" || dialer.Password != "" {
		methods = []byte{0x02, 0x00}
	}
	request := append([]byte{0x05, byte(len(methods))}, methods...)
	if _, err := connection.Write(request); err != nil {
		return err
	}
	response := make([]byte, 2)
	if _, err := io.ReadFull(connection, response); err != nil {
		return err
	}
	if response[0] != 0x05 {
		return ErrHandshake
	}
	switch response[1] {
	case 0x00:
		return nil
	case 0x02:
		return dialer.authenticate(connection)
	default:
		return ErrHandshake
	}
}

func (dialer *Dialer) authenticate(connection net.Conn) error {
	if len(dialer.Username) > 255 || len(dialer.Password) > 255 {
		return ErrHandshake
	}
	request := []byte{0x01, byte(len(dialer.Username))}
	request = append(request, dialer.Username...)
	request = append(request, byte(len(dialer.Password)))
	request = append(request, dialer.Password...)
	if _, err := connection.Write(request); err != nil {
		return err
	}
	response := make([]byte, 2)
	if _, err := io.ReadFull(connection, response); err != nil {
		return err
	}
	if response[1] != 0x00 {
		return ErrHandshake
	}
	return nil
}

func (dialer *Dialer) connect(connection net.Conn, address string) error {
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 0 || port > 65535 {
		return fmt.Errorf("socks: invalid port %q", portText)
	}

	request := []byte{0x05, 0x01, 0x00}
	if ip := net.ParseIP(host); ip != nil {
		if ipv4 := ip.To4(); ipv4 != nil {
			request = append(request, 0x01)
			request = append(request, ipv4...)
		} else {
			request = append(request, 0x04)
			request = append(request, ip.To16()...)
		}
	} else {
		if len(host) > 255 {
			return ErrHandshake
		}
		request = append(request, 0x03, byte(len(host)))
		request = append(request, host...)
	}
	request = binary.BigEndian.AppendUint16(request, uint16(port))
	if _, err := connection.Write(request); err != nil {
		return err
	}

	header := make([]byte, 4)
	if _, err := io.ReadFull(connection, header); err != nil {
		return err
	}
	if header[1] != 0x00 {
		return fmt.Errorf("%w: reply code 0x%02x", ErrReply, header[1])
	}
	if err := discardBoundAddress(connection, header[3]); err != nil {
		return err
	}
	return nil
}

func discardBoundAddress(connection net.Conn, addressType byte) error {
	switch addressType {
	case 0x01:
		return discard(connection, 4+2)
	case 0x04:
		return discard(connection, 16+2)
	case 0x03:
		length := make([]byte, 1)
		if _, err := io.ReadFull(connection, length); err != nil {
			return err
		}
		return discard(connection, int(length[0])+2)
	default:
		return ErrHandshake
	}
}

func discard(connection net.Conn, count int) error {
	buffer := make([]byte, count)
	_, err := io.ReadFull(connection, buffer)
	return err
}
