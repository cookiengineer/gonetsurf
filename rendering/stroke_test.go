package rendering

import "testing"

func TestStrokeLineButtCap(t *testing.T) {
	path := NewPath()
	path.MoveTo(10, 20)
	path.LineTo(30, 20)
	mask := RasterizeStroke(path, 40, 40, StrokeStyle{Width: 4, Cap: ButtCap, Join: RoundJoin})

	if coverage := mask.At(20, 20); coverage != 1 {
		t.Errorf("line body coverage = %v, want 1", coverage)
	}
	if coverage := mask.At(20, 17); coverage != 0 {
		t.Errorf("outside line coverage = %v, want 0", coverage)
	}
	if coverage := mask.At(31, 20); coverage != 0 {
		t.Errorf("beyond butt cap coverage = %v, want 0", coverage)
	}
}

func TestStrokeRoundCap(t *testing.T) {
	path := NewPath()
	path.MoveTo(10, 20)
	path.LineTo(30, 20)
	mask := RasterizeStroke(path, 40, 40, StrokeStyle{Width: 4, Cap: RoundCap, Join: RoundJoin})

	if coverage := mask.At(31, 20); coverage == 0 {
		t.Error("expected round cap to extend past the endpoint")
	}
	if coverage := mask.At(33, 20); coverage != 0 {
		t.Errorf("round cap extends too far: %v", coverage)
	}
}

func TestStrokeSquareCap(t *testing.T) {
	path := NewPath()
	path.MoveTo(10, 20)
	path.LineTo(30, 20)
	mask := RasterizeStroke(path, 40, 40, StrokeStyle{Width: 4, Cap: SquareCap, Join: RoundJoin})

	if coverage := mask.At(31, 20); coverage == 0 {
		t.Error("expected square cap to extend past the endpoint")
	}
	if coverage := mask.At(33, 20); coverage != 0 {
		t.Errorf("square cap extends too far: %v", coverage)
	}
}

func TestStrokeClosedCircle(t *testing.T) {
	path := NewPath()
	path.AddCircle(20, 20, 10)
	mask := RasterizeStroke(path, 40, 40, StrokeStyle{Width: 2, Cap: ButtCap, Join: RoundJoin})

	if coverage := mask.At(20, 10); coverage == 0 {
		t.Error("expected top of circle stroke to be covered")
	}
	if coverage := mask.At(20, 20); coverage != 0 {
		t.Errorf("circle interior coverage = %v, want 0", coverage)
	}
}

func TestStrokeJoinShapes(t *testing.T) {
	for _, join := range []LineJoin{MiterJoin, BevelJoin, RoundJoin} {
		path := NewPath()
		path.MoveTo(5, 25)
		path.LineTo(20, 25)
		path.LineTo(20, 5)
		mask := RasterizeStroke(path, 30, 30, StrokeStyle{Width: 6, Cap: ButtCap, Join: join})
		if coverage := mask.At(20, 25); coverage == 0 {
			t.Errorf("join %d did not cover the corner", join)
		}
	}
}
