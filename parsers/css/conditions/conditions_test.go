package conditions

import (
	"encoding/json"
	"os"
	"testing"
)

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

type conformanceChecker struct {
	declarations map[string]bool
	selectors    map[string]bool
}

func (check conformanceChecker) SupportsDeclaration(property, value string) bool {
	return check.declarations[property+"\x00"+value]
}

func (check conformanceChecker) SupportsSelector(selector string) bool {
	return check.selectors[selector]
}

type supportsCase struct {
	Input string `json:"input"`
	Parse bool   `json:"parse"`
	Match bool   `json:"match"`
}

func TestSupportsConformance(t *testing.T) {
	encoded, err := os.ReadFile("testdata/conditions.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []supportsCase
	if err := json.Unmarshal(encoded, &cases); err != nil {
		t.Fatal(err)
	}
	check := conformanceChecker{
		declarations: map[string]bool{
			"display\x00grid":   true,
			"color\x00red":      true,
			"margin\x000":       true,
			"--custom\x00value": true,
		},
		selectors: map[string]bool{
			"a:link.class#ident": true,
			":has(a)":            true,
			":nth-child(2n)":     false,
		},
	}
	for _, testCase := range cases {
		condition, err := ParseSupports(testCase.Input)
		if (err == nil) != testCase.Parse {
			t.Errorf("ParseSupports(%q) parse = %v, want %v (err: %v)", testCase.Input, err == nil, testCase.Parse, err)
			continue
		}
		if err != nil {
			continue
		}
		if match := condition.Evaluate(check); match != testCase.Match {
			t.Errorf("Evaluate(%q) = %v, want %v", testCase.Input, match, testCase.Match)
		}
	}
}

type importCase struct {
	Input string `json:"input"`
	URL   string `json:"url"`
	Layer string `json:"layer"`
	Media string `json:"media"`
}

func TestImportConformance(t *testing.T) {
	encoded, err := os.ReadFile("testdata/imports.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []importCase
	if err := json.Unmarshal(encoded, &cases); err != nil {
		t.Fatal(err)
	}
	for _, testCase := range cases {
		parsed, err := ParseImport(testCase.Input)
		if err != nil {
			t.Errorf("ParseImport(%q): %v", testCase.Input, err)
			continue
		}
		if parsed.URL != testCase.URL || parsed.Layer != testCase.Layer || parsed.MediaQueries != testCase.Media {
			t.Errorf("ParseImport(%q) = {url:%q layer:%q media:%q}, want {url:%q layer:%q media:%q}", testCase.Input, parsed.URL, parsed.Layer, parsed.MediaQueries, testCase.URL, testCase.Layer, testCase.Media)
		}
	}
}
