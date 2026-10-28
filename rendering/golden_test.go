package rendering

import (
	"bytes"
	"flag"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

var updateGolden = flag.Bool("update", false, "rewrite rendering golden images")

func TestRasterGolden(t *testing.T) {
	surface := renderGoldenScene()
	path := filepath.Join("testdata", "golden", "raster.png")

	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		file, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(file, surface.Image()); err != nil {
			file.Close()
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		return
	}

	expectedFile, err := os.Open(path)
	if err != nil {
		t.Fatalf("open golden: %v (run go test ./rendering -run TestRasterGolden -update)", err)
	}
	defer expectedFile.Close()
	expected, err := png.Decode(expectedFile)
	if err != nil {
		t.Fatalf("decode golden: %v", err)
	}
	actual := surface.Image()
	if expected.Bounds() != actual.Bounds() {
		t.Fatalf("golden bounds = %v, want %v", expected.Bounds(), actual.Bounds())
	}
	expectedBytes := imageBytes(expected)
	actualBytes := imageBytes(actual)
	if !bytes.Equal(expectedBytes, actualBytes) {
		firstDifference := -1
		for index := range expectedBytes {
			if expectedBytes[index] != actualBytes[index] {
				firstDifference = index
				break
			}
		}
		t.Fatalf("golden mismatch at byte %d", firstDifference)
	}
}

func renderGoldenScene() *Surface {
	surface := NewSurface(64, 64)
	surface.Clear(NewColor(255, 255, 255))

	surface.FillRect(Rect{X: 4, Y: 4, Width: 20, Height: 14}, NewColor(220, 40, 40))

	circle := NewPath()
	circle.AddCircle(44, 16, 10)
	surface.FillPath(circle, NonZeroFill, NewColor(40, 120, 220))

	triangle := NewPath()
	triangle.MoveTo(6, 58)
	triangle.LineTo(30, 28)
	triangle.LineTo(54, 58)
	triangle.Close()
	surface.FillPath(triangle, NonZeroFill, NewColor(40, 180, 90))

	line := NewPath()
	line.MoveTo(4, 50)
	line.LineTo(60, 30)
	surface.StrokePath(line, StrokeStyle{Width: 3, Cap: RoundCap, Join: RoundJoin}, NewColor(20, 20, 20))

	return surface
}

func imageBytes(source image.Image) []byte {
	bounds := source.Bounds()
	converted := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(converted, converted.Bounds(), source, bounds.Min, draw.Src)
	return converted.Pix
}
