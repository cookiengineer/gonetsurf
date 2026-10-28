package grammar

import (
	"testing"

	"gonetsurf/parsers/css/parser"
	"gonetsurf/parsers/css/tokenizer"
)

func testContext() *Context {
	context := NewContext()
	context.Types["length"] = func(values []parser.ComponentValue) int {
		token, ok := parser.AsToken(firstValue(values))
		if !ok {
			return 0
		}
		if token.Type == tokenizer.TokenDimension {
			switch token.Value {
			case "px", "em", "rem", "vh", "vw":
				return 1
			}
		}
		if token.Type == tokenizer.TokenNumber && token.Number == 0 {
			return 1
		}
		return 0
	}
	context.Types["angle"] = func(values []parser.ComponentValue) int {
		token, ok := parser.AsToken(firstValue(values))
		if ok && token.Type == tokenizer.TokenDimension {
			switch token.Value {
			case "deg", "rad", "turn":
				return 1
			}
		}
		return 0
	}
	context.Types["percentage"] = func(values []parser.ComponentValue) int {
		token, ok := parser.AsToken(firstValue(values))
		if ok && token.Type == tokenizer.TokenPercentage {
			return 1
		}
		return 0
	}
	context.Types["number"] = func(values []parser.ComponentValue) int {
		token, ok := parser.AsToken(firstValue(values))
		if ok && token.Type == tokenizer.TokenNumber {
			return 1
		}
		return 0
	}
	context.Types["color"] = func(values []parser.ComponentValue) int {
		token, ok := parser.AsToken(firstValue(values))
		if !ok || token.Type != tokenizer.TokenIdent {
			return 0
		}
		switch token.Value {
		case "red", "green", "blue":
			return 1
		}
		return 0
	}
	context.Properties["width"] = "<length> | <percentage> | auto"
	return context
}

func firstValue(values []parser.ComponentValue) parser.ComponentValue {
	if len(values) == 0 {
		return nil
	}
	return values[0]
}

func match(t *testing.T, definition, input string) (*Match, bool) {
	t.Helper()
	result, err := MatchDefinition(definition, parser.ParseComponentValueList(input), testContext())
	if err != nil {
		t.Fatalf("parse grammar %q: %v", definition, err)
	}
	return result, result != nil
}

func TestMatchSimpleAlternatives(t *testing.T) {
	if _, ok := match(t, "<length> | auto", "10px"); !ok {
		t.Error("expected 10px to match")
	}
	if _, ok := match(t, "<length> | auto", "auto"); !ok {
		t.Error("expected auto to match")
	}
	if _, ok := match(t, "<length> | auto", "10"); ok {
		t.Error("expected 10 to fail")
	}
}

func TestMatchSequence(t *testing.T) {
	if _, ok := match(t, "<length> <percentage>", "10px 50%"); !ok {
		t.Error("expected 10px 50% to match")
	}
	if _, ok := match(t, "<length> <percentage>", "50% 10px"); ok {
		t.Error("expected reversed order to fail")
	}
}

func TestMatchRepeat(t *testing.T) {
	if _, ok := match(t, "[ <length> | <percentage> ]{1,4}", "10px 20% 30px"); !ok {
		t.Error("expected three values to match")
	}
	if _, ok := match(t, "[ <length> | <percentage> ]{1,4}", "10px 20% 30px 40px 50px"); ok {
		t.Error("expected five values to fail")
	}
	if _, ok := match(t, "<length>{2}", "10px 20px"); !ok {
		t.Error("expected two lengths to match")
	}
	if _, ok := match(t, "<length>{2}", "10px"); ok {
		t.Error("expected one length to fail")
	}
}

func TestMatchCommaSeparated(t *testing.T) {
	if _, ok := match(t, "<length>#", "10px, 20px, 30px"); !ok {
		t.Error("expected comma-separated lengths to match")
	}
	if _, ok := match(t, "<length>#", "10px 20px"); ok {
		t.Error("expected missing commas to fail")
	}
}

func TestMatchAnyOrder(t *testing.T) {
	if _, ok := match(t, "<length> || auto", "auto 10px"); !ok {
		t.Error("expected any-order match")
	}
	if _, ok := match(t, "<length> || auto", "10px auto"); !ok {
		t.Error("expected any-order match")
	}
	if _, ok := match(t, "<length> || auto", "10px"); !ok {
		t.Error("expected one item to satisfy ||")
	}
}

func TestMatchAllAnyOrder(t *testing.T) {
	if _, ok := match(t, "<length> && <percentage>", "50% 10px"); !ok {
		t.Error("expected all-any-order match")
	}
	if _, ok := match(t, "<length> && <percentage>", "10px"); ok {
		t.Error("expected missing item to fail &&")
	}
}

func TestMatchRequiredGroup(t *testing.T) {
	if _, ok := match(t, "[ <length> | auto ]!", "auto"); !ok {
		t.Error("expected required group to match")
	}
	if _, ok := match(t, "[ <length>? ]!", "10px"); !ok {
		t.Error("expected required group to match a present value")
	}
	if _, ok := match(t, "[ <length>? ]!", ""); ok {
		t.Error("expected empty required group to fail")
	}
}

func TestMatchPropertyReference(t *testing.T) {
	result, ok := match(t, "<'width'>", "auto")
	if !ok {
		t.Fatal("expected property reference to match")
	}
	if len(result.Captures["width"]) == 0 {
		t.Error("expected width capture")
	}
}

func TestMatchCaptures(t *testing.T) {
	result, ok := match(t, "<length> <percentage>", "10px 50%")
	if !ok {
		t.Fatal("expected match")
	}
	if len(result.Captures["length"]) != 1 || len(result.Captures["percentage"]) != 1 {
		t.Errorf("captures = %#v", result.Captures)
	}
	if len(result.Values) != 3 {
		t.Errorf("consumed %d values, want 3", len(result.Values))
	}
}

func TestParseStructure(t *testing.T) {
	expression, err := Parse("a b | c")
	if err != nil {
		t.Fatal(err)
	}
	oneOf, ok := expression.(*OneOf)
	if !ok || len(oneOf.Items) != 2 {
		t.Fatalf("expression = %#v", expression)
	}
	if _, ok := oneOf.Items[0].(*Sequence); !ok {
		t.Errorf("first alternative = %T, want *Sequence", oneOf.Items[0])
	}
}

func TestMatchOptionalAndRepetition(t *testing.T) {
	tests := []struct {
		definition string
		input      string
		want       bool
	}{
		{"<length>?", "", true},
		{"<length>?", "10px", true},
		{"<length>*", "", true},
		{"<length>*", "10px 20px", true},
		{"<length>+", "", false},
		{"<length>+", "10px", true},
		{"<length>{0,2}", "", true},
		{"<length>{0,2}", "10px 20px", true},
		{"<length>{0,2}", "10px 20px 30px", false},
		{"[ <length># ]?", "", true},
		{"[ <length># ]?", "10px, 20px", true},
	}
	for _, test := range tests {
		if _, ok := match(t, test.definition, test.input); ok != test.want {
			t.Errorf("match(%q, %q) = %v, want %v", test.definition, test.input, ok, test.want)
		}
	}
}

func TestMatchAnyOrderRejectsDuplicates(t *testing.T) {
	if _, ok := match(t, "<length> || auto", "auto auto"); ok {
		t.Error("expected duplicate auto to fail ||")
	}
	if _, ok := match(t, "<length> && <percentage>", "10px 50% 20%"); ok {
		t.Error("expected duplicate percentage to fail &&")
	}
}

func TestMatchOptionalInAnyOrder(t *testing.T) {
	if _, ok := match(t, "<length> || auto?", "10px"); !ok {
		t.Error("expected optional term in || to be absent")
	}
	if _, ok := match(t, "<length> || auto?", "auto"); !ok {
		t.Error("expected optional term in || to be present")
	}
}

func TestMatchRangeAndDelimiters(t *testing.T) {
	if _, ok := match(t, "[a | b]!", "a"); !ok {
		t.Error("expected required group to match")
	}
	if _, ok := match(t, "a / b", "a / b"); !ok {
		t.Error("expected literal delimiter to match")
	}
	if _, ok := match(t, "a / b", "a b"); ok {
		t.Error("expected missing delimiter to fail")
	}
}

func TestParseErrors(t *testing.T) {
	if _, err := Parse("[ a b"); err == nil {
		t.Error("expected missing bracket error")
	}
	if _, err := Parse("<length"); err == nil {
		t.Error("expected missing > error")
	}
	if _, err := Parse("<length>{2,1}"); err == nil {
		t.Error("expected invalid range error")
	}
}

func TestMatchNumericRanges(t *testing.T) {
	tests := []struct {
		definition string
		input      string
		want       bool
	}{
		{"<length [0,∞]>", "10px", true},
		{"<length [0,∞]>", "0", true},
		{"<length [0,∞]>", "-10px", false},
		{"<number [1,1000]>", "1", true},
		{"<number [1,1000]>", "1000", true},
		{"<number [1,1000]>", "0", false},
		{"<number [1,1000]>", "1001", false},
		{"<angle [-90deg,90deg]>", "0deg", true},
		{"<angle [-90deg,90deg]>", "-90deg", true},
		{"<angle [-90deg,90deg]>", "91deg", false},
		{"<number (0,1)>", "0", false},
		{"<number (0,1)>", "1", false},
		{"<number (0,1)>", "0.5", true},
	}
	for _, test := range tests {
		if _, ok := match(t, test.definition, test.input); ok != test.want {
			t.Errorf("match(%q, %q) = %v, want %v", test.definition, test.input, ok, test.want)
		}
	}
}
