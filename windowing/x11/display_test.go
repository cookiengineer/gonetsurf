package x11

import (
	"encoding/binary"
	"image"
	"os"
	"testing"

	"gonetsurf/rendering"
)

func TestOpenDisplay(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("DISPLAY is not set")
	}
	display := New(Options{Width: 64, Height: 48, Title: "gonetsurf x11 test"})
	window, err := display.Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	surface := window.Surface()
	surface.Clear(rendering.NewColor(255, 0, 0))
	if err := window.Present(image.Rect(0, 0, 64, 48)); err != nil {
		t.Fatalf("Present: %v", err)
	}
	if width, height := window.Size(); width != 64 || height != 48 {
		t.Errorf("Size = %dx%d", width, height)
	}
	if err := display.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestOpenInvalidDisplay(t *testing.T) {
	display := New(Options{Display: ":99", Width: 32, Height: 32})
	if _, err := display.Open(); err == nil {
		display.Close()
		t.Fatal("expected an error for a missing display")
	}
}

func TestConfigureNotifyResize(t *testing.T) {
	window := &Window{
		display: &Display{atoms: map[string]uint32{}},
		id:      1,
		width:   64,
		height:  48,
		surface: rendering.NewSurface(64, 48),
	}
	calledWidth := 0
	calledHeight := 0
	window.SetResizeHandler(func(width, height int) {
		calledWidth = width
		calledHeight = height
	})
	packet := make([]byte, 32)
	packet[0] = eventConfigureNotify
	binary.LittleEndian.PutUint16(packet[20:22], 100)
	binary.LittleEndian.PutUint16(packet[22:24], 80)
	if _, err := window.handleEvent(packet); err != nil {
		t.Fatal(err)
	}
	if calledWidth != 100 || calledHeight != 80 {
		t.Errorf("resize handler = %dx%d", calledWidth, calledHeight)
	}
	if width, height := window.Size(); width != 100 || height != 80 {
		t.Errorf("window size = %dx%d", width, height)
	}
	if window.Surface().Width != 100 || window.Surface().Height != 80 {
		t.Errorf("surface size = %dx%d", window.Surface().Width, window.Surface().Height)
	}
}

func TestClientMessageClose(t *testing.T) {
	window := &Window{
		display: &Display{atoms: map[string]uint32{"WM_PROTOCOLS": 10, "WM_DELETE_WINDOW": 11}},
		id:      1,
		width:   10,
		height:  10,
		surface: rendering.NewSurface(10, 10),
	}
	packet := make([]byte, 32)
	packet[0] = eventClientMessage
	binary.LittleEndian.PutUint32(packet[8:12], 10)
	binary.LittleEndian.PutUint32(packet[16:20], 11)
	closeRequested, err := window.handleEvent(packet)
	if err != nil {
		t.Fatal(err)
	}
	if !closeRequested {
		t.Error("expected the delete message to close the window")
	}
}
