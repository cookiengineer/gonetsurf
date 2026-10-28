// Package images decodes raster images for the renderer. It supports PNG,
// JPEG, GIF (including animation), and WebP, applies EXIF orientation, and
// enforces pixel, dimension, and frame budgets before allocating decoded
// surfaces.
package images

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"time"

	_ "golang.org/x/image/webp"
)

// Orientation is an EXIF orientation value (1-8).
type Orientation uint8

// Frame is one decoded image frame.
type Frame struct {
	Image *image.NRGBA
	Delay time.Duration
}

// Image is a decoded image and its animation metadata.
type Image struct {
	Width       int
	Height      int
	Frames      []Frame
	LoopCount   int
	Orientation Orientation
}

// Options limits decoding resource use. Zero values select the defaults.
type Options struct {
	MaxPixels    int64
	MaxDimension int
	MaxFrames    int
}

// DefaultOptions returns conservative decode budgets.
func DefaultOptions() Options {
	return Options{
		MaxPixels:    64 << 20,
		MaxDimension: 16384,
		MaxFrames:    256,
	}
}

// ErrBudget reports an image that exceeds a decode budget.
var ErrBudget = errors.New("images: image exceeds decode budget")

func (options Options) normalized() Options {
	defaults := DefaultOptions()
	if options.MaxPixels <= 0 {
		options.MaxPixels = defaults.MaxPixels
	}
	if options.MaxDimension <= 0 {
		options.MaxDimension = defaults.MaxDimension
	}
	if options.MaxFrames <= 0 {
		options.MaxFrames = defaults.MaxFrames
	}
	return options
}

// Decode decodes data into one or more frames. The format is detected from the
// data; the decoded frames are NRGBA and already respect EXIF orientation.
func Decode(data []byte, options Options) (*Image, error) {
	options = options.normalized()
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if err := options.checkDimensions(config.Width, config.Height); err != nil {
		return nil, err
	}

	if format == "gif" {
		return decodeGIF(data, options)
	}

	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	bounds := decoded.Bounds()
	if err := options.checkDimensions(bounds.Dx(), bounds.Dy()); err != nil {
		return nil, err
	}
	converted := toNRGBA(decoded)
	orientation := Orientation(1)
	if format == "jpeg" {
		orientation = exifOrientation(data)
	}
	if orientation > 1 {
		converted = applyOrientation(converted, orientation)
	}
	return &Image{
		Width:       converted.Bounds().Dx(),
		Height:      converted.Bounds().Dy(),
		Frames:      []Frame{{Image: converted}},
		LoopCount:   -1,
		Orientation: orientation,
	}, nil
}

func (options Options) checkDimensions(width, height int) error {
	if width <= 0 || height <= 0 {
		return fmt.Errorf("images: invalid dimensions %dx%d", width, height)
	}
	if width > options.MaxDimension || height > options.MaxDimension {
		return ErrBudget
	}
	if int64(width)*int64(height) > options.MaxPixels {
		return ErrBudget
	}
	return nil
}

func toNRGBA(source image.Image) *image.NRGBA {
	if converted, ok := source.(*image.NRGBA); ok {
		copyImage := image.NewNRGBA(image.Rect(0, 0, converted.Bounds().Dx(), converted.Bounds().Dy()))
		draw.Draw(copyImage, copyImage.Bounds(), converted, converted.Bounds().Min, draw.Src)
		return copyImage
	}
	bounds := source.Bounds()
	converted := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(converted, converted.Bounds(), source, bounds.Min, draw.Src)
	return converted
}

func copyNRGBA(source *image.NRGBA) *image.NRGBA {
	copyImage := image.NewNRGBA(image.Rect(0, 0, source.Bounds().Dx(), source.Bounds().Dy()))
	copy(copyImage.Pix, source.Pix)
	return copyImage
}
