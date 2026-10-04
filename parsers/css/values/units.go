package values

import (
	"strings"

	"gonetsurf/parsers/css/parser"
	"gonetsurf/parsers/css/tokenizer"
)

// UnitKind classifies a CSS dimension's unit.
type UnitKind uint8

const (
	// Length is a distance unit.
	Length UnitKind = iota
	// Angle is an angle unit.
	Angle
	// Time is a duration unit.
	Time
	// Frequency is a frequency unit.
	Frequency
	// Resolution is an image resolution unit.
	Resolution
	// Flex is the fr unit.
	Flex
	// Other marks an unknown unit.
	Other
)

// Value is a typed numeric CSS value.
type Value struct {
	Kind   UnitKind
	Number float64
	Unit   string
}

var lengthUnits = map[string]float64{
	"px": 1,
	"cm": 96.0 / 2.54,
	"mm": 96.0 / 25.4,
	"q":  96.0 / 101.6,
	"in": 96,
	"pc": 16,
	"pt": 96.0 / 72.0,
	"em": 16, "rem": 16, "ex": 8, "ch": 8, "cap": 8, "ic": 16,
	"lh": 16, "rlh": 16,
	"vw": 16, "vh": 16, "vmin": 16, "vmax": 16, "vb": 16, "vi": 16,
	"svw": 16, "svh": 16, "lvw": 16, "lvh": 16, "dvw": 16, "dvh": 16,
	"cqw": 16, "cqh": 16, "cqi": 16, "cqb": 16, "cqmin": 16, "cqmax": 16,
}

var angleUnits = map[string]float64{
	"deg":  1,
	"grad": 360.0 / 400.0,
	"rad":  180.0 / 3.141592653589793,
	"turn": 360,
}

var timeUnits = map[string]float64{
	"s":  1,
	"ms": 0.001,
}

var frequencyUnits = map[string]float64{
	"hz":  1,
	"khz": 1000,
}

var resolutionUnits = map[string]float64{
	"dpi":  1,
	"dpcm": 2.54,
	"dppx": 96,
	"x":    96,
}

// ClassifyUnit returns the kind and the canonical lowercase unit.
func ClassifyUnit(unit string) (UnitKind, string, bool) {
	lower := strings.ToLower(unit)
	if _, ok := lengthUnits[lower]; ok {
		return Length, lower, true
	}
	if _, ok := angleUnits[lower]; ok {
		return Angle, lower, true
	}
	if _, ok := timeUnits[lower]; ok {
		return Time, lower, true
	}
	if _, ok := frequencyUnits[lower]; ok {
		return Frequency, lower, true
	}
	if _, ok := resolutionUnits[lower]; ok {
		return Resolution, lower, true
	}
	if lower == "fr" {
		return Flex, lower, true
	}
	return Other, lower, false
}

// ParseNumeric parses a single number, percentage, or dimension token into a
// Value. It returns false for any other token.
func ParseNumeric(value parser.ComponentValue) (Value, bool) {
	token, ok := parser.AsToken(value)
	if !ok {
		return Value{}, false
	}
	switch token.Type {
	case tokenizer.TokenNumber:
		return Value{Kind: Other, Number: token.Number}, true
	case tokenizer.TokenPercentage:
		return Value{Kind: Other, Number: token.Number / 100, Unit: "%"}, true
	case tokenizer.TokenDimension:
		kind, unit, valid := ClassifyUnit(token.Value)
		if !valid {
			return Value{}, false
		}
		return Value{Kind: kind, Number: token.Number, Unit: unit}, true
	default:
		return Value{}, false
	}
}

// ToPixels converts a length Value to pixels using a 16px font and a viewport
// of the given size.
func ToPixels(value Value, viewportWidth, viewportHeight float64) (float64, bool) {
	if value.Kind != Length {
		return 0, false
	}
	factor, ok := lengthUnits[value.Unit]
	if !ok {
		return 0, false
	}
	switch value.Unit {
	case "vw", "svw", "lvw", "dvw":
		return value.Number * viewportWidth / 100, true
	case "vh", "svh", "lvh", "dvh":
		return value.Number * viewportHeight / 100, true
	case "vmin", "svmin", "lvmin", "dvmin":
		smaller := viewportWidth
		if viewportHeight < smaller {
			smaller = viewportHeight
		}
		return value.Number * smaller / 100, true
	case "vmax", "svmax", "lvmax", "dvmax":
		larger := viewportWidth
		if viewportHeight > larger {
			larger = viewportHeight
		}
		return value.Number * larger / 100, true
	default:
		return value.Number * factor, true
	}
}

// ToDegrees converts an angle Value to degrees.
func ToDegrees(value Value) (float64, bool) {
	if value.Kind != Angle {
		return 0, false
	}
	return value.Number * angleUnits[value.Unit], true
}

// ToSeconds converts a time Value to seconds.
func ToSeconds(value Value) (float64, bool) {
	if value.Kind != Time {
		return 0, false
	}
	return value.Number * timeUnits[value.Unit], true
}
