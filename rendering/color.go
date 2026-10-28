package rendering

import (
	"math"

	"gonetsurf/parsers/css/colors"
)

// ParseColor parses a CSS color into a non-premultiplied 8-bit sRGB color.
// Out-of-gamut display-p3 and Lab-family colors are clipped to sRGB.
func ParseColor(input string) (Color, bool) {
	parsed, ok := colors.Parse(input)
	if !ok {
		return Transparent, false
	}
	return fromCSSColor(parsed)
}

func channelToByte(value float64) uint8 {
	if value <= 0 {
		return 0
	}
	if value >= 1 {
		return 255
	}
	return uint8(math.Round(value * 255))
}

func fromCSSColor(parsed *colors.Color) (Color, bool) {
	if parsed == nil {
		return Transparent, false
	}
	converted, ok := parsed.ToSRGBGamutMapped()
	if !ok {
		return Transparent, false
	}
	return Color{
		Red:   channelToByte(converted.Red),
		Green: channelToByte(converted.Green),
		Blue:  channelToByte(converted.Blue),
		Alpha: channelToByte(converted.Alpha),
	}, true
}
