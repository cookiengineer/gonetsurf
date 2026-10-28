package rendering

import (
	"image"
	"testing"
)

func BenchmarkFillRect(b *testing.B) {
	surface := NewSurface(256, 256)
	color := NewColor(200, 30, 30)
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		surface.FillRect(Rect{X: 10, Y: 10, Width: 200, Height: 100}, color)
	}
}

func BenchmarkFillPathCircle(b *testing.B) {
	surface := NewSurface(256, 256)
	path := NewPath()
	path.AddCircle(128, 128, 80)
	color := NewColor(30, 120, 200)
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		surface.FillPath(path, NonZeroFill, color)
	}
}

func BenchmarkStrokePath(b *testing.B) {
	surface := NewSurface(256, 256)
	path := NewPath()
	path.MoveTo(20, 20)
	path.CubicTo(80, 220, 180, 20, 240, 220)
	style := DefaultStrokeStyle(6)
	color := NewColor(20, 20, 20)
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		surface.StrokePath(path, style, color)
	}
}

func BenchmarkBlit(b *testing.B) {
	destination := NewSurface(256, 256)
	source := NewSurface(128, 128)
	source.Clear(Color{Red: 100, Green: 150, Blue: 200, Alpha: 128})
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		destination.Blit(10, 10, source, SourceOver)
	}
}

func BenchmarkCompositorFrame(b *testing.B) {
	compositor := NewCompositor(800, 600)
	background := NewSurface(800, 600)
	background.Clear(NewColor(255, 255, 255))
	foreground := NewSurface(200, 100)
	foreground.Clear(Color{Red: 20, Green: 40, Blue: 60, Alpha: 200})
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		compositor.BeginFrame()
		compositor.AddLayer(0, 0, background, 1)
		compositor.AddLayer(300, 250, foreground, 1)
		compositor.Composite()
	}
}

func BenchmarkTileComposite(b *testing.B) {
	grid := NewTileGrid(800, 600, 64)
	tile := grid.TileAt(100, 100)
	tile.Clear(NewColor(10, 20, 30))
	destination := NewSurface(800, 600)
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		grid.Invalidate(image.Rect(100, 100, 120, 120))
		grid.Composite(destination)
	}
}
