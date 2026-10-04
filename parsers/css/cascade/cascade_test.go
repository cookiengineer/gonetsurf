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
