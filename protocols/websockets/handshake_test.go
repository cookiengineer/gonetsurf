package websockets

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"

	"gonetsurf/protocols/http1"
	"gonetsurf/urls"
)

// newTestServer starts a listener that completes the opening handshake and then
// runs handler against the connected socket.
func newTestServer(t *testing.T, handler func(connection net.Conn, reader *bufio.Reader)) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		connection, err := listener.Accept()
		if err != nil {
			return
		}
		defer connection.Close()
		reader := bufio.NewReader(connection)
		request, err := http1.ReadRequest(reader)
		if err != nil {
			return
		}
		accept := AcceptKey(request.Header.Get("Sec-WebSocket-Key"))
		writer := bufio.NewWriter(connection)
		fmt.Fprintf(writer, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n", accept)
		writer.Flush()
		handler(connection, reader)
	}()
	return listener
}

func TestDialOpeningHandshakeAndEcho(t *testing.T) {
	listener := newTestServer(t, func(connection net.Conn, reader *bufio.Reader) {
		fin, opcode, payload := readClientFrame(t, reader)
		writer := bufio.NewWriter(connection)
		writeRawFrame(t, writer, fin, opcode, false, payload)
		writer.Flush()
	})
	defer listener.Close()

	target, _ := urls.Parse("ws://" + listener.Addr().String() + "/")
	connection, err := Dial(context.Background(), target, http1.Header{}, nil, nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer connection.Close()

	if err := connection.WriteMessage(OpcodeText, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	opcode, payload, err := connection.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if opcode != OpcodeText || string(payload) != "hello" {
		t.Errorf("echoed %d %q", opcode, payload)
	}
}

func TestDialRejectsNonUpgrade(t *testing.T) {
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
		bufio.NewReader(connection).ReadString('\n')
		io.WriteString(connection, "HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n")
	}()
	target, _ := urls.Parse("ws://" + listener.Addr().String() + "/")
	if _, err := Dial(context.Background(), target, http1.Header{}, nil, nil); !errors.Is(err, ErrNotUpgrade) {
		t.Fatalf("Dial error = %v, want ErrNotUpgrade", err)
	}
}

func TestReadFragmentedText(t *testing.T) {
	connection, server := newConn(t)
	defer connection.Close()
	defer server.Close()
	go func() {
		writeRawFrame(t, server, false, OpcodeText, false, []byte("hel"))
		writeRawFrame(t, server, true, OpcodeContinuation, false, []byte("lo"))
	}()
	opcode, payload, err := connection.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if opcode != OpcodeText || string(payload) != "hello" {
		t.Errorf("message = %d %q", opcode, payload)
	}
}

func TestReadPingPong(t *testing.T) {
	connection, server := newConn(t)
	defer connection.Close()
	defer server.Close()
	go func() {
		writeRawFrame(t, server, true, OpcodePing, false, []byte("hi"))
		writeRawFrame(t, server, true, OpcodeText, false, []byte("after"))
	}()
	// ReadMessage must answer the ping and then return the text frame.
	opcode, payload, err := connection.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if opcode != OpcodeText || string(payload) != "after" {
		t.Errorf("message = %d %q", opcode, payload)
	}
	// The server should observe a pong.
	fin, pongOpcode, pongPayload := readClientFrame(t, bufio.NewReader(server))
	if !fin || pongOpcode != OpcodePong || string(pongPayload) != "hi" {
		t.Errorf("pong = %v %d %q", fin, pongOpcode, pongPayload)
	}
}

func TestRejectMaskedServerFrame(t *testing.T) {
	connection, server := newConn(t)
	defer connection.Close()
	defer server.Close()
	go writeRawFrame(t, server, true, OpcodeText, true, []byte("bad"))
	if _, _, err := connection.ReadMessage(); !errors.Is(err, ErrProtocol) {
		t.Fatalf("error = %v, want ErrProtocol", err)
	}
}

func TestRejectReservedBits(t *testing.T) {
	connection, server := newConn(t)
	defer connection.Close()
	defer server.Close()
	go func() {
		// RSV1 set, masked bit clear, text opcode, empty payload.
		server.Write([]byte{0xC1, 0x00})
	}()
	if _, _, err := connection.ReadMessage(); !errors.Is(err, ErrProtocol) {
		t.Fatalf("error = %v, want ErrProtocol", err)
	}
}

func TestRejectLongControlFrame(t *testing.T) {
	connection, server := newConn(t)
	defer connection.Close()
	defer server.Close()
	go func() {
		payload := make([]byte, 126)
		writeRawFrame(t, server, true, OpcodePing, false, payload)
	}()
	if _, _, err := connection.ReadMessage(); !errors.Is(err, ErrProtocol) {
		t.Fatalf("error = %v, want ErrProtocol", err)
	}
}

func TestRejectFragmentedControlFrame(t *testing.T) {
	connection, server := newConn(t)
	defer connection.Close()
	defer server.Close()
	go writeRawFrame(t, server, false, OpcodePing, false, []byte("x"))
	if _, _, err := connection.ReadMessage(); !errors.Is(err, ErrProtocol) {
		t.Fatalf("error = %v, want ErrProtocol", err)
	}
}

func TestRejectInvalidUTF8Text(t *testing.T) {
	connection, server := newConn(t)
	defer connection.Close()
	defer server.Close()
	go writeRawFrame(t, server, true, OpcodeText, false, []byte{0xff, 0xfe})
	if _, _, err := connection.ReadMessage(); !errors.Is(err, ErrProtocol) {
		t.Fatalf("error = %v, want ErrProtocol", err)
	}
}

func TestCloseFrame(t *testing.T) {
	connection, server := newConn(t)
	defer connection.Close()
	defer server.Close()
	go func() {
		payload := []byte{0x03, 0xE8} // 1000
		writeRawFrame(t, server, true, OpcodeClose, false, payload)
	}()
	if _, _, err := connection.ReadMessage(); !errors.Is(err, ErrClosed) {
		t.Fatalf("error = %v, want ErrClosed", err)
	}
}

func TestRejectInvalidCloseCode(t *testing.T) {
	connection, server := newConn(t)
	defer connection.Close()
	defer server.Close()
	go func() {
		payload := []byte{0x03, 0xED} // 1005, not allowed on the wire
		writeRawFrame(t, server, true, OpcodeClose, false, payload)
	}()
	if _, _, err := connection.ReadMessage(); !errors.Is(err, ErrProtocol) {
		t.Fatalf("error = %v, want ErrProtocol", err)
	}
}

func TestWriteControlTooLong(t *testing.T) {
	connection, _ := newConn(t)
	defer connection.Close()
	if err := connection.WriteControl(OpcodePing, make([]byte, 126)); !errors.Is(err, ErrProtocol) {
		t.Fatalf("error = %v, want ErrProtocol", err)
	}
}

// newConn returns a client Conn and the server end of a real TCP connection,
// so writes buffer instead of deadlocking as they would with net.Pipe.
func newConn(t *testing.T) (*Conn, net.Conn) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	accepted := make(chan net.Conn, 1)
	go func() {
		connection, err := listener.Accept()
		if err == nil {
			accepted <- connection
		}
	}()
	client, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	server := <-accepted
	listener.Close()
	connection := &Conn{
		connection: client,
		reader:     bufio.NewReader(client),
		writer:     bufio.NewWriter(client),
		maxPayload: 1 << 20,
	}
	return connection, server
}

func readClientFrame(t *testing.T, reader *bufio.Reader) (bool, Opcode, []byte) {
	t.Helper()
	header := make([]byte, 2)
	if _, err := io.ReadFull(reader, header); err != nil {
		t.Errorf("read client frame header: %v", err)
		return false, 0, nil
	}
	fin := header[0]&0x80 != 0
	opcode := Opcode(header[0] & 0x0F)
	if header[1]&0x80 == 0 {
		t.Error("client frames must be masked")
		return false, 0, nil
	}
	length := int64(header[1] & 0x7F)
	switch length {
	case 126:
		buffer := make([]byte, 2)
		io.ReadFull(reader, buffer)
		length = int64(binary.BigEndian.Uint16(buffer))
	case 127:
		buffer := make([]byte, 8)
		io.ReadFull(reader, buffer)
		length = int64(binary.BigEndian.Uint64(buffer))
	}
	maskKey := make([]byte, 4)
	io.ReadFull(reader, maskKey)
	payload := make([]byte, length)
	io.ReadFull(reader, payload)
	for index := range payload {
		payload[index] ^= maskKey[index%4]
	}
	return fin, opcode, payload
}

func writeRawFrame(t *testing.T, writer io.Writer, fin bool, opcode Opcode, masked bool, payload []byte) {
	t.Helper()
	first := byte(opcode)
	if fin {
		first |= 0x80
	}
	header := []byte{first}
	maskBit := byte(0)
	if masked {
		maskBit = 0x80
	}
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
	if masked {
		maskKey := []byte{1, 2, 3, 4}
		header = append(header, maskKey...)
		maskedPayload := make([]byte, length)
		for index := range payload {
			maskedPayload[index] = payload[index] ^ maskKey[index%4]
		}
		payload = maskedPayload
	}
	writer.Write(header)
	writer.Write(payload)
}
