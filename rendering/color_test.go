package rendering

import "testing"

func TestParseColorConversions(t *testing.T) {
	tests := []struct {
		input string
		want  Color
	}{
		{"rgb(255 0 0)", Color{Red: 255, Alpha: 255}},
		{"hsl(120 100% 50%)", Color{Green: 255, Alpha: 255}},
		{"lab(50 0 0)", Color{Red: 119, Green: 119, Blue: 119, Alpha: 255}},
		{"oklab(0.5 0 0)", Color{Red: 99, Green: 99, Blue: 99, Alpha: 255}},
		{"color(xyz-d65 0.95047 1 1.08883)", Color{Red: 255, Green: 255, Blue: 255, Alpha: 255}},
		{"lab(50 0 0 / 0.5)", Color{Red: 119, Green: 119, Blue: 119, Alpha: 128}},
	}
	for _, test := range tests {
		got, ok := ParseColor(test.input)
		if !ok {
			t.Errorf("ParseColor(%q) failed", test.input)
			continue
		}
		if got != test.want {
			t.Errorf("ParseColor(%q) = %#v, want %#v", test.input, got, test.want)
		}
	}
	gamutMapped, ok := ParseColor("color(display-p3 1 0 0)")
	if !ok {
		t.Fatal("ParseColor(P3 red) failed")
	}
	if gamutMapped.Red != 255 || gamutMapped.Green > 100 || gamutMapped.Blue > 100 {
		t.Errorf("gamut-mapped P3 red = %#v", gamutMapped)
	}
}

func TestParseColorRejectsUnsupportedAndCurrent(t *testing.T) {
	for _, input := range []string{"currentcolor", "not-a-color"} {
		if _, ok := ParseColor(input); ok {
			t.Errorf("ParseColor(%q) unexpectedly succeeded", input)
		}
	}
}
