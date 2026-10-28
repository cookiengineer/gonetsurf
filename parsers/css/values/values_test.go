package values

import (
	"encoding/json"
	"math"
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

func TestMathFunctions(t *testing.T) {
	tests := []struct {
		input  string
		kind   UnitKind
		number float64
		unit   string
	}{
		{"round(10px, 3px)", Length, 9, "px"},
		{"round(up, 10px, 3px)", Length, 12, "px"},
		{"round(down, 10px, 3px)", Length, 9, "px"},
		{"round(to-zero, -10px, 3px)", Length, -9, "px"},
		{"mod(10, 3)", Other, 1, ""},
		{"rem(-10, 3)", Other, -1, ""},
		{"abs(-5px)", Length, 5, "px"},
		{"sign(-5px)", Other, -1, ""},
		{"hypot(3px, 4px)", Length, 5, "px"},
		{"pow(2, 3)", Other, 8, ""},
		{"sqrt(9)", Other, 3, ""},
		{"log(8, 2)", Other, 3, ""},
		{"exp(0)", Other, 1, ""},
		{"sin(30deg)", Other, 0.5, ""},
		{"cos(0deg)", Other, 1, ""},
		{"tan(45deg)", Other, 1, ""},
		{"asin(0.5)", Angle, 30, "deg"},
		{"acos(1)", Angle, 0, "deg"},
		{"atan(1)", Angle, 45, "deg"},
		{"atan2(1, 1)", Angle, 45, "deg"},
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
		if value.Kind != test.kind || value.Unit != test.unit || math.Abs(value.Number-test.number) > 1e-9 {
			t.Errorf("Simplify(%q) = %v%s (%v), want %v%s", test.input, value.Number, value.Unit, value.Kind, test.number, test.unit)
		}
	}
}

func TestMathFunctionsRejectInvalidArguments(t *testing.T) {
	for _, input := range []string{
		"round(up, 10px)",
		"round(10px, 0px)",
		"mod(10, 0)",
		"sqrt(-1)",
		"log(0)",
		"pow(2px, 2)",
		"sin(red)",
	} {
		function := parseMathFunction(t, input)
		expression, ok := ParseMath(function)
		if !ok {
			continue
		}
		if _, simplified := expression.Simplify(); simplified {
			t.Errorf("Simplify(%q) unexpectedly succeeded", input)
		}
	}
}

type mathCase struct {
	Input  string  `json:"input"`
	Kind   string  `json:"kind"`
	Number float64 `json:"number"`
	Unit   string  `json:"unit"`
}

func TestMathConformance(t *testing.T) {
	encoded, err := os.ReadFile("testdata/math.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []mathCase
	if err := json.Unmarshal(encoded, &cases); err != nil {
		t.Fatal(err)
	}
	kinds := map[string]UnitKind{
		"Length": Length,
		"Angle":  Angle,
		"Other":  Other,
	}
	for _, testCase := range cases {
		function := parseMathFunction(t, testCase.Input)
		expression, ok := ParseMath(function)
		if !ok {
			t.Errorf("ParseMath(%q) failed", testCase.Input)
			continue
		}
		value, ok := expression.Simplify()
		if !ok {
			t.Errorf("Simplify(%q) failed", testCase.Input)
			continue
		}
		if value.Kind != kinds[testCase.Kind] || value.Unit != testCase.Unit || math.Abs(value.Number-testCase.Number) > 1e-9 {
			t.Errorf("Simplify(%q) = %v%s (%v), want %v%s", testCase.Input, value.Number, value.Unit, value.Kind, testCase.Number, testCase.Unit)
		}
	}
}

func TestUnitConversions(t *testing.T) {
	lengths := []struct {
		unit  string
		value float64
		want  float64
	}{
		{"px", 1, 1},
		{"in", 1, 96},
		{"cm", 2.54, 96},
		{"mm", 25.4, 96},
		{"q", 101.6, 96},
		{"pc", 1, 16},
		{"pt", 72, 96},
	}
	for _, test := range lengths {
		got, ok := ToPixels(Value{Kind: Length, Number: test.value, Unit: test.unit}, 0, 0)
		if !ok || math.Abs(got-test.want) > 1e-9 {
			t.Errorf("ToPixels(%v%s) = %v, want %v", test.value, test.unit, got, test.want)
		}
	}
	angles := []struct {
		unit  string
		value float64
		want  float64
	}{
		{"deg", 180, 180},
		{"grad", 200, 180},
		{"rad", math.Pi, 180},
		{"turn", 0.5, 180},
	}
	for _, test := range angles {
		got, ok := ToDegrees(Value{Kind: Angle, Number: test.value, Unit: test.unit})
		if !ok || math.Abs(got-test.want) > 1e-9 {
			t.Errorf("ToDegrees(%v%s) = %v, want %v", test.value, test.unit, got, test.want)
		}
	}
	if got, ok := ToSeconds(Value{Kind: Time, Number: 500, Unit: "ms"}); !ok || math.Abs(got-0.5) > 1e-9 {
		t.Errorf("ToSeconds(500ms) = %v", got)
	}
}
