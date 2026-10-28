package images

import (
	"bytes"
	"image"
	"image/draw"
	"image/gif"
	"time"
)

func decodeGIF(data []byte, options Options) (*Image, error) {
	animation, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	width := animation.Config.Width
	height := animation.Config.Height
	if width == 0 || height == 0 {
		width = animation.Image[0].Bounds().Dx()
		height = animation.Image[0].Bounds().Dy()
	}
	if err := options.checkDimensions(width, height); err != nil {
		return nil, err
	}
	if len(animation.Image) > options.MaxFrames {
		return nil, ErrBudget
	}
	totalPixels := int64(0)
	for _, frame := range animation.Image {
		totalPixels += int64(frame.Bounds().Dx()) * int64(frame.Bounds().Dy())
		if totalPixels > options.MaxPixels {
			return nil, ErrBudget
		}
	}

	canvas := image.NewNRGBA(image.Rect(0, 0, width, height))
	var previous *image.NRGBA
	frames := make([]Frame, 0, len(animation.Image))
	for index, frame := range animation.Image {
		disposal := byte(0)
		if index < len(animation.Disposal) {
			disposal = animation.Disposal[index]
		}
		if disposal == gif.DisposalPrevious {
			previous = copyNRGBA(canvas)
		}
		bounds := frame.Bounds()
		draw.Draw(canvas, bounds, frame, bounds.Min, draw.Over)
		frames = append(frames, Frame{
			Image: copyNRGBA(canvas),
			Delay: gifDelay(animation, index),
		})
		switch disposal {
		case gif.DisposalBackground:
			draw.Draw(canvas, bounds, image.Transparent, image.Point{}, draw.Src)
		case gif.DisposalPrevious:
			if previous != nil {
				canvas = previous
				previous = nil
			}
		}
	}
	return &Image{
		Width:       width,
		Height:      height,
		Frames:      frames,
		LoopCount:   animation.LoopCount,
		Orientation: 1,
	}, nil
}

func gifDelay(animation *gif.GIF, index int) time.Duration {
	if index >= len(animation.Delay) {
		return 0
	}
	return time.Duration(animation.Delay[index]) * 10 * time.Millisecond
}
