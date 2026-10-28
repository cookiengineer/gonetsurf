package rendering

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"testing"
)

type mapImageProvider struct {
	images map[string]*image.NRGBA
}

func (provider mapImageProvider) LoadImage(resource string) (image.Image, error) {
	for name, image := range provider.images {
		if strings.Contains(resource, name) {
			return image, nil
		}
	}
	return nil, fmt.Errorf("missing image %q", resource)
}

func solidImage(fill color.NRGBA) *image.NRGBA {
	result := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for yPos := 0; yPos < 4; yPos++ {
		for xPos := 0; xPos < 4; xPos++ {
			result.SetNRGBA(xPos, yPos, fill)
		}
	}
	return result
}

func TestImageSrcsetSelection(t *testing.T) {
	provider := mapImageProvider{images: map[string]*image.NRGBA{
		"small.png": solidImage(color.NRGBA{R: 220, A: 255}),
		"large.png": solidImage(color.NRGBA{G: 200, A: 255}),
	}}
	source := `<!doctype html><html><head><style>
html, body { margin:0; padding:0 }
img { display:block; width:50px; height:20px }
</style></head><body><img src="small.png" srcset="small.png 10w, large.png 100w"></body></html>`
	surface := renderTestPageWithProvider(t, source, 80, 50, provider)
	if pixel := pixelAt(surface, 25, 10); pixel.Green < 150 {
		t.Errorf("srcset pixel = %#v, want large green image", pixel)
	}
}

func TestPictureSourceSelection(t *testing.T) {
	provider := mapImageProvider{images: map[string]*image.NRGBA{
		"fallback.png": solidImage(color.NRGBA{R: 220, A: 255}),
		"wide.png":     solidImage(color.NRGBA{B: 220, A: 255}),
	}}
	source := `<!doctype html><html><head><style>
html, body { margin:0; padding:0 }
img { display:block; width:30px; height:20px }
</style></head><body><picture><source srcset="wide.png 100w"><img src="fallback.png"></picture></body></html>`
	surface := renderTestPageWithProvider(t, source, 60, 40, provider)
	if pixel := pixelAt(surface, 15, 10); pixel.Blue < 150 || pixel.Red > 100 {
		t.Errorf("picture pixel = %#v, want wide blue image", pixel)
	}
}

func TestVideoPoster(t *testing.T) {
	provider := mapImageProvider{images: map[string]*image.NRGBA{
		"poster.png": solidImage(color.NRGBA{G: 200, A: 255}),
	}}
	source := `<!doctype html><html><head><style>
html, body { margin:0; padding:0 }
video { display:block; width:30px; height:20px }
</style></head><body><video poster="poster.png"></video></body></html>`
	surface := renderTestPageWithProvider(t, source, 60, 40, provider)
	if pixel := pixelAt(surface, 15, 10); pixel.Green < 150 {
		t.Errorf("video poster pixel = %#v, want green", pixel)
	}
}
