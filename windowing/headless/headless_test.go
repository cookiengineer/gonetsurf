package headless

import (
	"context"
	"image"
	"testing"

	"gonetsurf/rendering"
	"gonetsurf/windowing"
)

func TestDisplayLifecycle(t *testing.T) {
	display := New(32, 24, "headless test")
	window, err := display.Open()
	if err != nil {
		t.Fatal(err)
	}
	window.Surface().Clear(rendering.NewColor(0, 128, 0))
	if err := window.Present(image.Rect(0, 0, 32, 24)); err != nil {
		t.Fatal(err)
	}
	if width, height := window.Size(); width != 32 || height != 24 {
		t.Errorf("Size = %dx%d", width, height)
	}
	window.SetTitle("updated")
	if typed, ok := window.(*Window); !ok || typed.Title() != "updated" {
		t.Errorf("title was not updated")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := display.EventLoop(ctx); err == nil {
		t.Error("expected cancellation error")
	}
	if err := display.Close(); err != nil {
		t.Error(err)
	}
}

func TestInputWindowEventsAndClipboard(t *testing.T) {
	display := New(16, 16, "input")
	opened, err := display.Open()
	if err != nil {
		t.Fatal(err)
	}
	window, ok := opened.(*Window)
	if !ok {
		t.Fatalf("window = %T", opened)
	}
	if _, ok := opened.(windowing.InputWindow); !ok {
		t.Fatal("headless window does not implement InputWindow")
	}
	window.Inject(windowing.Event{Type: windowing.EventKeyPress, Key: "a", Text: "a"})
	select {
	case event := <-window.Events():
		if event.Key != "a" || event.Text != "a" {
			t.Errorf("event = %#v", event)
		}
	default:
		t.Fatal("event was not delivered")
	}
	if err := window.SetClipboard("hello"); err != nil {
		t.Fatal(err)
	}
	if text, err := window.GetClipboard(context.Background()); err != nil || text != "hello" {
		t.Errorf("clipboard = (%q,%v)", text, err)
	}
}
