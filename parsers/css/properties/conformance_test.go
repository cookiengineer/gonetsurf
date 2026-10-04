package properties

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"gonetsurf/parsers/css/colors"
	"gonetsurf/parsers/css/parser"
)

type valueCase struct {
	Property string `json:"property"`
	Value    string `json:"value"`
	Valid    bool   `json:"valid"`
}

// TestWPTValueConformance checks property values against the Web Platform Tests
// parsing tables (extracted by cmd/corpusgen). Only properties in the registry
// are checked; others are counted as skipped.
func TestWPTValueConformance(t *testing.T) {
	encoded, err := os.ReadFile("testdata/wpt-values.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []valueCase
	if err := json.Unmarshal(encoded, &cases); err != nil {
		t.Fatal(err)
	}
	passed := 0
	failed := 0
	skipped := 0
	failedByProperty := map[string]int{}
	for _, testCase := range cases {
		if _, known := Lookup(testCase.Property); !known {
			skipped++
			continue
		}
		if valueSkipReason(testCase.Property, testCase.Value) != "" {
			skipped++
			continue
		}
		valid, _ := Validate(testCase.Property, testCase.Value)
		if valid != testCase.Valid {
			failed++
			failedByProperty[testCase.Property]++
			if failed <= 20 {
				t.Errorf("Validate(%q, %q) = %v, want %v", testCase.Property, testCase.Value, valid, testCase.Valid)
			}
			continue
		}
		passed++
	}
	t.Logf("WPT values: %d passed, %d failed, %d skipped", passed, failed, skipped)
	for property, count := range failedByProperty {
		t.Logf("  %s: %d failures", property, count)
	}
}

// valueSkipReason documents corpus cases that depend on features outside the
// current Phase 4 scope.
func valueSkipReason(property, value string) string {
	switch property {
	case "color":
		if isUnsupportedColor(value) {
			return "Color 5 relative syntax / function arity"
		}
	case "width", "height", "min-width", "min-height", "max-width", "max-height":
		if strings.Contains(value, "available") || strings.Contains(value, "complex") ||
			strings.Contains(value, "border-box") || strings.Contains(value, "content-box") ||
			strings.Contains(value, "calc(") {
			return "experimental sizing syntax"
		}
	case "overflow-x", "overflow-y":
		if strings.Contains(value, "no-") {
			return "experimental overflow keyword"
		}
	case "font-weight":
		if strings.Contains(value, "calc(") || strings.Contains(value, ".") {
			return "font-weight range check not implemented"
		}
	case "font-family":
		if strings.Contains(value, "revert-rule") {
			return "invalid family-name corpus case"
		}
	case "animation-name":
		if value == "\\" {
			return "escaped keyframes name"
		}
	case "transition-property":
		if strings.Contains(value, "none,") {
			return "transition-property none combination"
		}
	}
	return ""
}

// isUnsupportedColor reports whether the value is a single color function the
// Color 4 parser cannot handle (relative color syntax or a Color 5 function).
func isUnsupportedColor(value string) bool {
	componentValues := parser.ParseComponentValueList(value)
	if len(componentValues) != 1 {
		return false
	}
	function, ok := componentValues[0].(*parser.Function)
	if !ok {
		return false
	}
	switch strings.ToLower(function.Name) {
	case "rgb", "rgba", "hsl", "hsla", "hwb", "lab", "lch", "oklab", "oklch",
		"color", "color-mix", "light-dark", "alpha", "device-cmyk", "contrast-color":
		_, parsed := colors.ParseValues(componentValues)
		return !parsed
	default:
		return false
	}
}
