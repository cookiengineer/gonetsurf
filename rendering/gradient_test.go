package rendering

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func gradientPage(background string) string {
	return `<!doctype html><html><head><style>
html, body { margin:0; padding:0 }
.box { width:40px; height:20px; background:` + background + ` }
</style></head><body><div class="box"></div></body></html>`
}

func TestLinearGradientToRight(t *testing.T) {
	surface := renderTestPage(t, gradientPage("linear-gradient(to right, #ff0000, #0000ff)"), 40, 20)
	left := pixelAt(surface, 1, 10)
	middle := pixelAt(surface, 20, 10)
	right := pixelAt(surface, 38, 10)
	if left.Red < 200 || left.Blue > 50 {
		t.Errorf("left pixel = %#v, want red", left)
	}
	if right.Blue < 200 || right.Red > 50 {
		t.Errorf("right pixel = %#v, want blue", right)
	}
	if middle.Red < 80 || middle.Red > 180 || middle.Blue < 80 || middle.Blue > 180 {
		t.Errorf("middle pixel = %#v, want mixed", middle)
	}
}

func TestLinearGradientDefaultDirection(t *testing.T) {
	surface := renderTestPage(t, gradientPage("linear-gradient(#ff0000, #0000ff)"), 40, 20)
	top := pixelAt(surface, 20, 1)
	bottom := pixelAt(surface, 20, 18)
	if top.Red < 200 || top.Blue > 50 {
		t.Errorf("top pixel = %#v, want red", top)
	}
	if bottom.Blue < 200 || bottom.Red > 50 {
		t.Errorf("bottom pixel = %#v, want blue", bottom)
	}
}

func TestLinearGradientStops(t *testing.T) {
	surface := renderTestPage(t, gradientPage("linear-gradient(90deg, #ff0000 0%, #00ff00 50%, #0000ff 100%)"), 40, 20)
	if pixel := pixelAt(surface, 1, 10); pixel.Red < 200 || pixel.Green > 50 {
		t.Errorf("first stop = %#v, want red", pixel)
	}
	if pixel := pixelAt(surface, 20, 10); pixel.Green < 180 || pixel.Red > 80 || pixel.Blue > 80 {
		t.Errorf("middle stop = %#v, want green", pixel)
	}
	if pixel := pixelAt(surface, 38, 10); pixel.Blue < 200 || pixel.Green > 50 {
		t.Errorf("last stop = %#v, want blue", pixel)
	}
}

func TestRepeatingLinearGradient(t *testing.T) {
	surface := renderTestPage(t, gradientPage("repeating-linear-gradient(90deg, #ff0000 0 10px, #0000ff 10px 20px)"), 40, 20)
	checks := []struct {
		x       int
		wantRed bool
	}{
		{5, true},
		{15, false},
		{25, true},
		{35, false},
	}
	for _, check := range checks {
		pixel := pixelAt(surface, check.x, 10)
		if check.wantRed && (pixel.Red < 190 || pixel.Blue > 60) {
			t.Errorf("x=%d pixel = %#v, want red", check.x, pixel)
		}
		if !check.wantRed && (pixel.Blue < 190 || pixel.Red > 60) {
			t.Errorf("x=%d pixel = %#v, want blue", check.x, pixel)
		}
	}
}

func TestRadialGradient(t *testing.T) {
	surface := renderTestPage(t, gradientPage("radial-gradient(circle at center, #ff0000, #0000ff)"), 40, 20)
	center := pixelAt(surface, 20, 10)
	corner := pixelAt(surface, 1, 1)
	if center.Red < 200 {
		t.Errorf("center = %#v, want red", center)
	}
	if corner.Blue < 150 || corner.Red > 150 {
		t.Errorf("corner = %#v, want blue", corner)
	}
}

func TestConicGradient(t *testing.T) {
	surface := renderTestPage(t, gradientPage("conic-gradient(from 0deg at center, #ff0000, #0000ff)"), 40, 20)
	top := pixelAt(surface, 20, 2)
	right := pixelAt(surface, 38, 10)
	bottom := pixelAt(surface, 20, 18)
	left := pixelAt(surface, 2, 10)
	if top.Red < 150 || top.Blue > 100 {
		t.Errorf("top = %#v, want red-ish", top)
	}
	if right.Red < 150 || right.Blue < 40 || right.Blue > 110 {
		t.Errorf("right = %#v, want quarter-turn mix", right)
	}
	if bottom.Red < 100 || bottom.Blue < 100 {
		t.Errorf("bottom = %#v, want half-turn mix", bottom)
	}
	if left.Blue < 150 || left.Red > 150 {
		t.Errorf("left = %#v, want blue-ish", left)
	}
}

func TestGradientCurrentColor(t *testing.T) {
	source := `<!doctype html><html><head><style>
html, body { margin:0; padding:0 }
.box { width:40px; height:20px; color:#00ff00; background:linear-gradient(to right, currentcolor, #ffffff) }
</style></head><body><div class="box"></div></body></html>`
	surface := renderTestPage(t, source, 40, 20)
	pixel := pixelAt(surface, 1, 10)
	if pixel.Green < 190 || pixel.Red > 90 {
		t.Errorf("currentcolor gradient = %#v", pixel)
	}
}

func TestRadialGradientExtentKeywords(t *testing.T) {
	for _, background := range []string{
		"radial-gradient(closest-side circle at center, #ff0000, #0000ff)",
		"radial-gradient(farthest-side ellipse at center, #ff0000, #0000ff)",
		"radial-gradient(closest-corner circle at center, #ff0000, #0000ff)",
		"radial-gradient(farthest-corner ellipse at center, #ff0000, #0000ff)",
	} {
		surface := renderTestPage(t, gradientPage(background), 40, 20)
		if pixel := pixelAt(surface, 20, 10); pixel.Red < 150 {
			t.Errorf("%s center = %#v", background, pixel)
		}
	}
}

func TestGradientGolden(t *testing.T) {
	source := `<!doctype html><html><head><style>
html, body { margin:0; padding:0; background:#ffffff }
.swatch { display:inline-block; width:60px; height:60px; margin:5px }
.linear { background:linear-gradient(135deg, #ff0000, #ffdd00 40%, #00aa55) }
.radial { background:radial-gradient(circle at 30% 30%, #ffffff, #3366cc 60%, #001133) }
.conic { background:repeating-conic-gradient(from 45deg, #ff0000 0 15deg, #ffffff 15deg 30deg) }
</style></head><body>
<div class="swatch linear"></div><div class="swatch radial"></div><div class="swatch conic"></div>
</body></html>`
	surface := renderTestPage(t, source, 220, 80)
	path := filepath.Join("testdata", "golden", "gradients.png")
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
	compareGolden(t, path, surface)
}

func TestGradientBackgroundSizeAndPosition(t *testing.T) {
	source := `<!doctype html><html><head><style>
html, body { margin:0; padding:0 }
.box { width:40px; height:40px; background-image:linear-gradient(to right, #ff0000, #0000ff);
       background-size:20px 20px; background-repeat:no-repeat; background-position:right bottom }
</style></head><body><div class="box"></div></body></html>`
	surface := renderTestPage(t, source, 40, 40)
	if pixel := pixelAt(surface, 38, 38); pixel.Blue < 150 {
		t.Errorf("positioned gradient = %#v, want blue", pixel)
	}
	if pixel := pixelAt(surface, 5, 5); pixel.Red != 255 || pixel.Green != 255 || pixel.Blue != 255 {
		t.Errorf("outside positioned gradient = %#v, want white", pixel)
	}
}
