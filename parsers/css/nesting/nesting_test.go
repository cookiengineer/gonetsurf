package nesting

import "testing"

func TestParseNestedRules(t *testing.T) {
	rules := Parse("a { color: red; & .b { color: blue } }")
	if len(rules) != 1 {
		t.Fatalf("got %d top-level rules, want 1", len(rules))
	}
	if len(rules[0].Declarations) != 1 || rules[0].Declarations[0].Name != "color" {
		t.Errorf("declarations = %#v", rules[0].Declarations)
	}
	if len(rules[0].Nested) != 1 {
		t.Fatalf("nested = %#v", rules[0].Nested)
	}
	if len(rules[0].Nested[0].Declarations) != 1 || rules[0].Nested[0].Declarations[0].Name != "color" {
		t.Errorf("nested declarations = %#v", rules[0].Nested[0].Declarations)
	}
}

func TestResolveSelector(t *testing.T) {
	tests := []struct {
		nested string
		parent string
		want   string
	}{
		{"& .b", "a", "a .b"},
		{"&:hover", "a", "a:hover"},
		{".b", "a", "a .b"},
		{"& + &", "a", "a + a"},
		{"& .b, & .c", "a", "a .b, a .c"},
		{"& .b", "a, b", "a .b, b .b"},
		{"&.b", "a", "a.b"},
		{".b &", "a", ".b a"},
		{"& > & .bar", "a", "a > a .bar"},
		{"&:is(.a, .b)", "a", "a:is(.a, .b)"},
		{"&:hover, &:focus", "a", "a:hover, a:focus"},
		{"& .b & .c", "a", "a .b a .c"},
	}
	for _, test := range tests {
		if got := ResolveSelector(test.nested, test.parent); got != test.want {
			t.Errorf("ResolveSelector(%q, %q) = %q, want %q", test.nested, test.parent, got, test.want)
		}
	}
}

func TestFlatten(t *testing.T) {
	rules := Flatten(Parse("a { color: red; & .b { color: blue } }"))
	if len(rules) != 2 {
		t.Fatalf("got %d flattened rules, want 2", len(rules))
	}
	if rules[0].Selector != "a" {
		t.Errorf("rule 0 selector = %q, want a", rules[0].Selector)
	}
	if rules[1].Selector != "a .b" {
		t.Errorf("rule 1 selector = %q, want a .b", rules[1].Selector)
	}
}
