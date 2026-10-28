package x11

import (
	"encoding/binary"
	"testing"

	"gonetsurf/windowing"
)

func TestKeysymValues(t *testing.T) {
	tests := []struct {
		keysym uint32
		key    string
		text   string
	}{
		{'a', "a", "a"},
		{' ', " ", " "},
		{0xff0d, "Enter", ""},
		{0xff51, "Left", ""},
		{0xffbe, "F1", ""},
		{0x010000e9, "é", "é"},
	}
	for _, test := range tests {
		key, text := keysymValues(test.keysym)
		if key != test.key || text != test.text {
			t.Errorf("keysymValues(%#x) = (%q,%q), want (%q,%q)", test.keysym, key, text, test.key, test.text)
		}
	}
}

func TestModifiersFromState(t *testing.T) {
	modifiers := modifiersFromState(1 | 4 | 8 | 64)
	if !modifiers.Shift || !modifiers.Control || !modifiers.Alt || !modifiers.Super {
		t.Errorf("modifiers = %#v", modifiers)
	}
}

func TestHandleKeyAndPointerEvents(t *testing.T) {
	display := &Display{keymap: map[byte][]uint32{38: {'a', 'A'}}}
	window := &Window{display: display, events: make(chan windowing.Event, 4)}

	packet := make([]byte, 32)
	packet[0] = eventKeyPressEvent
	packet[1] = 38
	window.handleKeyEvent(packet, true)
	event := <-window.events
	if event.Type != windowing.EventKeyPress || event.Key != "a" || event.Text != "a" || event.Modifiers.Shift {
		t.Errorf("key event = %#v", event)
	}

	binary.LittleEndian.PutUint16(packet[28:30], 1)
	window.handleKeyEvent(packet, false)
	event = <-window.events
	if event.Type != windowing.EventKeyRelease || event.Key != "A" || !event.Modifiers.Shift {
		t.Errorf("shifted key event = %#v", event)
	}

	button := make([]byte, 32)
	button[0] = eventButtonPressEvent
	button[1] = 1
	binary.LittleEndian.PutUint16(button[24:26], 15)
	binary.LittleEndian.PutUint16(button[26:28], 25)
	window.handlePointerEvent(button, windowing.EventMousePress)
	pointer := <-window.events
	if pointer.Type != windowing.EventMousePress || pointer.Button != 1 || pointer.X != 15 || pointer.Y != 25 {
		t.Errorf("pointer event = %#v", pointer)
	}
}
