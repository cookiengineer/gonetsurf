package images

import (
	"encoding/base64"
	"testing"
)

func FuzzDecode(f *testing.F) {
	pngBytes, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAFgwJ/lJ8QGQAAAABJRU5ErkJggg==")
	f.Add(pngBytes)
	f.Add([]byte{0xFF, 0xD8, 0xFF, 0xD9})
	f.Add([]byte("GIF89a"))
	f.Add([]byte("RIFF\x00\x00\x00\x00WEBPVP8 "))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = Decode(data, Options{MaxPixels: 1 << 20, MaxDimension: 4096, MaxFrames: 8})
	})
}
