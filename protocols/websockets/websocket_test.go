package websockets

import (
	"bufio"
	"io"
	"net"
	"testing"
)

func TestAcceptKey(t *testing.T) {
	// Vector from RFC 6455 section 1.3.
	got := AcceptKey("dGhlIHNhbXBsZSBub25jZQ==")
	if got != "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=" {
		t.Fatalf("AcceptKey = %q", got)
	}
}

func TestFrameRoundTrip(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	connection := &Conn{
		connection: client,
		reader:     bufio.NewReader(client),
		writer:     bufio.NewWriter(client),
		maxPayload: 1024,
	}

	go func() {
		connection.WriteMessage(OpcodeText, []byte("hello"))
	}()

	// Read the frame from the other end and mimic a server by unmasking it.
	header := make([]byte, 2)
	if _, err := io.ReadFull(server, header); err != nil {
		t.Fatal(err)
	}
	if header[0]&0x0F != byte(OpcodeText) {
		t.Fatalf("opcode = %d", header[0]&0x0F)
	}
	if header[1]&0x80 == 0 {
		t.Fatal("client frames must be masked")
	}
	length := int(header[1] & 0x7F)
	maskKey := make([]byte, 4)
	if _, err := io.ReadFull(server, maskKey); err != nil {
		t.Fatal(err)
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(server, payload); err != nil {
		t.Fatal(err)
	}
	for index := range payload {
		payload[index] ^= maskKey[index%4]
	}
	if string(payload) != "hello" {
		t.Errorf("payload = %q", string(payload))
	}
}
