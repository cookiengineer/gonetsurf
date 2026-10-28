// Package ipc implements versioned message framing, capability tokens, and
// shared-memory buffers for the browser's broker/renderer boundary.
package ipc

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

// Version is the wire protocol version.
const Version uint16 = 1

// MaxPayload is the default maximum payload size.
const MaxPayload = 16 << 20

var frameMagic = [4]byte{'G', 'N', 'I', 'P'}

// FrameKind identifies a protocol message.
type FrameKind uint8

const (
	// KindHandshake establishes a session and capabilities.
	KindHandshake FrameKind = 1
	// KindRequest carries a request.
	KindRequest FrameKind = 2
	// KindResponse carries a response.
	KindResponse FrameKind = 3
	// KindEvent carries an asynchronous event.
	KindEvent FrameKind = 4
	// KindCancel cancels an in-flight request.
	KindCancel FrameKind = 5
	// KindError reports a protocol or broker error.
	KindError FrameKind = 6
)

// Capability is a 128-bit capability token.
type Capability [16]byte

// NewCapability returns a cryptographically random capability token.
func NewCapability() (Capability, error) {
	var capability Capability
	if _, err := rand.Read(capability[:]); err != nil {
		return Capability{}, err
	}
	return capability, nil
}

// IsZero reports whether the token is empty.
func (capability Capability) IsZero() bool {
	var zero Capability
	return capability == zero
}

// String returns the token as hexadecimal text.
func (capability Capability) String() string {
	return hex.EncodeToString(capability[:])
}

// ParseCapability parses a hexadecimal capability token.
func ParseCapability(value string) (Capability, error) {
	var capability Capability
	if len(value) != hex.EncodedLen(len(capability)) {
		return capability, fmt.Errorf("ipc: invalid capability length")
	}
	decoded, err := hex.DecodeString(value)
	if err != nil {
		return capability, err
	}
	copy(capability[:], decoded)
	return capability, nil
}

// Message is one framed protocol message.
type Message struct {
	Kind       FrameKind
	Flags      uint8
	RequestID  uint64
	Capability Capability
	Payload    []byte
}

const headerSize = 36

// ErrInvalidFrame reports malformed framing.
var ErrInvalidFrame = errors.New("ipc: invalid frame")

// ErrPayloadTooLarge reports a frame exceeding the configured limit.
var ErrPayloadTooLarge = errors.New("ipc: payload too large")

// WriteFrame writes one framed message.
func WriteFrame(writer io.Writer, message Message) error {
	if len(message.Payload) > MaxPayload {
		return ErrPayloadTooLarge
	}
	header := make([]byte, headerSize)
	copy(header[0:4], frameMagic[:])
	binary.BigEndian.PutUint16(header[4:6], Version)
	header[6] = byte(message.Kind)
	header[7] = message.Flags
	binary.BigEndian.PutUint64(header[8:16], message.RequestID)
	copy(header[16:32], message.Capability[:])
	binary.BigEndian.PutUint32(header[32:36], uint32(len(message.Payload)))
	if _, err := writer.Write(header); err != nil {
		return err
	}
	if len(message.Payload) > 0 {
		if _, err := writer.Write(message.Payload); err != nil {
			return err
		}
	}
	return nil
}

// ReadFrame reads one framed message, enforcing maximumPayload.
func ReadFrame(reader io.Reader, maximumPayload int) (Message, error) {
	if maximumPayload <= 0 || maximumPayload > MaxPayload {
		maximumPayload = MaxPayload
	}
	header := make([]byte, headerSize)
	if _, err := io.ReadFull(reader, header); err != nil {
		return Message{}, err
	}
	if [4]byte{header[0], header[1], header[2], header[3]} != frameMagic {
		return Message{}, ErrInvalidFrame
	}
	if binary.BigEndian.Uint16(header[4:6]) != Version {
		return Message{}, fmt.Errorf("ipc: unsupported version %d", binary.BigEndian.Uint16(header[4:6]))
	}
	kind := FrameKind(header[6])
	switch kind {
	case KindHandshake, KindRequest, KindResponse, KindEvent, KindCancel, KindError:
	default:
		return Message{}, fmt.Errorf("ipc: unknown frame kind %d", kind)
	}
	length := int(binary.BigEndian.Uint32(header[32:36]))
	if length < 0 || length > maximumPayload {
		return Message{}, ErrPayloadTooLarge
	}
	var message Message
	message.Kind = kind
	message.Flags = header[7]
	message.RequestID = binary.BigEndian.Uint64(header[8:16])
	copy(message.Capability[:], header[16:32])
	if length > 0 {
		message.Payload = make([]byte, length)
		if _, err := io.ReadFull(reader, message.Payload); err != nil {
			return Message{}, err
		}
	}
	return message, nil
}
