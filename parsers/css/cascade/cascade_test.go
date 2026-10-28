package cascade

import (
	"testing"

	"gonetsurf/parsers/css/parser"
	"gonetsurf/parsers/css/selectors"
)

func value(input string) []parser.ComponentValue {
	return parser.ParseComponentValueList(input)
}

func TestOriginAndImportance(t *testing.T) {
	declarations := []Declaration{
		{Property: "color", Value: value("red"), Origin: OriginAuthor, Order: 0},
		{Property: "color", Value: value("blue"), Origin: OriginUserAgent, Order: 1},
		{Property: "color", Value: value("green"), Origin: OriginUser, Order: 2},
	}
	result := Cascade(declarations)
	winner, _ := result.Get("color")
	if got := parser.SerializeValues(winner.Value); got != "red" {
		t.Errorf("normal cascade winner = %q, want red (author beats user)", got)
	}

	declarations = append(declarations, Declaration{Property: "color", Value: value("black"), Origin: OriginUserAgent, Important: true, Order: 3})
	result = Cascade(declarations)
	winner, _ = result.Get("color")
	if got := parser.SerializeValues(winner.Value); got != "black" {
		t.Errorf("important cascade winner = %q, want black", got)
	}
}

func TestSpecificityBeatsOrder(t *testing.T) {
	declarations := []Declaration{
		{Property: "color", Value: value("red"), Origin: OriginAuthor, Specificity: selectors.Specificity{A: 1}, Order: 0},
		{Property: "color", Value: value("blue"), Origin: OriginAuthor, Specificity: selectors.Specificity{C: 1}, Order: 1},
	}
	result := Cascade(declarations)
	winner, _ := result.Get("color")
	if got := parser.SerializeValues(winner.Value); got != "red" {
		t.Errorf("high-specificity winner = %q, want red", got)
	}
}

func TestOrderBreaksTies(t *testing.T) {
	declarations := []Declaration{
		{Property: "color", Value: value("red"), Order: 0},
		{Property: "color", Value: value("blue"), Order: 1},
	}
	result := Cascade(declarations)
	winner, _ := result.Get("color")
	if got := parser.SerializeValues(winner.Value); got != "blue" {
		t.Errorf("later winner = %q, want blue", got)
	}
}

func TestLayerOrder(t *testing.T) {
	declarations := []Declaration{
		{Property: "color", Value: value("red"), Origin: OriginAuthor, Layer: 0, Order: 0},
		{Property: "color", Value: value("blue"), Origin: OriginAuthor, Layer: 1, Order: 0},
	}
	result := Cascade(declarations)
	winner, _ := result.Get("color")
	if got := parser.SerializeValues(winner.Value); got != "blue" {
		t.Errorf("later-layer winner = %q, want blue", got)
	}
}

func TestResolveInheritance(t *testing.T) {
	result := Cascade(nil)
	parent := value("red")
	if got := parser.SerializeValues(result.Resolve("color", parent, value("black"))); got != "red" {
		t.Errorf("inherited color = %q, want red", got)
	}
	if got := parser.SerializeValues(result.Resolve("display", parent, value("inline"))); got != "inline" {
		t.Errorf("non-inherited display = %q, want inline", got)
	}

	declarations := []Declaration{{Property: "color", Value: value("inherit")}}
	result = Cascade(declarations)
	if got := parser.SerializeValues(result.Resolve("color", parent, value("black"))); got != "red" {
		t.Errorf("inherit keyword = %q, want red", got)
	}
}

func TestCustomProperty(t *testing.T) {
	if !IsCustom("--brand") {
		t.Error("--brand should be custom")
	}
	if IsCustom("color") {
		t.Error("color should not be custom")
	}
}

func serializeValues(values []parser.ComponentValue) string {
	return parser.SerializeValues(values)
}

func TestCustomPropertyInheritance(t *testing.T) {
	declarations := []Declaration{{Property: "--brand", Value: value("green")}}
	result := Cascade(declarations)
	if got := serializeValues(result.ResolveWithCustom("--brand", nil, nil)); got != "green" {
		t.Errorf("custom value = %q, want green", got)
	}
	parent := InheritedValues{"--brand": value("red")}
	if got := serializeValues(result.ResolveWithCustom("--unknown", parent, nil)); got != "" {
		t.Errorf("unknown custom value = %q, want empty", got)
	}
	inherited := Cascade(nil).ResolveWithCustom("--brand", parent, nil)
	if got := serializeValues(inherited); got != "red" {
		t.Errorf("inherited custom value = %q, want red", got)
	}
	initial := Cascade([]Declaration{{Property: "--brand", Value: value("initial")}}).ResolveWithCustom("--brand", parent, nil)
	if initial != nil {
		t.Errorf("initial custom value = %#v, want nil", initial)
	}
}

func TestVarSubstitution(t *testing.T) {
	declarations := []Declaration{
		{Property: "--brand", Value: value("green")},
		{Property: "color", Value: value("var(--brand)")},
	}
	result := Cascade(declarations)
	if got := serializeValues(result.Resolve("color", nil, value("black"))); got != "green" {
		t.Errorf("var substitution = %q, want green", got)
	}
	fallback := Cascade([]Declaration{{Property: "color", Value: value("var(--missing, red)")}})
	if got := serializeValues(fallback.Resolve("color", nil, value("black"))); got != "red" {
		t.Errorf("fallback = %q, want red", got)
	}
	invalid := Cascade([]Declaration{{Property: "color", Value: value("var(--missing)")}})
	got, ok := invalid.ResolveChecked("color", nil, value("black"))
	if ok {
		t.Error("expected invalid var() to be reported")
	}
	if got := serializeValues(got); got != "black" {
		t.Errorf("invalid var() fallback = %q, want black", got)
	}
	nested := Cascade([]Declaration{
		{Property: "--a", Value: value("var(--b)")},
		{Property: "--b", Value: value("blue")},
		{Property: "color", Value: value("var(--a)")},
	})
	if got := serializeValues(nested.Resolve("color", nil, value("black"))); got != "blue" {
		t.Errorf("nested var = %q, want blue", got)
	}
	cycle := Cascade([]Declaration{
		{Property: "--a", Value: value("var(--b)")},
		{Property: "--b", Value: value("var(--a)")},
		{Property: "color", Value: value("var(--a, red)")},
	})
	if got := serializeValues(cycle.Resolve("color", nil, value("black"))); got != "red" {
		t.Errorf("cycle fallback = %q, want red", got)
	}
	inCalc := Cascade([]Declaration{
		{Property: "--x", Value: value("10px")},
		{Property: "width", Value: value("calc(var(--x) + 5px)")},
	})
	if got := serializeValues(inCalc.Resolve("width", nil, value("auto"))); got != "calc(10px + 5px)" {
		t.Errorf("var in calc = %q, want calc(10px + 5px)", got)
	}
}

func TestAllShorthand(t *testing.T) {
	result := Cascade([]Declaration{
		{Property: "color", Value: value("red"), Order: 0},
		{Property: "all", Value: value("initial"), Order: 1},
	})
	if got := serializeValues(result.Resolve("color", value("green"), value("black"))); got != "black" {
		t.Errorf("all initial = %q, want black", got)
	}
	result = Cascade([]Declaration{
		{Property: "color", Value: value("red"), Order: 1},
		{Property: "all", Value: value("initial"), Order: 0},
	})
	if got := serializeValues(result.Resolve("color", nil, value("black"))); got != "red" {
		t.Errorf("property over all = %q, want red", got)
	}
	inherit := Cascade([]Declaration{{Property: "all", Value: value("inherit")}})
	if got := serializeValues(inherit.Resolve("color", value("green"), value("black"))); got != "green" {
		t.Errorf("all inherit = %q, want green", got)
	}
	custom := Cascade([]Declaration{
		{Property: "all", Value: value("initial")},
		{Property: "--brand", Value: value("green")},
	})
	if got := serializeValues(custom.ResolveWithCustom("--brand", nil, nil)); got != "green" {
		t.Errorf("all should not affect custom properties: %q", got)
	}
}

func TestRevertAndRevertLayer(t *testing.T) {
	result := Cascade([]Declaration{
		{Property: "color", Value: value("blue"), Origin: OriginUserAgent, Order: 0},
		{Property: "color", Value: value("revert"), Origin: OriginAuthor, Order: 1},
	})
	if got := serializeValues(result.Resolve("color", nil, value("black"))); got != "blue" {
		t.Errorf("revert = %q, want blue", got)
	}
	result = Cascade([]Declaration{
		{Property: "color", Value: value("blue"), Origin: OriginUserAgent, Order: 0},
		{Property: "color", Value: value("green"), Origin: OriginUser, Order: 1},
		{Property: "color", Value: value("revert"), Origin: OriginAuthor, Order: 2},
	})
	if got := serializeValues(result.Resolve("color", nil, value("black"))); got != "green" {
		t.Errorf("revert to user = %q, want green", got)
	}
	layers := Cascade([]Declaration{
		{Property: "color", Value: value("red"), Origin: OriginAuthor, Layer: 0, Order: 0},
		{Property: "color", Value: value("blue"), Origin: OriginAuthor, Layer: 1, Order: 1},
		{Property: "color", Value: value("revert-layer"), Origin: OriginAuthor, Layer: 2, Order: 2},
	})
	if got := serializeValues(layers.Resolve("color", nil, value("black"))); got != "blue" {
		t.Errorf("revert-layer = %q, want blue", got)
	}
}

func TestAllInvalidCustomCycle(t *testing.T) {
	result := Cascade([]Declaration{
		{Property: "--a", Value: value("var(--b)")},
		{Property: "--b", Value: value("var(--a)")},
	})
	value, ok := result.ResolveChecked("--a", nil, nil)
	if ok || value != nil {
		t.Errorf("cyclic custom property = %#v, %v; want nil, false", value, ok)
	}
}

func TestRevertImportantLayer(t *testing.T) {
	result := Cascade([]Declaration{
		{Property: "color", Value: value("red"), Origin: OriginAuthor, Layer: 0, Important: true, Order: 0},
		{Property: "color", Value: value("blue"), Origin: OriginAuthor, Layer: 1, Important: true, Order: 1},
		{Property: "color", Value: value("revert-layer"), Origin: OriginAuthor, Layer: 2, Important: true, Order: 2},
	})
	if got := serializeValues(result.Resolve("color", nil, value("black"))); got != "red" {
		t.Errorf("important revert-layer = %q, want red", got)
	}
}

func TestResolveInheritedWithoutParent(t *testing.T) {
	result := Cascade(nil)
	if got := serializeValues(result.Resolve("color", nil, value("black"))); got != "black" {
		t.Errorf("inherited property without parent = %q, want initial black", got)
	}
}
