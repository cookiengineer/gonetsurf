package x11

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestParseDisplayName(t *testing.T) {
	tests := []struct {
		input  string
		host   string
		number int
		screen int
		isUnix bool
	}{
		{":0", "", 0, 0, true},
		{"unix:1.2", "unix", 1, 2, true},
		{"localhost:10.0", "localhost", 10, 0, true},
		{"example.com:0", "example.com", 0, 0, false},
	}
	for _, test := range tests {
		parsed, err := parseDisplayName(test.input)
		if err != nil {
			t.Fatalf("parseDisplayName(%q): %v", test.input, err)
		}
		if parsed.host != test.host || parsed.number != test.number || parsed.screen != test.screen || parsed.isUnix != test.isUnix {
			t.Errorf("parseDisplayName(%q) = %#v", test.input, parsed)
		}
	}
	if _, err := parseDisplayName(""); err == nil {
		t.Error("expected an error for an empty display name")
	}
	if _, err := parseDisplayName("host"); err == nil {
		t.Error("expected an error for a display name without a colon")
	}
}

func writeAuthorityEntry(buffer *bytes.Buffer, family uint16, address, number, name, data string) {
	binary.Write(buffer, binary.BigEndian, family)
	fields := [][]byte{[]byte(address), []byte(number), []byte(name), []byte(data)}
	for _, field := range fields {
		binary.Write(buffer, binary.BigEndian, uint16(len(field)))
		buffer.Write(field)
	}
}

func TestAuthorityLookup(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "Xauthority")
	hostname, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}
	var buffer bytes.Buffer
	writeAuthorityEntry(&buffer, familyLocal, hostname, "0", magicCookieName, "secret")
	writeAuthorityEntry(&buffer, familyInternet, "example.com", "1", magicCookieName, "remote")
	if err := os.WriteFile(path, buffer.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XAUTHORITY", path)

	name, data := authorityFor(displayName{number: 0, isUnix: true}, hostname)
	if string(name) != magicCookieName || string(data) != "secret" {
		t.Errorf("local authority = (%q, %q)", name, data)
	}
	name, data = authorityFor(displayName{number: 1, host: "example.com", isUnix: false}, hostname)
	if string(name) != magicCookieName || string(data) != "remote" {
		t.Errorf("remote authority = (%q, %q)", name, data)
	}
	name, data = authorityFor(displayName{number: 9, isUnix: true}, hostname)
	if name != nil || data != nil {
		t.Errorf("unexpected authority for unknown display: (%q, %q)", name, data)
	}
}

func buildSetupResponse() []byte {
	var body []byte
	appendUint32 := func(value uint32) {
		var bytes [4]byte
		binary.LittleEndian.PutUint32(bytes[:], value)
		body = append(body, bytes[:]...)
	}
	appendUint16 := func(value uint16) {
		var bytes [2]byte
		binary.LittleEndian.PutUint16(bytes[:], value)
		body = append(body, bytes[:]...)
	}
	appendUint32(12101024)
	appendUint32(0x00400000)
	appendUint32(0x001fffff)
	appendUint32(0)
	appendUint16(0)
	appendUint16(65535)
	body = append(body, 1, 1, 0, 0, 32, 32, 8, 255)
	body = append(body, 0, 0, 0, 0)
	body = append(body,
		24, 32, 32, 0, 0, 0, 0, 0,
	)
	appendUint32(0x123)
	appendUint32(0x20)
	appendUint32(0xffffff)
	appendUint32(0)
	appendUint32(0)
	appendUint16(800)
	appendUint16(600)
	appendUint16(300)
	appendUint16(200)
	appendUint16(1)
	appendUint16(1)
	appendUint32(0x21)
	body = append(body, 0, 0, 24, 1)
	body = append(body, 24, 0)
	appendUint16(1)
	appendUint32(0)
	appendUint32(0x21)
	body = append(body, 4, 8)
	appendUint16(256)
	appendUint32(0xff0000)
	appendUint32(0x00ff00)
	appendUint32(0x0000ff)
	appendUint32(0)

	header := make([]byte, 8)
	header[0] = 1
	binary.LittleEndian.PutUint16(header[2:4], 11)
	binary.LittleEndian.PutUint16(header[6:8], uint16(len(body)/4))
	return append(header, body...)
}

func TestReadSetup(t *testing.T) {
	info, err := readSetup(bytes.NewReader(buildSetupResponse()))
	if err != nil {
		t.Fatal(err)
	}
	if info.resourceBase != 0x00400000 || info.resourceMask != 0x001fffff {
		t.Errorf("resource ids = %#x/%#x", info.resourceBase, info.resourceMask)
	}
	if len(info.screens) != 1 || info.screens[0].root != 0x123 || info.screens[0].rootVisual != 0x21 {
		t.Fatalf("screens = %#v", info.screens)
	}
	visual, bitsPerPixel, err := info.selectVisual(info.screens[0])
	if err != nil {
		t.Fatal(err)
	}
	if visual.id != 0x21 || bitsPerPixel != 32 {
		t.Errorf("visual = %#v, bpp = %d", visual, bitsPerPixel)
	}
}

func TestRequestLengths(t *testing.T) {
	create := createWindowRequest(24, 0x400001, 0x123, 800, 600, 0x21, 0x2, []uint32{0xffffff})
	if len(create)%4 != 0 || int(binary.LittleEndian.Uint16(create[2:4])) != len(create)/4 {
		t.Errorf("create request length = %d (field %d)", len(create), binary.LittleEndian.Uint16(create[2:4]))
	}
	property := changePropertyRequest(0x400001, 100, 31, 8, []byte("hello"))
	if int(binary.LittleEndian.Uint16(property[2:4])) != len(property)/4 {
		t.Errorf("property request length mismatch")
	}
	image := putImageRequest(0x400001, 0x400002, 2, 1, 0, 0, 24, make([]byte, 8))
	if int(binary.LittleEndian.Uint16(image[2:4])) != len(image)/4 {
		t.Errorf("put image request length mismatch")
	}
}

func TestPackPixel(t *testing.T) {
	masks := [3]uint32{0xff0000, 0x00ff00, 0x0000ff}
	if value := packPixel(0x12, 0x34, 0x56, masks); value != 0x123456 {
		t.Errorf("packPixel = %#x, want 0x123456", value)
	}
}

func TestUnsupportedVisual(t *testing.T) {
	info := &setupInfo{
		formats: []pixmapFormat{{depth: 24, bitsPerPixel: 24}},
	}
	screen := screenInfo{
		rootVisual: 1,
		rootDepth:  24,
		visuals:    []visualInfo{{id: 1, class: visualClassTrueColor, depth: 24}},
	}
	if _, _, err := info.selectVisual(screen); err == nil {
		t.Fatal("expected an unsupported visual error")
	}
}
