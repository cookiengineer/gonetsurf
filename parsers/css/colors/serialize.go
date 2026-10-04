package colors

import (
	"math"
	"strconv"
	"strings"
)

// Serialize returns the CSS Color Level 4 serialization of the color.
func (color Color) Serialize() string {
	switch color.Space {
	case SpaceRGB:
		return color.serializeRGB()
	case SpaceHSL:
		return color.hslToRGB().serializeRGB()
	case SpaceHWB:
		return color.hwbToRGB().serializeRGB()
	case SpaceLab:
		return color.serializeLabLike("lab")
	case SpaceLCH:
		return color.serializeLabLike("lch")
	case SpaceOKLab:
		return color.serializeLabLike("oklab")
	case SpaceOKLCH:
		return color.serializeLabLike("oklch")
	case SpaceCustom:
		return color.serializeCustom()
	default:
		return ""
	}
}

func (color Color) serializeRGB() string {
	red := color.channel(0)
	green := color.channel(1)
	blue := color.channel(2)
	if color.HasAlpha && color.Alpha < 1 {
		return "rgba(" + red + ", " + green + ", " + blue + ", " + formatNumber(color.Alpha) + ")"
	}
	return "rgb(" + red + ", " + green + ", " + blue + ")"
}

func (color Color) channel(index int) string {
	if color.None[index] {
		return "none"
	}
	return formatNumber(color.Components[index])
}

func (color Color) serializeLabLike(name string) string {
	parts := make([]string, 3)
	for index := 0; index < 3; index++ {
		parts[index] = color.channel(index)
	}
	body := strings.Join(parts, " ")
	if color.HasAlpha && color.Alpha < 1 {
		return name + "(" + body + " / " + formatNumber(color.Alpha) + ")"
	}
	return name + "(" + body + ")"
}

func (color Color) serializeCustom() string {
	parts := make([]string, 3)
	for index := 0; index < 3; index++ {
		parts[index] = color.channel(index)
	}
	body := canonicalSpace(color.CustomSpace) + " " + strings.Join(parts, " ")
	if color.HasAlpha && color.Alpha < 1 {
		return "color(" + body + " / " + formatNumber(color.Alpha) + ")"
	}
	return "color(" + body + ")"
}

// canonicalSpace maps color() space aliases to their serialized names.
func canonicalSpace(space string) string {
	switch space {
	case "xyz":
		return "xyz-d65"
	default:
		return space
	}
}

func formatNumber(value float64) string {
	if value == math.Trunc(value) && math.Abs(value) < 1e15 {
		return strconv.FormatInt(int64(value), 10)
	}
	rounded := math.Round(value*1e6) / 1e6
	return strconv.FormatFloat(rounded, 'f', -1, 64)
}

func (color Color) hslToRGB() *Color {
	hue := math.Mod(color.Components[0], 360)
	if hue < 0 {
		hue += 360
	}
	saturation := color.Components[1]
	lightness := color.Components[2]
	var red, green, blue float64
	if saturation == 0 {
		red, green, blue = lightness, lightness, lightness
	} else {
		var q float64
		if lightness < 0.5 {
			q = lightness * (1 + saturation)
		} else {
			q = lightness + saturation - lightness*saturation
		}
		p := 2*lightness - q
		red = hueToRGB(p, q, hue/360+1.0/3.0)
		green = hueToRGB(p, q, hue/360)
		blue = hueToRGB(p, q, hue/360-1.0/3.0)
	}
	return &Color{
		Space:      SpaceRGB,
		Components: [3]float64{red * 255, green * 255, blue * 255},
		Alpha:      color.Alpha,
		HasAlpha:   color.HasAlpha,
	}
}

func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t += 1
	}
	if t > 1 {
		t -= 1
	}
	switch {
	case t < 1.0/6.0:
		return p + (q-p)*6*t
	case t < 1.0/2.0:
		return q
	case t < 2.0/3.0:
		return p + (q-p)*(2.0/3.0-t)*6
	default:
		return p
	}
}

func (color Color) hwbToRGB() *Color {
	hue := math.Mod(color.Components[0], 360)
	if hue < 0 {
		hue += 360
	}
	whiteness := color.Components[1]
	blackness := color.Components[2]
	if whiteness+blackness >= 1 {
		gray := whiteness / (whiteness + blackness)
		return &Color{
			Space:      SpaceRGB,
			Components: [3]float64{gray * 255, gray * 255, gray * 255},
			Alpha:      color.Alpha,
			HasAlpha:   color.HasAlpha,
		}
	}
	pure := (&Color{Space: SpaceHSL, Components: [3]float64{hue, 1, 0.5}}).hslToRGB()
	factor := 1 - whiteness - blackness
	components := [3]float64{}
	for index := 0; index < 3; index++ {
		components[index] = (pure.Components[index]/255*factor + whiteness) * 255
	}
	return &Color{Space: SpaceRGB, Components: components, Alpha: color.Alpha, HasAlpha: color.HasAlpha}
}
