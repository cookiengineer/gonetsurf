// Package websockets implements an RFC 6455 WebSocket client.
package websockets

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"unicode/utf8"

	"gonetsurf/protocols/http1"
	"gonetsurf/urls"
)

const acceptMagic = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// Opcode is a WebSocket frame opcode.
type Opcode byte

// Frame opcodes.
const (
	OpcodeContinuation Opcode = 0x0
	OpcodeText         Opcode = 1
	OpcodeBinary       Opcode = 2
	OpcodeClose        Opcode = 8
	OpcodePing         Opcode = 9
	OpcodePong         Opcode = 10
)

// Errors returned by the WebSocket client.
var (
	ErrHandshake  = errors.New("websockets: handshake failed")
	ErrNotUpgrade = errors.New("websockets: server did not switch protocols")
	ErrProtocol   = errors.New("websockets: protocol error")
	ErrClosed     = errors.New("websockets: connection closed")
)

// Dialer establishes network connections.
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// AcceptKey computes the Sec-WebSocket-Accept value for a client key.
func AcceptKey(key string) string {
	hash := sha1.Sum([]byte(key + acceptMagic))
	return base64.StdEncoding.EncodeToString(hash[:])
}

// Conn is a WebSocket connection.
type Conn struct {
	connection net.Conn
	reader     *bufio.Reader
	writer     *bufio.Writer
	maxPayload int64
}

// Dial performs the opening handshake and returns a connection.
func Dial(ctx context.Context, target *urls.URL, header http1.Header, dialer Dialer, tlsConfig *tls.Config) (*Conn, error) {
	if dialer == nil {
		dialer = &net.Dialer{}
	}
	address, useTLS := addressFor(target)
	rawConnection, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	connection := rawConnection
	if useTLS {
		config := tlsConfig
		if config != nil {
			config = config.Clone()
		} else {
			config = &tls.Config{}
		}
		if config.ServerName == "" {
			config.ServerName = target.Host
		}
		tlsConnection := tls.Client(rawConnection, config)
		if err := tlsConnection.HandshakeContext(ctx); err != nil {
			rawConnection.Close()
			return nil, err
		}
		connection = tlsConnection
	}

	keyBytes := make([]byte, 16)
	if _, err := rand.Read(keyBytes); err != nil {
		connection.Close()
		return nil, err
	}
	key := base64.StdEncoding.EncodeToString(keyBytes)

	requestHeader := header.Clone()
	requestHeader.Set("Host", target.HostWithPort())
	requestHeader.Set("Upgrade", "websocket")
	requestHeader.Set("Connection", "Upgrade")
	requestHeader.Set("Sec-WebSocket-Key", key)
	requestHeader.Set("Sec-WebSocket-Version", "13")
	request := &http1.Request{Method: "GET", URL: target, Proto: "HTTP/1.1", Header: requestHeader}

	writer := bufio.NewWriter(connection)
	if err := http1.WriteRequest(writer, request); err != nil {
		connection.Close()
		return nil, err
	}
	reader := bufio.NewReader(connection)
	response, err := http1.ReadResponse(reader, request)
	if err != nil {
		connection.Close()
		return nil, err
	}
	if response.StatusCode != 101 {
		response.Close()
		connection.Close()
		return nil, ErrNotUpgrade
	}
	if response.Header.Get("Sec-WebSocket-Accept") != AcceptKey(key) {
		connection.Close()
		return nil, ErrHandshake
	}
	return &Conn{connection: connection, reader: reader, writer: writer, maxPayload: 16 * 1024 * 1024}, nil
}

func addressFor(target *urls.URL) (address string, useTLS bool) {
	host := target.Host
	port := target.Port
	switch target.Scheme {
	case "wss":
		useTLS = true
		if port == "" {
			port = "443"
		}
	default:
		if port == "" {
			port = "80"
		}
	}
	return net.JoinHostPort(host, port), useTLS
}

// WriteMessage sends a data frame, masking it as required for clients.
func (connection *Conn) WriteMessage(opcode Opcode, payload []byte) error {
	return connection.writeFrame(true, opcode, payload)
}

// WriteControl sends a control frame. Control frames must be at most 125 bytes
// and must not be fragmented.
func (connection *Conn) WriteControl(opcode Opcode, payload []byte) error {
	if opcode < OpcodeClose || opcode > OpcodePong || len(payload) > 125 {
		return ErrProtocol
	}
	return connection.writeFrame(true, opcode, payload)
}

func (connection *Conn) writeFrame(fin bool, opcode Opcode, payload []byte) error {
	if int64(len(payload)) > connection.maxPayload {
		return ErrProtocol
	}
	first := byte(opcode)
	if fin {
		first |= 0x80
	}
	header := []byte{first}
	maskBit := byte(0x80)
	length := len(payload)
	switch {
	case length < 126:
		header = append(header, maskBit|byte(length))
	case length < 65536:
		header = append(header, maskBit|126)
		header = binary.BigEndian.AppendUint16(header, uint16(length))
	default:
		header = append(header, maskBit|127)
		header = binary.BigEndian.AppendUint64(header, uint64(length))
	}
	maskKey := make([]byte, 4)
	if _, err := rand.Read(maskKey); err != nil {
		return err
	}
	header = append(header, maskKey...)
	masked := make([]byte, length)
	for index := 0; index < length; index++ {
		masked[index] = payload[index] ^ maskKey[index%4]
	}
	if _, err := connection.writer.Write(header); err != nil {
		return err
	}
	if _, err := connection.writer.Write(masked); err != nil {
		return err
	}
	return connection.writer.Flush()
}

// ReadMessage reads the next complete data message, handling fragmentation and
// responding to pings.
func (connection *Conn) ReadMessage() (Opcode, []byte, error) {
	var message []byte
	var messageOpcode Opcode
	fragmented := false
	for {
		fin, opcode, payload, err := connection.readFrame()
		if err != nil {
			return 0, nil, err
		}
		switch opcode {
		case OpcodePing:
			if err := connection.WriteControl(OpcodePong, payload); err != nil {
				return 0, nil, err
			}
			continue
		case OpcodePong:
			continue
		case OpcodeClose:
			connection.writeFrame(true, OpcodeClose, nil)
			return 0, nil, ErrClosed
		case OpcodeContinuation:
			if !fragmented {
				return 0, nil, ErrProtocol
			}
			message = append(message, payload...)
			fragmented = !fin
		case OpcodeText, OpcodeBinary:
			if fragmented {
				return 0, nil, ErrProtocol
			}
			messageOpcode = opcode
			message = append(message, payload...)
			fragmented = !fin
		default:
			return 0, nil, ErrProtocol
		}
		if int64(len(message)) > connection.maxPayload {
			return 0, nil, ErrProtocol
		}
		if fin && !fragmented {
			if messageOpcode == OpcodeText && !utf8.Valid(message) {
				return 0, nil, ErrProtocol
			}
			return messageOpcode, message, nil
		}
	}
}

func (connection *Conn) readFrame() (bool, Opcode, []byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(connection.reader, header); err != nil {
		return false, 0, nil, err
	}
	if header[0]&0x70 != 0 {
		// Reserved bits must be zero.
		return false, 0, nil, ErrProtocol
	}
	fin := header[0]&0x80 != 0
	opcode := Opcode(header[0] & 0x0F)
	if opcode >= 0x3 && opcode <= 0x7 || opcode >= 0xB {
		// Reserved opcodes.
		return false, 0, nil, ErrProtocol
	}
	masked := header[1]&0x80 != 0
	if masked {
		// A client must reject masked frames from the server.
		return false, 0, nil, ErrProtocol
	}
	length := int64(header[1] & 0x7F)
	switch length {
	case 126:
		buffer := make([]byte, 2)
		if _, err := io.ReadFull(connection.reader, buffer); err != nil {
			return false, 0, nil, err
		}
		length = int64(binary.BigEndian.Uint16(buffer))
	case 127:
		buffer := make([]byte, 8)
		if _, err := io.ReadFull(connection.reader, buffer); err != nil {
			return false, 0, nil, err
		}
		length = int64(binary.BigEndian.Uint64(buffer))
	}
	if opcode >= OpcodeClose {
		// Control frames must not be fragmented and must be at most 125 bytes.
		if !fin || length > 125 {
			return false, 0, nil, ErrProtocol
		}
	}
	if length > connection.maxPayload {
		return false, 0, nil, ErrProtocol
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(connection.reader, payload); err != nil {
		return false, 0, nil, err
	}
	if opcode == OpcodeClose {
		if !validClosePayload(payload) {
			return false, 0, nil, ErrProtocol
		}
	}
	return fin, opcode, payload, nil
}

// validClosePayload reports whether a close frame payload carries a valid
// status code. See RFC 6455 §7.4.
func validClosePayload(payload []byte) bool {
	if len(payload) == 0 {
		return true
	}
	if len(payload) == 1 {
		return false
	}
	code := binary.BigEndian.Uint16(payload[:2])
	switch {
	case code >= 3000 && code <= 4999:
		return true
	case code == 1000, code == 1001, code == 1002, code == 1003:
		return true
	case code >= 1007 && code <= 1011:
		return true
	default:
		return false
	}
}

// Close sends a close frame and closes the underlying connection.
func (connection *Conn) Close() error {
	connection.writeFrame(true, OpcodeClose, nil)
	return connection.connection.Close()
}
