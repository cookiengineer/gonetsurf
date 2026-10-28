package rendering

import (
	"image"
	"testing"
)

func TestDamageMergesAdjacentRects(t *testing.T) {
	var damage Damage
	damage.Add(image.Rect(0, 0, 10, 10))
	damage.Add(image.Rect(10, 0, 20, 10))
	rects := damage.Rects()
	if len(rects) != 1 || rects[0] != image.Rect(0, 0, 20, 10) {
		t.Fatalf("damage = %#v", rects)
	}
	if damage.Bounds() != image.Rect(0, 0, 20, 10) {
		t.Errorf("bounds = %v", damage.Bounds())
	}
	damage.Clear()
	if !damage.Empty() {
		t.Error("damage was not cleared")
	}
}

func TestCompositorLayersAndDamage(t *testing.T) {
	compositor := NewCompositor(40, 30)
	background := NewSurface(40, 30)
	background.Clear(NewColor(0, 0, 255))
	foreground := NewSurface(10, 10)
	foreground.Clear(NewColor(255, 0, 0))

	compositor.BeginFrame()
	compositor.AddLayer(0, 0, background, 1)
	compositor.AddLayer(5, 5, foreground, 1)
	damage := compositor.Composite()
	if damage != image.Rect(0, 0, 40, 30) {
		t.Errorf("damage = %v", damage)
	}
	if pixel := pixelAt(compositor.Surface, 8, 8); pixel.Red < 200 {
		t.Errorf("foreground pixel = %#v", pixel)
	}
	if pixel := pixelAt(compositor.Surface, 30, 20); pixel.Blue < 200 {
		t.Errorf("background pixel = %#v", pixel)
	}

	compositor.BeginFrame()
	compositor.AddLayer(5, 5, foreground, 1)
	damage = compositor.Composite()
	if damage != image.Rect(5, 5, 15, 15) {
		t.Errorf("second damage = %v", damage)
	}
	if pixel := pixelAt(compositor.Surface, 8, 8); pixel.Red < 200 {
		t.Errorf("second foreground pixel = %#v", pixel)
	}
	if pixel := pixelAt(compositor.Surface, 30, 20); pixel.Blue < 200 {
		t.Errorf("outside damage was cleared: %#v", pixel)
	}
}

func TestCompositorOpacity(t *testing.T) {
	compositor := NewCompositor(10, 10)
	background := NewSurface(10, 10)
	background.Clear(NewColor(255, 255, 255))
	foreground := NewSurface(10, 10)
	foreground.Clear(NewColor(255, 0, 0))

	compositor.BeginFrame()
	compositor.AddLayer(0, 0, background, 1)
	compositor.AddLayer(0, 0, foreground, 0.5)
	compositor.Composite()
	pixel := pixelAt(compositor.Surface, 5, 5)
	if pixel.Green < 120 || pixel.Green > 140 || pixel.Blue < 120 || pixel.Blue > 140 {
		t.Errorf("opacity pixel = %#v", pixel)
	}
}

func TestTileGridComposite(t *testing.T) {
	grid := NewTileGrid(64, 64, 32)
	if grid.DirtyCount() != 0 {
		t.Fatalf("initial dirty = %d", grid.DirtyCount())
	}
	tile := grid.TileAt(40, 40)
	tile.Clear(NewColor(0, 200, 0))
	if grid.DirtyCount() != 1 {
		t.Fatalf("dirty after tile creation = %d", grid.DirtyCount())
	}
	destination := NewSurface(64, 64)
	if count := grid.Composite(destination); count != 1 {
		t.Fatalf("composited tiles = %d", count)
	}
	if pixel := pixelAt(destination, 40, 40); pixel.Green < 150 {
		t.Errorf("tile pixel = %#v", pixel)
	}
	if grid.DirtyCount() != 0 {
		t.Errorf("dirty after composite = %d", grid.DirtyCount())
	}
	grid.Invalidate(image.Rect(0, 0, 33, 33))
	if grid.DirtyCount() != 1 {
		t.Errorf("dirty after invalidation = %d", grid.DirtyCount())
	}
	grid.InvalidateAll()
	if grid.DirtyCount() != 1 {
		t.Errorf("dirty after invalidate-all = %d", grid.DirtyCount())
	}
}

func TestCompositorDocumentFrame(t *testing.T) {
	source := `<!doctype html><html><head><style>
html, body { margin:0; padding:0 }
.box { width:40px; height:30px; background:#3366cc; border:2px solid #000000 }
</style></head><body><div class="box"></div></body></html>`
	page := renderTestPage(t, source, 80, 60)
	compositor := NewCompositor(80, 60)
	compositor.BeginFrame()
	compositor.AddLayer(0, 0, page, 1)
	damage := compositor.Composite()
	if damage != image.Rect(0, 0, 80, 60) {
		t.Errorf("damage = %v", damage)
	}
	for index := range page.Pix {
		if page.Pix[index] != compositor.Surface.Pix[index] {
			t.Fatalf("composited frame differs at byte %d", index)
		}
	}
}
