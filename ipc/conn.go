package ipc

import (
	"bufio"
	"io"
	"net"
	"sync"
	"time"
)

// Conn is a framed message connection.
type Conn struct {
	connection io.ReadWriteCloser
	reader     *bufio.Reader
	writeMutex sync.Mutex
	maxPayload int
}

// NewConn wraps connection with the IPC framing protocol.
func NewConn(connection io.ReadWriteCloser) *Conn {
	return &Conn{connection: connection, reader: bufio.NewReader(connection), maxPayload: MaxPayload}
}

// SetMaxPayload sets the receive limit.
func (conn *Conn) SetMaxPayload(maximum int) {
	if maximum > 0 && maximum <= MaxPayload {
		conn.maxPayload = maximum
	}
}

// Send writes one message.
func (conn *Conn) Send(message Message) error {
	conn.writeMutex.Lock()
	defer conn.writeMutex.Unlock()
	return WriteFrame(conn.connection, message)
}

// Receive reads one message.
func (conn *Conn) Receive() (Message, error) {
	return ReadFrame(conn.reader, conn.maxPayload)
}

// Cancel sends a cancellation for requestID.
func (conn *Conn) Cancel(requestID uint64) error {
	return conn.Send(Message{Kind: KindCancel, RequestID: requestID})
}

// SetDeadline sets the connection deadline when supported.
func (conn *Conn) SetDeadline(deadline time.Time) error {
	if setter, ok := conn.connection.(interface{ SetDeadline(time.Time) error }); ok {
		return setter.SetDeadline(deadline)
	}
	return nil
}

// Close closes the connection.
func (conn *Conn) Close() error {
	return conn.connection.Close()
}

// LocalAddr returns the local network address when available.
func (conn *Conn) LocalAddr() net.Addr {
	if network, ok := conn.connection.(net.Conn); ok {
		return network.LocalAddr()
	}
	return nil
}
