package images

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
	"time"
)

const webpFixture = "UklGRjwAAABXRUJQVlA4IDAAAADQAQCdASoCAAIAAgA0JaACdLoB+AADsAD+8MQL/yC5YXXI1/8gP+QH/ID/+PIAAAA="

func encodePNG(t *testing.T, source image.Image) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, source); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestDecodePNG(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	source.SetNRGBA(1, 1, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	decoded, err := Decode(encodePNG(t, source), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Width != 3 || decoded.Height != 2 || len(decoded.Frames) != 1 {
		t.Fatalf("decoded = %#v", decoded)
	}
	pixel := decoded.Frames[0].Image.NRGBAAt(1, 1)
	if pixel.R != 10 || pixel.G != 20 || pixel.B != 30 || pixel.A != 255 {
		t.Errorf("pixel = %#v", pixel)
	}
	if decoded.Orientation != 1 {
		t.Errorf("orientation = %d", decoded.Orientation)
	}
}

func encodeJPEGWithOrientation(t *testing.T, source image.Image, orientation uint16) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, source, nil); err != nil {
		t.Fatal(err)
	}
	jpegData := buffer.Bytes()
	tiff := make([]byte, 26)
	copy(tiff[0:2], "II")
	binary.LittleEndian.PutUint16(tiff[2:4], 42)
	binary.LittleEndian.PutUint32(tiff[4:8], 8)
	binary.LittleEndian.PutUint16(tiff[8:10], 1)
	binary.LittleEndian.PutUint16(tiff[10:12], 0x0112)
	binary.LittleEndian.PutUint16(tiff[12:14], 3)
	binary.LittleEndian.PutUint32(tiff[14:18], 1)
	binary.LittleEndian.PutUint16(tiff[18:20], orientation)
	payload := append([]byte("Exif\x00\x00"), tiff...)
	segmentLength := 2 + len(payload)
	var result []byte
	result = append(result, 0xFF, 0xD8, 0xFF, 0xE1)
	result = append(result, byte(segmentLength>>8), byte(segmentLength))
	result = append(result, payload...)
	result = append(result, jpegData[2:]...)
	return result
}

func TestDecodeJPEGOrientation(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 16, 8))
	for yPos := 0; yPos < 8; yPos++ {
		for xPos := 0; xPos < 16; xPos++ {
			if xPos < 8 {
				source.SetNRGBA(xPos, yPos, color.NRGBA{R: 255, A: 255})
			} else {
				source.SetNRGBA(xPos, yPos, color.NRGBA{B: 255, A: 255})
			}
		}
	}
	decoded, err := Decode(encodeJPEGWithOrientation(t, source, 6), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Orientation != 6 {
		t.Errorf("orientation = %d, want 6", decoded.Orientation)
	}
	if decoded.Width != 8 || decoded.Height != 16 {
		t.Fatalf("oriented dimensions = %dx%d, want 8x16", decoded.Width, decoded.Height)
	}
	top := decoded.Frames[0].Image.NRGBAAt(4, 4)
	bottom := decoded.Frames[0].Image.NRGBAAt(4, 12)
	if top.R < top.B || bottom.B < bottom.R {
		t.Errorf("oriented pixels top=%#v bottom=%#v", top, bottom)
	}
}

func TestExifOrientationParser(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	data := encodeJPEGWithOrientation(t, source, 8)
	if got := exifOrientation(data); got != 8 {
		t.Errorf("exifOrientation = %d, want 8", got)
	}
	if got := exifOrientation([]byte("not a jpeg")); got != 1 {
		t.Errorf("invalid exif orientation = %d, want 1", got)
	}
}

func TestApplyOrientation(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	values := map[string]color.NRGBA{
		"A": {R: 10, A: 255},
		"B": {R: 20, A: 255},
		"C": {R: 30, A: 255},
		"D": {R: 40, A: 255},
	}
	source.SetNRGBA(0, 0, values["A"])
	source.SetNRGBA(1, 0, values["B"])
	source.SetNRGBA(0, 1, values["C"])
	source.SetNRGBA(1, 1, values["D"])
	expected := map[Orientation][][]string{
		1: {{"A", "B"}, {"C", "D"}},
		2: {{"B", "A"}, {"D", "C"}},
		3: {{"D", "C"}, {"B", "A"}},
		4: {{"C", "D"}, {"A", "B"}},
		5: {{"A", "C"}, {"B", "D"}},
		6: {{"C", "A"}, {"D", "B"}},
		7: {{"D", "B"}, {"C", "A"}},
		8: {{"B", "D"}, {"A", "C"}},
	}
	for orientation, rows := range expected {
		result := applyOrientation(source, orientation)
		if result.Bounds().Dx() != len(rows[0]) || result.Bounds().Dy() != len(rows) {
			t.Fatalf("orientation %d bounds = %v", orientation, result.Bounds())
		}
		for yPos, row := range rows {
			for xPos, name := range row {
				pixel := result.NRGBAAt(xPos, yPos)
				if pixel != values[name] {
					t.Errorf("orientation %d pixel (%d,%d) = %#v, want %s", orientation, xPos, yPos, pixel, name)
				}
			}
		}
	}
}

func gifFixture(t *testing.T) []byte {
	t.Helper()
	palette := color.Palette{
		color.NRGBA{},
		color.NRGBA{R: 255, A: 255},
		color.NRGBA{G: 255, A: 255},
		color.NRGBA{B: 255, A: 255},
	}
	first := image.NewPaletted(image.Rect(0, 0, 2, 1), palette)
	first.SetColorIndex(0, 0, 1)
	first.SetColorIndex(1, 0, 3)
	second := image.NewPaletted(image.Rect(0, 0, 2, 1), palette)
	second.SetColorIndex(0, 0, 2)
	second.SetColorIndex(1, 0, 2)
	animation := &gif.GIF{
		Image:     []*image.Paletted{first, second},
		Delay:     []int{10, 20},
		Disposal:  []byte{gif.DisposalNone, gif.DisposalNone},
		LoopCount: 0,
		Config:    image.Config{ColorModel: palette, Width: 2, Height: 1},
	}
	var buffer bytes.Buffer
	if err := gif.EncodeAll(&buffer, animation); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestDecodeGIFAnimation(t *testing.T) {
	decoded, err := Decode(gifFixture(t), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Frames) != 2 || decoded.LoopCount != 0 {
		t.Fatalf("frames = %d loop = %d", len(decoded.Frames), decoded.LoopCount)
	}
	if decoded.Frames[0].Delay != 100*time.Millisecond || decoded.Frames[1].Delay != 200*time.Millisecond {
		t.Errorf("delays = %v, %v", decoded.Frames[0].Delay, decoded.Frames[1].Delay)
	}
	first := decoded.Frames[0].Image.NRGBAAt(0, 0)
	if first.R < 200 {
		t.Errorf("first frame pixel = %#v, want red", first)
	}
	second := decoded.Frames[1].Image.NRGBAAt(0, 0)
	if second.G < 200 {
		t.Errorf("second frame pixel = %#v, want green", second)
	}
}

func TestDecodeWebP(t *testing.T) {
	data, err := base64.StdEncoding.DecodeString(webpFixture)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(data, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Width != 2 || decoded.Height != 2 {
		t.Errorf("webp dimensions = %dx%d", decoded.Width, decoded.Height)
	}
	for yPos := 0; yPos < 2; yPos++ {
		for xPos := 0; xPos < 2; xPos++ {
			pixel := decoded.Frames[0].Image.NRGBAAt(xPos, yPos)
			if pixel.R < 200 || pixel.G > 50 || pixel.B > 50 {
				t.Errorf("webp pixel (%d,%d) = %#v, want red", xPos, yPos, pixel)
			}
		}
	}
}

func TestDecodeBudgets(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	data := encodePNG(t, source)
	if _, err := Decode(data, Options{MaxPixels: 100}); !errors.Is(err, ErrBudget) {
		t.Errorf("max pixels error = %v", err)
	}
	if _, err := Decode(data, Options{MaxDimension: 32}); !errors.Is(err, ErrBudget) {
		t.Errorf("max dimension error = %v", err)
	}
	if _, err := Decode(gifFixture(t), Options{MaxFrames: 1}); !errors.Is(err, ErrBudget) {
		t.Errorf("max frames error = %v", err)
	}
}

func TestDecodeInvalid(t *testing.T) {
	if _, err := Decode([]byte("not an image"), Options{}); err == nil {
		t.Fatal("expected an invalid image error")
	}
}
