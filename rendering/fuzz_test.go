package rendering

import "testing"

func FuzzRaster(f *testing.F) {
	f.Add([]byte{0x10, 0x10, 0x20, 0x20, 0x00}, byte(0))
	f.Add([]byte{0x08, 0x08, 0x18, 0x08, 0x01, 0x18, 0x18, 0x02}, byte(3))
	f.Fuzz(func(t *testing.T, data []byte, styleByte byte) {
		if len(data) > 128 {
			return
		}
		path := NewPath()
		for index := 0; index+4 < len(data); index += 5 {
			xPos := float64(int8(data[index])) + 32
			yPos := float64(int8(data[index+1])) + 32
			secondX := float64(int8(data[index+2])) + 32
			secondY := float64(int8(data[index+3])) + 32
			switch data[index+4] % 4 {
			case 0:
				path.MoveTo(xPos, yPos)
			case 1:
				path.LineTo(xPos, yPos)
			case 2:
				path.QuadTo(xPos, yPos, secondX, secondY)
			default:
				path.CubicTo(xPos, yPos, secondX, secondY, secondX+1, secondY+1)
			}
		}
		rule := NonZeroFill
		if styleByte%2 == 1 {
			rule = EvenOddFill
		}
		mask := RasterizeFill(path, 64, 64, rule)
		_ = mask.At(0, 0)
		style := StrokeStyle{
			Width:      1 + float64(styleByte%8),
			Cap:        LineCap(styleByte % 3),
			Join:       LineJoin((int(styleByte) / 3) % 3),
			MiterLimit: 10,
		}
		stroke := RasterizeStroke(path, 64, 64, style)
		_ = stroke.At(32, 32)

		surface := NewSurface(64, 64)
		surface.FillPath(path, rule, Color{Red: styleByte, Green: 128, Blue: 64, Alpha: 200})
		surface.StrokePath(path, style, Color{Red: 20, Green: 40, Blue: 60, Alpha: 255})
		_ = surface.Image()
	})
}
