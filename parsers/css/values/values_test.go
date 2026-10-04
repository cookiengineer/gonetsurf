package values

import (
	"encoding/json"
	"os"
	"testing"

	"gonetsurf/parsers/css/parser"
)

func TestURangeConformance(t *testing.T) {
	encoded, err := os.ReadFile("testdata/urange.json")
	if err != nil {
		t.Fatalf("read corpus: %v", err)
	}
	var items []any
	if err := json.Unmarshal(encoded, &items); err != nil {
		t.Fatalf("decode corpus: %v", err)
	}
	for index := 0; index < len(items); index += 2 {
		input, ok := items[index].(string)
		if !ok {
			t.Fatalf("case %d: input is not a string", index/2)
		}
		expected, ok := items[index+1].([]any)
		if !ok {
			t.Fatalf("case %d: expected is not a list", index/2)
		}
		actual := ParseURangeList(input)
		if len(actual) != len(expected) {
			t.Errorf("ParseURangeList(%q) returned %d ranges, want %d", input, len(actual), len(expected))
			continue
		}
		for rangeIndex, want := range expected {
			if want == nil {
				if actual[rangeIndex] != nil {
					t.Errorf("ParseURangeList(%q)[%d] = %+v, want nil", input, rangeIndex, actual[rangeIndex])
				}
				continue
			}
			pair := want.([]any)
			if actual[rangeIndex] == nil {
				t.Errorf("ParseURangeList(%q)[%d] = nil, want %v", input, rangeIndex, pair)
				continue
			}
			if actual[rangeIndex].Start != int(pair[0].(float64)) || actual[rangeIndex].End != int(pair[1].(float64)) {
				t.Errorf("ParseURangeList(%q)[%d] = %+v, want %v", input, rangeIndex, actual[rangeIndex], pair)
			}
		}
	}
}

func TestClassifyUnit(t *testing.T) {
	tests := []struct {
		unit string
		kind UnitKind
		ok   bool
	}{
		{"px", Length, true},
		{"EM", Length, true},
		{"deg", Angle, true},
		{"ms", Time, true},
		{"kHz", Frequency, true},
		{"dppx", Resolution, true},
		{"fr", Flex, true},
		{"bogus", Other, false},
	}
	for _, test := range tests {
		kind, _, ok := ClassifyUnit(test.unit)
		if kind != test.kind || ok != test.ok {
			t.Errorf("ClassifyUnit(%q) = (%v, %v), want (%v, %v)", test.unit, kind, ok, test.kind, test.ok)
		}
	}
}

func TestToPixels(t *testing.T) {
	value := Value{Kind: Length, Number: 1, Unit: "in"}
	if pixels, ok := ToPixels(value, 0, 0); !ok || pixels != 96 {
		t.Errorf("1in = %v px", pixels)
	}
	value = Value{Kind: Length, Number: 50, Unit: "vw"}
	if pixels, ok := ToPixels(value, 1000, 500); !ok || pixels != 500 {
		t.Errorf("50vw = %v px", pixels)
	}
}

func parseMathFunction(t *testing.T, input string) *parser.Function {
	t.Helper()
	values := parser.ParseComponentValueList(input)
	function, ok := values[0].(*parser.Function)
	if !ok {
		t.Fatalf("%q is not a function", input)
	}
	return function
}

func TestCalcSimplification(t *testing.T) {
	tests := []struct {
		input  string
		number float64
		unit   string
	}{
		{"calc(1px + 2px)", 3, "px"},
		{"calc(10px - 4px)", 6, "px"},
		{"calc(2 * 3px)", 6, "px"},
		{"calc(10px / 2)", 5, "px"},
		{"calc((1px + 2px) * 2)", 6, "px"},
		{"calc(1in + 0px)", 96, "px"},
		{"min(1px, 2px)", 1, "px"},
		{"max(1px, 2px)", 2, "px"},
		{"clamp(1px, 5px, 3px)", 3, "px"},
	}
	for _, test := range tests {
		function := parseMathFunction(t, test.input)
		expression, ok := ParseMath(function)
		if !ok {
			t.Errorf("ParseMath(%q) failed", test.input)
			continue
		}
		value, ok := expression.Simplify()
		if !ok {
			t.Errorf("Simplify(%q) failed", test.input)
			continue
		}
		if value.Number != test.number || value.Unit != test.unit {
			t.Errorf("Simplify(%q) = %v%s, want %v%s", test.input, value.Number, value.Unit, test.number, test.unit)
		}
	}
}

func TestCalcInvalid(t *testing.T) {
	for _, input := range []string{"calc(1px + )", "calc(1px * 2px)"} {
		function := parseMathFunction(t, input)
		if expression, ok := ParseMath(function); ok {
			if _, simplified := expression.Simplify(); simplified {
				t.Errorf("Simplify(%q) unexpectedly succeeded", input)
			}
		}
	}
}
