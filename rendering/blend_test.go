package rendering

import "testing"

func solidSurface(width, height int, color Color) *Surface {
	surface := NewSurface(width, height)
	surface.Clear(color)
	return surface
}

func TestBlitSourceOver(t *testing.T) {
	destination := solidSurface(1, 1, NewColor(0, 0, 255))
	source := solidSurface(1, 1, Color{Red: 255, Alpha: 128})
	destination.Blit(0, 0, source, SourceOver)

	red := destination.Pix[0]
	blue := destination.Pix[2]
	alpha := destination.Pix[3]
	if red < 127 || red > 128 {
		t.Errorf("red = %d, want ~128", red)
	}
	if blue < 126 || blue > 127 {
		t.Errorf("blue = %d, want ~127", blue)
	}
	if alpha != 255 {
		t.Errorf("alpha = %d, want 255", alpha)
	}
}

func TestBlitSourceIn(t *testing.T) {
	destination := solidSurface(1, 1, Color{Blue: 255, Alpha: 128})
	source := solidSurface(1, 1, NewColor(255, 0, 0))
	destination.Blit(0, 0, source, SourceIn)

	red := destination.Pix[0]
	blue := destination.Pix[2]
	alpha := destination.Pix[3]
	if red < 127 || red > 128 {
		t.Errorf("red = %d, want ~128", red)
	}
	if blue != 0 {
		t.Errorf("blue = %d, want 0", blue)
	}
	if alpha < 127 || alpha > 128 {
		t.Errorf("alpha = %d, want ~128", alpha)
	}
}

func TestBlitSourceOut(t *testing.T) {
	destination := solidSurface(1, 1, Color{Blue: 255, Alpha: 64})
	source := solidSurface(1, 1, NewColor(255, 0, 0))
	destination.Blit(0, 0, source, SourceOut)

	red := destination.Pix[0]
	alpha := destination.Pix[3]
	if red < 191 || red > 192 {
		t.Errorf("red = %d, want ~191", red)
	}
	if alpha < 191 || alpha > 192 {
		t.Errorf("alpha = %d, want ~191", alpha)
	}
}

func TestBlitClearAndCopy(t *testing.T) {
	destination := solidSurface(2, 2, NewColor(1, 2, 3))
	source := solidSurface(2, 2, NewColor(4, 5, 6))

	destination.Blit(1, 1, source, Clear)
	clearedOffset := destination.Stride + 4
	if destination.Pix[clearedOffset+3] != 0 {
		t.Errorf("cleared alpha = %d, want 0", destination.Pix[clearedOffset+3])
	}

	destination = solidSurface(2, 2, NewColor(1, 2, 3))
	destination.Blit(1, 1, source, Copy)
	offset := destination.Stride + 4
	if destination.Pix[offset] != 4 || destination.Pix[offset+1] != 5 || destination.Pix[offset+2] != 6 {
		t.Errorf("copied pixel = %v", destination.Pix[offset:offset+4])
	}
}

func TestBlitRespectsClip(t *testing.T) {
	destination := solidSurface(4, 4, NewColor(0, 0, 0))
	source := solidSurface(4, 4, NewColor(255, 255, 255))
	destination.SetClip(Rect{X: 1, Y: 1, Width: 1, Height: 1})
	destination.Blit(0, 0, source, Copy)

	if destination.Pix[0] != 0 {
		t.Errorf("unclipped pixel changed: %d", destination.Pix[0])
	}
	offset := destination.Stride + 4
	if destination.Pix[offset] != 255 {
		t.Errorf("clipped pixel red = %d, want 255", destination.Pix[offset])
	}
}
