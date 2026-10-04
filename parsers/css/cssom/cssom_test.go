package cssom

import "testing"

func TestParseStylesheet(t *testing.T) {
	sheet := Parse("a { color: red; margin: 0 !important } b {}")
	if len(sheet.Rules) != 2 {
		t.Fatalf("got %d rules, want 2", len(sheet.Rules))
	}
	style, ok := sheet.Rules[0].(*StyleRule)
	if !ok {
		t.Fatalf("rule 0 is %T, want *StyleRule", sheet.Rules[0])
	}
	if style.Selectors == nil || len(style.Selectors.Selectors) != 1 {
		t.Fatalf("selectors = %#v", style.Selectors)
	}
	if len(style.Declarations) != 2 {
		t.Fatalf("got %d declarations, want 2", len(style.Declarations))
	}
	if style.Declarations[0].Name != "color" || style.Declarations[0].Important {
		t.Errorf("declaration 0 = %#v", style.Declarations[0])
	}
	if style.Declarations[1].Name != "margin" || !style.Declarations[1].Important {
		t.Errorf("declaration 1 = %#v", style.Declarations[1])
	}
}

func TestSerializeStyleRule(t *testing.T) {
	sheet := Parse("a, b { color: red; margin: 0 !important }")
	got := sheet.Serialize()
	want := "a, b { color: red; margin: 0 !important; }"
	if got != want {
		t.Errorf("Serialize() = %q, want %q", got, want)
	}
}

func TestParseAtRule(t *testing.T) {
	sheet := Parse("@media screen { a { color: red } }")
	if len(sheet.Rules) != 1 {
		t.Fatalf("got %d rules, want 1", len(sheet.Rules))
	}
	atRule, ok := sheet.Rules[0].(*AtRule)
	if !ok {
		t.Fatalf("rule = %T, want *AtRule", sheet.Rules[0])
	}
	if atRule.Name != "media" {
		t.Errorf("name = %q, want media", atRule.Name)
	}
}

func TestParseStatementAtRule(t *testing.T) {
	sheet := Parse("@import url(foo.css);")
	if len(sheet.Rules) != 1 {
		t.Fatalf("got %d rules, want 1", len(sheet.Rules))
	}
	atRule := sheet.Rules[0].(*AtRule)
	if atRule.Block != nil {
		t.Errorf("block = %#v, want nil", atRule.Block)
	}
	if got := atRule.Serialize(); got != "@import url(foo.css);" {
		t.Errorf("Serialize() = %q", got)
	}
}

func TestSerializeRoundTrip(t *testing.T) {
	inputs := []string{
		"a { color: red; }",
		"a, b:hover > c::after { color: #abcdef; width: calc(100% - 2px); }",
		"@media screen { a { color: red } }",
		"@import url(foo.css);",
		"div { margin: 0 auto !important; --brand: green; }",
	}
	for _, input := range inputs {
		sheet := Parse(input)
		first := sheet.Serialize()
		reparsed := Parse(first)
		second := reparsed.Serialize()
		if first != second {
			t.Errorf("serialize not stable for %q:\n first: %q\nsecond: %q", input, first, second)
		}
		if len(reparsed.Rules) != len(sheet.Rules) {
			t.Errorf("rule count changed for %q: %d -> %d", input, len(sheet.Rules), len(reparsed.Rules))
		}
	}
}

func TestParseRule(t *testing.T) {
	rule, err := ParseRule("div { color: blue }")
	if err != nil {
		t.Fatalf("ParseRule: %v", err)
	}
	style, ok := rule.(*StyleRule)
	if !ok {
		t.Fatalf("rule = %T", rule)
	}
	if len(style.Declarations) != 1 {
		t.Errorf("declarations = %#v", style.Declarations)
	}
}
