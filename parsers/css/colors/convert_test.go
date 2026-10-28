package colors

import (
	"math"
	"testing"
)

func requireClose(t *testing.T, name string, got, want, tolerance float64) {
	t.Helper()
	if math.Abs(got-want) > tolerance {
		t.Errorf("%s = %v, want %v (±%v)", name, got, want, tolerance)
	}
}

func requireSRGB(t *testing.T, input string, red, green, blue, alpha float64) {
	t.Helper()
	parsed, ok := Parse(input)
	if !ok {
		t.Fatalf("Parse(%q) failed", input)
	}
	converted, ok := parsed.ToSRGB()
	if !ok {
		t.Fatalf("ToSRGB(%q) failed", input)
	}
	requireClose(t, input+" red", converted.Red, red, 0.002)
	requireClose(t, input+" green", converted.Green, green, 0.002)
	requireClose(t, input+" blue", converted.Blue, blue, 0.002)
	requireClose(t, input+" alpha", converted.Alpha, alpha, 0.002)
}

func TestToSRGBPrimaryVectors(t *testing.T) {
	requireSRGB(t, "rgb(255 0 0)", 1, 0, 0, 1)
	requireSRGB(t, "rgb(0 255 0)", 0, 1, 0, 1)
	requireSRGB(t, "rgb(0 0 255)", 0, 0, 1, 1)
	requireSRGB(t, "hsl(120 100% 50%)", 0, 1, 0, 1)
	requireSRGB(t, "hwb(240 0% 0%)", 0, 0, 1, 1)
	requireSRGB(t, "white", 1, 1, 1, 1)
	requireSRGB(t, "transparent", 0, 0, 0, 0)
}

func TestToSRGBLabAndOKLab(t *testing.T) {
	requireSRGB(t, "lab(50 0 0)", 0.4663266, 0.4663266, 0.4663266, 1)
	requireSRGB(t, "lab(100 0 0)", 1, 1, 1, 1)
	requireSRGB(t, "lch(50 0 120)", 0.4663266, 0.4663266, 0.4663266, 1)
	requireSRGB(t, "oklab(0.5 0 0)", 0.3885728, 0.3885728, 0.3885728, 1)
	requireSRGB(t, "oklch(0.5 0 120)", 0.3885728, 0.3885728, 0.3885728, 1)
}

func TestToSRGBColorFunction(t *testing.T) {
	requireSRGB(t, "color(srgb 1 0 0)", 1, 0, 0, 1)
	requireSRGB(t, "color(srgb-linear 1 0 0)", 1, 0, 0, 1)
	requireSRGB(t, "color(xyz-d65 0.95047 1 1.08883)", 1, 1, 1, 1)
	requireSRGB(t, "color(display-p3 1 0 0)", 1.0930663, -0.5433758, -0.2537172, 1)
	requireSRGB(t, "color(display-p3 0 1 0)", -2.9062270, 1.0182656, -1.0159777, 1)
}

func TestToSRGBAlpha(t *testing.T) {
	requireSRGB(t, "rgb(255 0 0 / 50%)", 1, 0, 0, 0.5)
	requireSRGB(t, "lab(50 0 0 / 0.25)", 0.4663266, 0.4663266, 0.4663266, 0.25)
}

func TestRGBA8GamutMaps(t *testing.T) {
	parsed, ok := Parse("color(display-p3 1 0 0)")
	if !ok {
		t.Fatal("parse failed")
	}
	mapped, ok := parsed.ToSRGBGamutMapped()
	if !ok {
		t.Fatal("gamut mapping failed")
	}
	if !inSRGBGamut(mapped) {
		t.Errorf("mapped color is outside sRGB: %#v", mapped)
	}
	if mapped.Red < 0.9 || mapped.Green > 0.4 || mapped.Blue > 0.4 {
		t.Errorf("mapped P3 red = %#v", mapped)
	}
	red, green, blue, alpha, ok := parsed.RGBA8()
	if !ok {
		t.Fatal("RGBA8 failed")
	}
	if red != 255 || green > 100 || blue > 100 || alpha != 255 {
		t.Errorf("P3 red RGBA8 = (%d,%d,%d,%d)", red, green, blue, alpha)
	}
}

func TestToSRGBWideGamutWhite(t *testing.T) {
	for _, input := range []string{
		"color(a98-rgb 1 1 1)",
		"color(prophoto-rgb 1 1 1)",
		"color(rec2020 1 1 1)",
	} {
		requireSRGB(t, input, 1, 1, 1, 1)
	}
}

func TestToSRGBWideGamutPrimaries(t *testing.T) {
	for _, input := range []string{
		"color(a98-rgb 1 0 0)",
		"color(prophoto-rgb 1 0 0)",
		"color(rec2020 1 0 0)",
	} {
		parsed, ok := Parse(input)
		if !ok {
			t.Fatalf("Parse(%q) failed", input)
		}
		converted, ok := parsed.ToSRGB()
		if !ok {
			t.Fatalf("ToSRGB(%q) failed", input)
		}
		if converted.Red < 0.9 || converted.Green > 0.2 || converted.Blue > 0.2 {
			t.Errorf("%s = %#v, want a red primary", input, converted)
		}
	}
}
