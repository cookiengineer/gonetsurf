package ipc

import (
	"bytes"
	"testing"
)

func FuzzReadFrame(f *testing.F) {
	var buffer bytes.Buffer
	_ = WriteFrame(&buffer, Message{Kind: KindRequest, RequestID: 1, Payload: []byte("hello")})
	f.Add(buffer.Bytes())
	f.Add([]byte("GNIP"))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ReadFrame(bytes.NewReader(data), 4096)
	})
}
