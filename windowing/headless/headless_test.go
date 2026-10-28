package headless

import (
	"context"
	"image"
	"testing"

	"gonetsurf/rendering"
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
