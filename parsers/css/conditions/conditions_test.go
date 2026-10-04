package conditions

import "testing"

type checker struct {
	declarations map[string]bool
	selectors    map[string]bool
}

func (c checker) SupportsDeclaration(property, value string) bool {
	return c.declarations[property]
}

func (c checker) SupportsSelector(selector string) bool {
	return c.selectors[selector]
}

func TestSupportsCondition(t *testing.T) {
	check := checker{
		declarations: map[string]bool{"display": true, "color": false},
		selectors:    map[string]bool{":has(a)": true},
	}
	tests := []struct {
		input string
		want  bool
	}{
		{"(display: grid)", true},
		{"(color: red)", false},
		{"(display: grid) and (color: red)", false},
		{"(display: grid) or (color: red)", true},
		{"not (color: red)", true},
		{"not (display: grid)", false},
		{"selector(:has(a))", true},
		{"selector(:nth-child(2n))", false},
	}
	for _, test := range tests {
		condition, err := ParseSupports(test.input)
		if err != nil {
			t.Errorf("ParseSupports(%q): %v", test.input, err)
			continue
		}
		if got := condition.Evaluate(check); got != test.want {
			t.Errorf("Evaluate(%q) = %v, want %v", test.input, got, test.want)
		}
	}
}

func TestParseImport(t *testing.T) {
	parsed, err := ParseImport("url('theme.css') layer(base) supports(display: grid) screen and (min-width: 600px)")
	if err != nil {
		t.Fatalf("ParseImport: %v", err)
	}
	if parsed.URL != "theme.css" {
		t.Errorf("URL = %q", parsed.URL)
	}
	if parsed.Layer != "base" {
		t.Errorf("Layer = %q", parsed.Layer)
	}
	if parsed.Supports == nil {
		t.Error("Supports is nil")
	}
	if parsed.MediaQueries != "screen and (min-width: 600px)" {
		t.Errorf("MediaQueries = %q", parsed.MediaQueries)
	}
}
