package rendering

import (
	"image"
	"math"
)

// Color is a straight (non-premultiplied) sRGB color with 8-bit channels.
type Color struct {
	Red   uint8
	Green uint8
	Blue  uint8
	Alpha uint8
}

// NewColor returns an opaque color from its channels.
func NewColor(red, green, blue uint8) Color {
	return Color{Red: red, Green: green, Blue: blue, Alpha: 255}
}

// Transparent is the fully transparent color.
var Transparent = Color{}

// Rect is an axis-aligned rectangle in surface coordinates.
type Rect struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

// IsEmpty reports whether the rectangle has no area.
func (rect Rect) IsEmpty() bool {
	return rect.Width <= 0 || rect.Height <= 0
}

func (rect Rect) intersect(other Rect) Rect {
	minX := math.Max(rect.X, other.X)
	minY := math.Max(rect.Y, other.Y)
	maxX := math.Min(rect.X+rect.Width, other.X+other.Width)
	maxY := math.Min(rect.Y+rect.Height, other.Y+other.Height)
	if maxX <= minX || maxY <= minY {
		return Rect{}
	}
	return Rect{X: minX, Y: minY, Width: maxX - minX, Height: maxY - minY}
}

// Surface is a premultiplied RGBA8 pixel buffer.
//
// Pixels are stored row-major with four bytes per pixel. Drawing operations
// respect the current clip rectangle.
type Surface struct {
	Width  int
	Height int
	Stride int
	Pix    []uint8

	clip Rect
}

// NewSurface returns a transparent surface of the requested size.
func NewSurface(width, height int) *Surface {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	return &Surface{
		Width:  width,
		Height: height,
		Stride: width * 4,
		Pix:    make([]uint8, width*height*4),
		clip:   Rect{Width: float64(width), Height: float64(height)},
	}
}

// Bounds returns the surface rectangle in pixels.
func (surface *Surface) Bounds() Rect {
	return Rect{Width: float64(surface.Width), Height: float64(surface.Height)}
}

// SetClip replaces the clip rectangle. It is clipped to the surface bounds.
func (surface *Surface) SetClip(rect Rect) {
	surface.clip = rect.intersect(surface.Bounds())
}

// ResetClip restores the clip to the full surface.
func (surface *Surface) ResetClip() {
	surface.clip = surface.Bounds()
}

// Clip returns the current clip rectangle.
func (surface *Surface) Clip() Rect {
	return surface.clip
}

// Clear replaces every pixel with color.
func (surface *Surface) Clear(color Color) {
	red, green, blue, alpha := color.premultiplied(1)
	for offset := 0; offset < len(surface.Pix); offset += 4 {
		surface.Pix[offset] = red
		surface.Pix[offset+1] = green
		surface.Pix[offset+2] = blue
		surface.Pix[offset+3] = alpha
	}
}

// FillRect fills the part of rect inside the clip with color.
func (surface *Surface) FillRect(rect Rect, color Color) {
	if color.Alpha == 0 {
		return
	}
	rect = rect.intersect(surface.clip)
	if rect.IsEmpty() {
		return
	}
	minX, minY, maxX, maxY := surface.pixelBounds(rect)
	for yPos := minY; yPos < maxY; yPos++ {
		top := math.Max(rect.Y, float64(yPos))
		bottom := math.Min(rect.Y+rect.Height, float64(yPos+1))
		overlapY := bottom - top
		if overlapY <= 0 {
			continue
		}
		for xPos := minX; xPos < maxX; xPos++ {
			left := math.Max(rect.X, float64(xPos))
			right := math.Min(rect.X+rect.Width, float64(xPos+1))
			overlapX := right - left
			if overlapX <= 0 {
				continue
			}
			surface.blendPixel(xPos, yPos, color, overlapX*overlapY)
		}
	}
}

// FillPath fills path according to rule with color.
func (surface *Surface) FillPath(path *Path, rule FillRule, color Color) {
	if path == nil || color.Alpha == 0 {
		return
	}
	mask := RasterizeFill(path, surface.Width, surface.Height, rule)
	surface.applyMask(mask, color)
}

// StrokePath strokes path with the given style and color.
func (surface *Surface) StrokePath(path *Path, style StrokeStyle, color Color) {
	if path == nil || color.Alpha == 0 || style.Width <= 0 {
		return
	}
	mask := RasterizeStroke(path, surface.Width, surface.Height, style)
	surface.applyMask(mask, color)
}

// FillMask composites color through mask onto the surface.
func (surface *Surface) FillMask(mask *Mask, color Color) {
	surface.applyMask(mask, color)
}

func (surface *Surface) applyMask(mask *Mask, color Color) {
	if mask == nil {
		return
	}
	minX := int(math.Max(0, surface.clip.X))
	minY := int(math.Max(0, surface.clip.Y))
	maxX := int(math.Min(float64(surface.Width), surface.clip.X+surface.clip.Width))
	maxY := int(math.Min(float64(surface.Height), surface.clip.Y+surface.clip.Height))
	for yPos := minY; yPos < maxY; yPos++ {
		for xPos := minX; xPos < maxX; xPos++ {
			coverage := mask.At(xPos, yPos)
			if coverage > 0 {
				surface.blendPixel(xPos, yPos, color, coverage)
			}
		}
	}
}

func (surface *Surface) blendPixel(xPos, yPos int, color Color, coverage float64) {
	if coverage <= 0 || color.Alpha == 0 {
		return
	}
	if xPos < 0 || yPos < 0 || xPos >= surface.Width || yPos >= surface.Height {
		return
	}
	alpha := float64(color.Alpha) / 255 * coverage
	if alpha > 1 {
		alpha = 1
	}
	offset := yPos*surface.Stride + xPos*4
	destinationAlpha := float64(surface.Pix[offset+3]) / 255
	sourceRed := float64(color.Red) / 255 * alpha
	sourceGreen := float64(color.Green) / 255 * alpha
	sourceBlue := float64(color.Blue) / 255 * alpha

	surface.Pix[offset] = toByte(sourceRed + float64(surface.Pix[offset])/255*(1-alpha))
	surface.Pix[offset+1] = toByte(sourceGreen + float64(surface.Pix[offset+1])/255*(1-alpha))
	surface.Pix[offset+2] = toByte(sourceBlue + float64(surface.Pix[offset+2])/255*(1-alpha))
	surface.Pix[offset+3] = toByte(alpha + destinationAlpha*(1-alpha))
}

func (surface *Surface) pixelBounds(rect Rect) (minX, minY, maxX, maxY int) {
	minX = int(math.Floor(rect.X))
	minY = int(math.Floor(rect.Y))
	maxX = int(math.Ceil(rect.X + rect.Width))
	maxY = int(math.Ceil(rect.Y + rect.Height))
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX > surface.Width {
		maxX = surface.Width
	}
	if maxY > surface.Height {
		maxY = surface.Height
	}
	return minX, minY, maxX, maxY
}

// Image converts the premultiplied surface to a non-premultiplied NRGBA image.
func (surface *Surface) Image() *image.NRGBA {
	result := image.NewNRGBA(image.Rect(0, 0, surface.Width, surface.Height))
	for yPos := 0; yPos < surface.Height; yPos++ {
		for xPos := 0; xPos < surface.Width; xPos++ {
			sourceOffset := yPos*surface.Stride + xPos*4
			destinationOffset := yPos*result.Stride + xPos*4
			alpha := surface.Pix[sourceOffset+3]
			if alpha == 0 {
				continue
			}
			result.Pix[destinationOffset] = unmultiply(surface.Pix[sourceOffset], alpha)
			result.Pix[destinationOffset+1] = unmultiply(surface.Pix[sourceOffset+1], alpha)
			result.Pix[destinationOffset+2] = unmultiply(surface.Pix[sourceOffset+2], alpha)
			result.Pix[destinationOffset+3] = alpha
		}
	}
	return result
}

func (color Color) premultiplied(coverage float64) (red, green, blue, alpha uint8) {
	if coverage <= 0 || color.Alpha == 0 {
		return 0, 0, 0, 0
	}
	scale := float64(color.Alpha) / 255 * coverage
	return toByte(float64(color.Red) / 255 * scale),
		toByte(float64(color.Green) / 255 * scale),
		toByte(float64(color.Blue) / 255 * scale),
		toByte(scale)
}

func toByte(value float64) uint8 {
	if value <= 0 {
		return 0
	}
	if value >= 1 {
		return 255
	}
	return uint8(math.Round(value * 255))
}

func unmultiply(value, alpha uint8) uint8 {
	if alpha == 0 {
		return 0
	}
	scaled := float64(value) * 255 / float64(alpha)
	if scaled >= 255 {
		return 255
	}
	return uint8(math.Round(scaled))
}
