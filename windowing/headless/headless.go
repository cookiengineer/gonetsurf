// Package headless provides an in-memory display backend for tests and
// off-screen rendering. It has no platform dependencies.
package headless

import (
	"context"
	"image"

	"gonetsurf/rendering"
	"gonetsurf/windowing"
)

// Display is an in-memory display.
type Display struct {
	width  int
	height int
	title  string
	window *Window
}

// New returns a headless display with the given viewport size.
func New(width, height int, title string) *Display {
	return &Display{width: width, height: height, title: title}
}

// Open creates the in-memory window.
func (display *Display) Open() (windowing.Window, error) {
	if display.window == nil {
		display.window = &Window{
			width:   display.width,
			height:  display.height,
			title:   display.title,
			surface: rendering.NewSurface(display.width, display.height),
			scale:   1,
		}
	}
	return display.window, nil
}

// EventLoop blocks until the context is cancelled.
func (display *Display) EventLoop(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

// Close releases the window.
func (display *Display) Close() error {
	display.window = nil
	return nil
}

// Window is an in-memory window backed by a rendering surface.
type Window struct {
	width   int
	height  int
	title   string
	surface *rendering.Surface
	scale   float64
}

// Surface returns the window's rendering surface.
func (window *Window) Surface() *rendering.Surface {
	return window.surface
}

// Present accepts a damage rectangle. The surface is already the backing store.
func (window *Window) Present(image.Rectangle) error {
	return nil
}

// SetTitle changes the window title.
func (window *Window) SetTitle(title string) {
	window.title = title
}

// Size returns the current window size.
func (window *Window) Size() (width, height int) {
	return window.width, window.height
}

// Scale returns the window scale factor.
func (window *Window) Scale() float64 {
	return window.scale
}

// Title returns the current window title.
func (window *Window) Title() string {
	return window.title
}
