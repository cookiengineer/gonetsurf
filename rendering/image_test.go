package rendering

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"gonetsurf/rendering/images"
	"gonetsurf/urls"
)

func TestURLImageProviderHTTP(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	for yPos := 0; yPos < 2; yPos++ {
		for xPos := 0; xPos < 3; xPos++ {
			source.SetNRGBA(xPos, yPos, color.NRGBA{R: 40, G: 80, B: 120, A: 255})
		}
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, source); err != nil {
		t.Fatal(err)
	}
	payload := buffer.Bytes()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/assets/background.png" {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", "image/png")
		writer.Write(payload)
	}))
	defer server.Close()

	base, err := urls.Parse(server.URL + "/page.html")
	if err != nil {
		t.Fatal(err)
	}
	provider := URLImageProvider{
		Base: base,
		Fetch: func(resource string) ([]byte, error) {
			response, err := http.Get(resource)
			if err != nil {
				return nil, err
			}
			defer response.Body.Close()
			return io.ReadAll(response.Body)
		},
	}
	loaded, err := provider.LoadImage("url(assets/background.png)")
	if err != nil {
		t.Fatal(err)
	}
	bounds := loaded.Bounds()
	if bounds.Dx() != 3 || bounds.Dy() != 2 {
		t.Errorf("image bounds = %v", bounds)
	}
}

func TestHTTPBackgroundImagePaint(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for yPos := 0; yPos < 4; yPos++ {
		for xPos := 0; xPos < 4; xPos++ {
			source.SetNRGBA(xPos, yPos, color.NRGBA{G: 200, A: 255})
		}
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, source); err != nil {
		t.Fatal(err)
	}
	payload := buffer.Bytes()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "image/png")
		writer.Write(payload)
	}))
	defer server.Close()

	page := `<!doctype html><html><head><style>
html, body { margin:0; padding:0 }
.box { width:20px; height:20px; background-image:url(` + server.URL + `/bg.png); background-size:20px 20px; background-repeat:no-repeat }
</style></head><body><div class="box"></div></body></html>`
	provider := URLImageProvider{
		Fetch: func(resource string) ([]byte, error) {
			response, err := http.Get(resource)
			if err != nil {
				return nil, err
			}
			defer response.Body.Close()
			return io.ReadAll(response.Body)
		},
	}
	surface := renderTestPageWithProvider(t, page, 30, 30, provider)
	if pixel := pixelAt(surface, 5, 5); pixel.Green < 150 {
		t.Errorf("HTTP background pixel = %#v, want green", pixel)
	}
}

func TestURLImageProviderBudget(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, source); err != nil {
		t.Fatal(err)
	}
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buffer.Bytes())
	provider := URLImageProvider{Options: images.Options{MaxPixels: 100}}
	if _, err := provider.LoadImage("url(" + dataURL + ")"); err == nil {
		t.Fatal("expected a decode budget error")
	}
}
