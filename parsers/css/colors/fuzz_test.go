package colors

import "testing"

func FuzzToSRGB(f *testing.F) {
	for _, seed := range []string{
		"",
		"red",
		"rgb(255 0 0)",
		"lab(50 0 0 / 0.5)",
		"oklch(0.7 0.2 120)",
		"color(display-p3 1 0 0)",
		"color(prophoto-rgb 0.5 0.25 0.75)",
		"color(rec2020 0.1 0.2 0.3)",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		parsed, ok := Parse(input)
		if !ok {
			return
		}
		converted, ok := parsed.ToSRGB()
		if !ok {
			return
		}
		if converted.Alpha < 0 || converted.Alpha > 1 {
			t.Errorf("alpha out of range: %v", converted.Alpha)
		}
		_, _, _, _, _ = parsed.RGBA8()
	})
}
