package rendering

import (
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestStackingZIndex(t *testing.T) {
	source := `<!doctype html><html><head><style>
html, body { margin:0; padding:0 }
.parent { position:relative; width:80px; height:40px; background-color:#ffffff }
.red { position:absolute; left:0; top:0; width:40px; height:40px; background-color:#ff0000; z-index:1 }
.blue { position:absolute; left:20px; top:0; width:40px; height:40px; background-color:#0000ff; z-index:2 }
</style></head><body><div class="parent"><div class="red"></div><div class="blue"></div></div></body></html>`
	surface := renderTestPage(t, source, 100, 60)
	if pixel := pixelAt(surface, 10, 20); pixel.Red < 200 || pixel.Blue > 50 {
		t.Errorf("z-index 1 region = %#v, want red", pixel)
	}
	if pixel := pixelAt(surface, 30, 20); pixel.Blue < 200 || pixel.Red > 50 {
		t.Errorf("z-index 2 overlap = %#v, want blue", pixel)
	}
}

func TestNegativeZIndexPaintsAboveBackground(t *testing.T) {
	source := `<!doctype html><html><head><style>
html, body { margin:0; padding:0 }
.parent { position:relative; width:40px; height:40px; background-color:#ffffff }
.behind { position:absolute; left:0; top:0; width:40px; height:40px; background-color:#ff0000; z-index:-1 }
.above { position:absolute; left:0; top:0; width:20px; height:20px; background-color:#00ff00; z-index:0 }
</style></head><body><div class="parent"><div class="behind"></div><div class="above"></div></div></body></html>`
	surface := renderTestPage(t, source, 60, 60)
	if pixel := pixelAt(surface, 30, 30); pixel.Red < 200 {
		t.Errorf("negative z region = %#v, want red above parent background", pixel)
	}
	if pixel := pixelAt(surface, 10, 10); pixel.Green < 200 {
		t.Errorf("z-index 0 overlap = %#v, want green", pixel)
	}
}

func TestPositionedAutoPaintsAboveFlow(t *testing.T) {
	source := `<!doctype html><html><head><style>
html, body { margin:0; padding:0 }
.flow { width:40px; height:40px; background-color:#00ff00 }
.overlay { position:absolute; left:20px; top:0; width:40px; height:40px; background-color:#ff0000; z-index:auto }
</style></head><body><div class="flow"></div><div class="overlay"></div></body></html>`
	surface := renderTestPage(t, source, 80, 60)
	if pixel := pixelAt(surface, 30, 20); pixel.Red < 200 {
		t.Errorf("positioned auto overlap = %#v, want red", pixel)
	}
}

func TestTransformTranslate(t *testing.T) {
	source := `<!doctype html><html><head><style>
html, body { margin:0; padding:0 }
.box { position:absolute; left:0; top:0; width:20px; height:20px; background-color:#ff0000; transform:translate(20px, 10px) }
</style></head><body><div class="box"></div></body></html>`
	surface := renderTestPage(t, source, 60, 60)
	if pixel := pixelAt(surface, 5, 5); pixel.Red < 200 {
		t.Errorf("untransformed origin = %#v, want white", pixel)
	}
	if pixel := pixelAt(surface, 25, 15); pixel.Red < 200 {
		t.Errorf("translated box = %#v, want red", pixel)
	}
}

func TestTransformRotate(t *testing.T) {
	source := `<!doctype html><html><head><style>
html, body { margin:0; padding:0 }
.box { position:absolute; left:20px; top:20px; width:20px; height:10px; background-color:#0000ff; transform:rotate(90deg) }
</style></head><body><div class="box"></div></body></html>`
	surface := renderTestPage(t, source, 60, 60)
	if pixel := pixelAt(surface, 30, 25); pixel.Blue < 200 {
		t.Errorf("rotated center = %#v, want blue", pixel)
	}
	if pixel := pixelAt(surface, 22, 22); pixel.Red != 255 || pixel.Green != 255 || pixel.Blue != 255 {
		t.Errorf("outside rotated box = %#v, want white", pixel)
	}
}

func TestOpacityLayer(t *testing.T) {
	source := `<!doctype html><html><head><style>
html, body { margin:0; padding:0; background-color:#ffffff }
.box { width:20px; height:20px; background-color:#ff0000; opacity:0.5 }
</style></head><body><div class="box"></div></body></html>`
	surface := renderTestPage(t, source, 40, 40)
	pixel := pixelAt(surface, 10, 10)
	if pixel.Red < 240 || pixel.Green < 120 || pixel.Green > 140 || pixel.Blue < 120 || pixel.Blue > 140 {
		t.Errorf("opacity pixel = %#v, want ~half red over white", pixel)
	}
}

func TestReplacedImage(t *testing.T) {
	dataURL := imageDataURL(t, color.NRGBA{G: 190, A: 255}, 4, 4)
	source := `<!doctype html><html><head><style>
html, body { margin:0; padding:0 }
img { display:block; width:20px; height:20px }
</style></head><body><img src="` + dataURL + `"></body></html>`
	surface := renderTestPage(t, source, 40, 40)
	if pixel := pixelAt(surface, 10, 10); pixel.Green < 150 {
		t.Errorf("replaced image pixel = %#v, want green", pixel)
	}
}

func TestStackingGolden(t *testing.T) {
	dataURL := imageDataURL(t, color.NRGBA{R: 30, G: 160, B: 220, A: 255}, 8, 8)
	source := `<!doctype html><html><head><style>
html, body { margin:0; padding:0; background-color:#ffffff }
.stage { position:relative; width:160px; height:100px }
.back { position:absolute; left:10px; top:10px; width:70px; height:60px; background-color:#f0c060; z-index:1 }
.front { position:absolute; left:40px; top:30px; width:70px; height:60px; background-color:#3366cc; z-index:2; opacity:0.75 }
.rotated { position:absolute; left:110px; top:20px; width:30px; height:16px; background-color:#cc3333; transform:rotate(25deg) }
.photo { position:absolute; left:110px; top:60px; width:40px; height:30px; object-fit:contain; z-index:3 }
</style></head><body><div class="stage">
<div class="back"></div><div class="front"></div><div class="rotated"></div>
<img class="photo" src="` + dataURL + `">
</div></body></html>`
	surface := renderTestPage(t, source, 200, 130)
	path := filepath.Join("testdata", "golden", "stacking.png")
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
