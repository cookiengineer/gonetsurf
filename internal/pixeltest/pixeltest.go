// Package pixeltest compares rendered images against golden or reference
// images. It is shared by the rendering tests and the future WPT reftest
// harness.
package pixeltest

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
)

// Options controls comparison strictness.
type Options struct {
	// ChannelTolerance is the largest per-channel difference that is ignored.
	ChannelTolerance uint8
	// MaxDifferent is the number of differing pixels allowed.
	MaxDifferent int
}

// Result summarizes an image comparison.
type Result struct {
	Total           int
	Different       int
	MaxChannelDelta int
}

// OK reports whether the result satisfies options.
func (result Result) OK(options Options) bool {
	return result.Different <= options.MaxDifferent
}

func (result Result) String() string {
	return fmt.Sprintf("%d/%d pixels differ (max channel delta %d)", result.Different, result.Total, result.MaxChannelDelta)
}

// Compare compares two images pixel by pixel.
func Compare(expected, actual image.Image, options Options) (Result, error) {
	expectedBounds := expected.Bounds()
	actualBounds := actual.Bounds()
	if expectedBounds.Size() != actualBounds.Size() {
		return Result{}, fmt.Errorf("pixeltest: image sizes differ: %v vs %v", expectedBounds.Size(), actualBounds.Size())
	}
	expectedPixels := toNRGBA(expected)
	actualPixels := toNRGBA(actual)
	result := Result{Total: expectedBounds.Dx() * expectedBounds.Dy()}
	for offset := 0; offset < len(expectedPixels.Pix); offset += 4 {
		different := false
		for channel := 0; channel < 4; channel++ {
			delta := int(expectedPixels.Pix[offset+channel]) - int(actualPixels.Pix[offset+channel])
			if delta < 0 {
				delta = -delta
			}
			if delta > result.MaxChannelDelta {
				result.MaxChannelDelta = delta
			}
			if delta > int(options.ChannelTolerance) {
				different = true
			}
		}
		if different {
			result.Different++
		}
	}
	return result, nil
}

// CompareFile decodes the PNG at path and compares it with actual.
func CompareFile(path string, actual image.Image, options Options) (Result, error) {
	file, err := os.Open(path)
	if err != nil {
		return Result{}, err
	}
	defer file.Close()
	expected, err := png.Decode(file)
	if err != nil {
		return Result{}, fmt.Errorf("pixeltest: decode %s: %w", path, err)
	}
	return Compare(expected, actual, options)
}

func toNRGBA(source image.Image) *image.NRGBA {
	bounds := source.Bounds()
	converted := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(converted, converted.Bounds(), source, bounds.Min, draw.Src)
	return converted
}
