package ipc

import (
	"bytes"
	"errors"
	"net"
	"sync"
	"testing"
)

func TestCapabilityRoundTrip(t *testing.T) {
	capability, err := NewCapability()
	if err != nil {
		t.Fatal(err)
	}
	if capability.IsZero() {
		t.Fatal("capability is zero")
	}
	parsed, err := ParseCapability(capability.String())
	if err != nil {
		t.Fatal(err)
	}
	if parsed != capability {
		t.Errorf("parsed capability = %v, want %v", parsed, capability)
	}
	if _, err := ParseCapability("not-hex"); err == nil {
		t.Fatal("expected a capability parse error")
	}
}

func TestFrameRoundTrip(t *testing.T) {
	capability, _ := NewCapability()
	message := Message{Kind: KindRequest, RequestID: 42, Capability: capability, Payload: []byte("hello")}
	var buffer bytes.Buffer
	if err := WriteFrame(&buffer, message); err != nil {
		t.Fatal(err)
	}
	decoded, err := ReadFrame(&buffer, MaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Kind != message.Kind || decoded.RequestID != message.RequestID || decoded.Capability != message.Capability || string(decoded.Payload) != "hello" {
		t.Errorf("decoded = %#v", decoded)
	}
}

func TestFrameErrors(t *testing.T) {
	if err := WriteFrame(&bytes.Buffer{}, Message{Payload: make([]byte, MaxPayload+1)}); !errors.Is(err, ErrPayloadTooLarge) {
		t.Errorf("oversized write error = %v", err)
	}
	var buffer bytes.Buffer
	header := make([]byte, headerSize)
	header[0] = 'X'
	buffer.Write(header)
	if _, err := ReadFrame(&buffer, MaxPayload); !errors.Is(err, ErrInvalidFrame) {
		t.Errorf("invalid magic error = %v", err)
	}

	buffer.Reset()
	if err := WriteFrame(&buffer, Message{Kind: KindRequest, Payload: make([]byte, 100)}); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadFrame(&buffer, 10); !errors.Is(err, ErrPayloadTooLarge) {
		t.Errorf("limited payload error = %v", err)
	}

	buffer.Reset()
	header[0], header[1], header[2], header[3] = 'G', 'N', 'I', 'P'
	buffer.Write(header)
	if _, err := ReadFrame(&buffer, MaxPayload); err == nil {
		t.Fatal("expected a truncated frame error")
	}
}

func TestConnSendReceive(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()
	sender := NewConn(client)
	receiver := NewConn(server)
	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		for index := 0; index < 32; index++ {
			if err := sender.Send(Message{Kind: KindEvent, RequestID: uint64(index), Payload: []byte("event")}); err != nil {
				t.Errorf("send: %v", err)
				return
			}
			if index%8 == 0 {
				if err := sender.Cancel(uint64(index)); err != nil {
					t.Errorf("cancel: %v", err)
					return
				}
			}
		}
	}()
	received := 0
	for received < 32 {
		message, err := receiver.Receive()
		if err != nil {
			t.Fatal(err)
		}
		if message.Kind == KindCancel {
			continue
		}
		if message.Kind != KindEvent || message.RequestID != uint64(received) {
			t.Errorf("message = %#v", message)
		}
		received++
	}
	waitGroup.Wait()
}

func TestConcurrentSends(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()
	sender := NewConn(client)
	receiver := NewConn(server)
	var waitGroup sync.WaitGroup
	const count = 16
	for index := 0; index < count; index++ {
		waitGroup.Add(1)
		go func(id int) {
			defer waitGroup.Done()
			if err := sender.Send(Message{Kind: KindEvent, RequestID: uint64(id)}); err != nil {
				t.Errorf("send %d: %v", id, err)
			}
		}(index)
	}
	for index := 0; index < count; index++ {
		if _, err := receiver.Receive(); err != nil {
			t.Fatal(err)
		}
	}
	waitGroup.Wait()
}
