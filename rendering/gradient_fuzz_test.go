package rendering

import (
	"testing"

	"gonetsurf/parsers/css/parser"
)

func FuzzGradient(f *testing.F) {
	for _, seed := range []string{
		"linear-gradient(red, blue)",
		"linear-gradient(to right, red 0%, blue 100%)",
		"repeating-linear-gradient(45deg, red 0 10px, blue 10px 20px)",
		"radial-gradient(circle at center, red, blue)",
		"radial-gradient(ellipse 20px 10px at 30% 40%, red, blue)",
		"conic-gradient(from 45deg at center, red, blue)",
		"repeating-conic-gradient(red 0 15deg, white 15deg 30deg)",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, source string) {
		values := parser.ParseComponentValueList(source)
		var gradient *gradientSpec
		for _, value := range values {
			if function, ok := value.(*parser.Function); ok {
				parsed, ok := parseGradientFunction(function, NewColor(10, 20, 30))
				if ok {
					gradient = parsed
				}
			}
		}
		if gradient == nil {
			return
		}
		surface := NewSurface(32, 32)
		painter := NewPainter(nil)
		painter.paintGradient(gradient, surface, Rect{Width: 32, Height: 32}, Rect{Width: 32, Height: 32}, nil)
	})
}
