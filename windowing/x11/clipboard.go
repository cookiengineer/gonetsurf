package x11

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"
)

// SetClipboard takes ownership of the CLIPBOARD selection and publishes text.
func (window *Window) SetClipboard(text string) error {
	window.clipboardMutex.Lock()
	window.clipboardText = text
	window.clipboardOwned = true
	window.clipboardMutex.Unlock()
	selection := window.display.atoms["CLIPBOARD"]
	_, err := window.display.sendRequest(setSelectionOwnerRequest(window.id, selection, 0))
	return err
}

// GetClipboard reads the current CLIPBOARD selection. The event loop must be
// running for text owned by another client.
func (window *Window) GetClipboard(ctx context.Context) (string, error) {
	window.clipboardMutex.Lock()
	if window.clipboardOwned {
		text := window.clipboardText
		window.clipboardMutex.Unlock()
		return text, nil
	}
	window.clipboardMutex.Unlock()

	property := window.display.atoms["_NETSURF_CLIPBOARD"]
	selection := window.display.atoms["CLIPBOARD"]
	target := window.display.atoms["UTF8_STRING"]
	results := make(chan string, 1)
	window.clipboardMutex.Lock()
	window.clipboardPending[property] = results
	window.clipboardMutex.Unlock()

	_, err := window.display.sendRequest(convertSelectionRequest(window.id, selection, target, property, 0))
	if err != nil {
		window.clipboardMutex.Lock()
		delete(window.clipboardPending, property)
		window.clipboardMutex.Unlock()
		return "", err
	}
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case text := <-results:
		return text, nil
	case <-ctx.Done():
		window.clipboardMutex.Lock()
		delete(window.clipboardPending, property)
		window.clipboardMutex.Unlock()
		return "", ctx.Err()
	case <-timer.C:
		window.clipboardMutex.Lock()
		delete(window.clipboardPending, property)
		window.clipboardMutex.Unlock()
		return "", fmt.Errorf("x11: clipboard request timed out")
	}
}

func (window *Window) handleSelectionClear(packet []byte) {
	selection := binary.LittleEndian.Uint32(packet[12:16])
	if selection != window.display.atoms["CLIPBOARD"] {
		return
	}
	window.clipboardMutex.Lock()
	window.clipboardOwned = false
	window.clipboardText = ""
	window.clipboardMutex.Unlock()
}

func (window *Window) handleSelectionRequest(packet []byte) {
	timestamp := binary.LittleEndian.Uint32(packet[4:8])
	requestor := binary.LittleEndian.Uint32(packet[12:16])
	selection := binary.LittleEndian.Uint32(packet[16:20])
	target := binary.LittleEndian.Uint32(packet[20:24])
	property := binary.LittleEndian.Uint32(packet[24:28])
	if selection != window.display.atoms["CLIPBOARD"] {
		window.sendSelectionNotify(timestamp, requestor, selection, target, 0)
		return
	}
	window.clipboardMutex.Lock()
	text := window.clipboardText
	window.clipboardMutex.Unlock()

	propertyOut := property
	switch target {
	case window.display.atoms["TARGETS"]:
		targets := []uint32{
			window.display.atoms["TARGETS"],
			window.display.atoms["UTF8_STRING"],
			window.display.atoms["TEXT"],
		}
		data := make([]byte, len(targets)*4)
		for index, atom := range targets {
			binary.LittleEndian.PutUint32(data[index*4:], atom)
		}
		window.display.sendRequest(changePropertyRequest(requestor, property, 4, 32, data))
	case window.display.atoms["UTF8_STRING"], window.display.atoms["TEXT"]:
		window.display.sendRequest(changePropertyRequest(requestor, property, window.display.atoms["UTF8_STRING"], 8, []byte(text)))
	default:
		propertyOut = 0
	}
	window.sendSelectionNotify(timestamp, requestor, selection, target, propertyOut)
}

func (window *Window) sendSelectionNotify(timestamp, requestor, selection, target, property uint32) {
	event := selectionNotifyEvent(timestamp, requestor, selection, target, property)
	window.display.sendRequest(sendEventRequest(requestor, 0, event))
}

func (window *Window) handleSelectionNotify(packet []byte) {
	property := binary.LittleEndian.Uint32(packet[20:24])
	if property == 0 {
		window.clipboardMutex.Lock()
		for key, channel := range window.clipboardPending {
			delete(window.clipboardPending, key)
			select {
			case channel <- "":
			default:
			}
		}
		window.clipboardMutex.Unlock()
		return
	}
	data, err := window.readProperty(property)
	if err != nil {
		data = nil
	}
	window.display.sendRequest(deletePropertyRequest(window.id, property))
	window.clipboardMutex.Lock()
	channel := window.clipboardPending[property]
	delete(window.clipboardPending, property)
	window.clipboardMutex.Unlock()
	if channel != nil {
		select {
		case channel <- string(data):
		default:
		}
	}
}

func (window *Window) readProperty(property uint32) ([]byte, error) {
	sequence, err := window.display.sendRequest(getPropertyRequest(window.id, property, 0, 0, 0x1fffffff))
	if err != nil {
		return nil, err
	}
	reply, err := window.display.waitForReplyData(sequence)
	if err != nil {
		return nil, err
	}
	format := reply[1]
	valueLength := int(binary.LittleEndian.Uint32(reply[16:20]))
	data := reply[32:]
	if format != 8 {
		return nil, fmt.Errorf("x11: clipboard property format is %d", format)
	}
	if valueLength > len(data) {
		valueLength = len(data)
	}
	return data[:valueLength], nil
}
