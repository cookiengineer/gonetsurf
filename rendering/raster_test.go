package rendering

import (
	"math"
	"testing"
)

func TestRasterizeFillRect(t *testing.T) {
	path := NewPath()
	path.AddRect(Rect{X: 2, Y: 2, Width: 4, Height: 3})
	mask := RasterizeFill(path, 10, 10, NonZeroFill)

	if coverage := mask.At(4, 3); coverage != 1 {
		t.Errorf("interior coverage = %v, want 1", coverage)
	}
	if coverage := mask.At(1, 3); coverage != 0 {
		t.Errorf("left exterior coverage = %v, want 0", coverage)
	}
	if coverage := mask.At(6, 3); coverage != 0 {
		t.Errorf("right exterior coverage = %v, want 0", coverage)
	}
	if coverage := mask.At(4, 5); coverage != 0 {
		t.Errorf("bottom exterior coverage = %v, want 0", coverage)
	}
}

func TestRasterizeEvenOddHole(t *testing.T) {
	path := NewPath()
	path.AddRect(Rect{X: 0, Y: 0, Width: 20, Height: 20})
	path.AddRect(Rect{X: 5, Y: 5, Width: 10, Height: 10})
	mask := RasterizeFill(path, 20, 20, EvenOddFill)

	if coverage := mask.At(2, 2); coverage != 1 {
		t.Errorf("outer ring coverage = %v, want 1", coverage)
	}
	if coverage := mask.At(10, 10); coverage != 0 {
		t.Errorf("hole coverage = %v, want 0", coverage)
	}
}

func TestRasterizeNonZeroHole(t *testing.T) {
	path := NewPath()
	path.AddRect(Rect{X: 0, Y: 0, Width: 20, Height: 20})
	path.MoveTo(5, 5)
	path.LineTo(5, 15)
	path.LineTo(15, 15)
	path.LineTo(15, 5)
	path.Close()
	mask := RasterizeFill(path, 20, 20, NonZeroFill)

	if coverage := mask.At(2, 2); coverage != 1 {
		t.Errorf("outer ring coverage = %v, want 1", coverage)
	}
	if coverage := mask.At(10, 10); coverage != 0 {
		t.Errorf("hole coverage = %v, want 0", coverage)
	}
}

func TestRasterizeAntiAliasedDiagonal(t *testing.T) {
	path := NewPath()
	path.MoveTo(0, 0)
	path.LineTo(10, 10)
	path.LineTo(0, 10)
	path.Close()
	mask := RasterizeFill(path, 10, 10, NonZeroFill)

	partial := 0
	for yPos := 0; yPos < 10; yPos++ {
		for xPos := 0; xPos < 10; xPos++ {
			coverage := mask.At(xPos, yPos)
			if coverage > 0 && coverage < 1 {
				partial++
			}
		}
	}
	if partial == 0 {
		t.Fatal("expected anti-aliased partial coverage on the diagonal")
	}
	if coverage := mask.At(1, 8); coverage != 1 {
		t.Errorf("interior coverage = %v, want 1", coverage)
	}
}

func TestRasterizeEmptyPath(t *testing.T) {
	mask := RasterizeFill(NewPath(), 4, 4, NonZeroFill)
	for _, coverage := range mask.Coverage {
		if coverage != 0 {
			t.Fatalf("empty path coverage = %v", coverage)
		}
	}
}

func TestMaskUnion(t *testing.T) {
	first := NewMask(1, 1)
	first.Coverage[0] = 0.5
	second := NewMask(1, 1)
	second.Coverage[0] = 0.5
	first.Union(second)
	if math.Abs(first.Coverage[0]-0.75) > 1e-9 {
		t.Errorf("union coverage = %v, want 0.75", first.Coverage[0])
	}
}
