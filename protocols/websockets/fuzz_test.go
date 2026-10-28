package websockets

import (
	"bufio"
	"bytes"
	"io"
	"testing"
)

func FuzzReadFrame(f *testing.F) {
	f.Add([]byte{0x81, 0x02, 'h', 'i'})
	f.Add([]byte{0x88, 0x02, 0x03, 0xE8})
	f.Add([]byte{0x89, 0x00})
	f.Add([]byte{0x81, 0xFE})
	f.Fuzz(func(t *testing.T, data []byte) {
		connection := &Conn{
			reader:     bufio.NewReader(bytes.NewReader(data)),
			writer:     bufio.NewWriter(io.Discard),
			maxPayload: 1 << 20,
		}
		for {
			if _, err := connection.reader.Peek(1); err != nil {
				break
			}
			if _, _, _, err := connection.readFrame(); err != nil {
				break
			}
		}
	})
}

func FuzzClosePayload(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0x03, 0xE8})
	f.Add([]byte{0x00})
	f.Fuzz(func(t *testing.T, payload []byte) {
		_ = validClosePayload(payload)
	})
}
