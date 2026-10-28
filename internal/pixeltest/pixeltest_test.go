package pixeltest

import (
	"image"
	"image/color"
	"testing"
)

func TestIdenticalImages(t *testing.T) {
	first := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	second := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	first.SetNRGBA(0, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	second.SetNRGBA(0, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	result, err := Compare(first, second, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Different != 0 {
		t.Errorf("different = %d, want 0", result.Different)
	}
}

func TestToleranceAndBudget(t *testing.T) {
	first := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	second := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	first.SetNRGBA(0, 0, color.NRGBA{A: 255})
	second.SetNRGBA(0, 0, color.NRGBA{R: 4, A: 255})
	first.SetNRGBA(1, 0, color.NRGBA{A: 255})
	second.SetNRGBA(1, 0, color.NRGBA{R: 40, A: 255})

	result, err := Compare(first, second, Options{ChannelTolerance: 5})
	if err != nil {
		t.Fatal(err)
	}
	if result.Different != 1 {
		t.Errorf("different = %d, want 1", result.Different)
	}
	if result.MaxChannelDelta != 40 {
		t.Errorf("max delta = %d, want 40", result.MaxChannelDelta)
	}
	if result.OK(Options{MaxDifferent: 1}) != true {
		t.Error("expected result within differing-pixel budget")
	}
	if result.OK(Options{}) != false {
		t.Error("expected result outside default budget")
	}
}

func TestSizeMismatch(t *testing.T) {
	first := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	second := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	if _, err := Compare(first, second, Options{}); err == nil {
		t.Fatal("expected a size mismatch error")
	}
}
