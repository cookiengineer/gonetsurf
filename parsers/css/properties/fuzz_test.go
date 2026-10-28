package properties

import "testing"

// FuzzValidate checks that arbitrary values never panic a property matcher.
func FuzzValidate(f *testing.F) {
	properties := []string{
		"color", "display", "width", "height", "margin", "padding",
		"background", "font", "animation", "transition", "box-shadow",
	}
	seeds := []string{
		"", "red", "10px", "calc(1em + 2px)", "none",
		"var(--x, 10px)", "url(foo.png)", "linear-gradient(red, blue)",
	}
	for _, property := range properties {
		for _, seed := range seeds {
			f.Add(property, seed)
		}
	}
	f.Fuzz(func(t *testing.T, property, value string) {
		_, _ = Validate(property, value)
	})
}
