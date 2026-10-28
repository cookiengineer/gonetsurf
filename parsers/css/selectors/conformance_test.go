package selectors_test

import (
	"strings"
	"testing"

	"gonetsurf/internal/conformance"
	"gonetsurf/parsers/css/selectors"
)

// selectorSkipReason reports why a WPT case is not part of the supported
// Selectors Level 4 subset yet.
func selectorSkipReason(input string) string {
	switch {
	case strings.Contains(input, ":has-slotted"):
		return "tentative :has-slotted()"
	case strings.Contains(input, ":heading"):
		return "tentative :heading()"
	case strings.Contains(input, "-*"), strings.Contains(input, "*-"):
		return "tentative class-prefix syntax"
	default:
		return ""
	}
}

// TestWPTSelectorConformance checks selector validity against the Web Platform
// Tests selector parsing tables (extracted by cmd/corpusgen).
func TestWPTSelectorConformance(t *testing.T) {
	entries, err := conformance.Load("testdata/wpt-selectors.json")
	if err != nil {
		t.Fatal(err)
	}
	failures := 0
	skipped := 0
	for _, entry := range entries {
		input := entry.Input.(string)
		if reason := selectorSkipReason(input); reason != "" {
			skipped++
			t.Logf("skip %q: %s", input, reason)
			continue
		}
		wantValid := entry.Expected.(bool)
		_, parseErr := selectors.Parse(input)
		gotValid := parseErr == nil
		if gotValid != wantValid {
			failures++
			if failures <= 40 {
				t.Errorf("Parse(%q): valid = %v, want %v (err: %v)", input, gotValid, wantValid, parseErr)
			}
		}
	}
	if skipped != 78 {
		t.Errorf("WPT selectors: got %d documented skips, want 78", skipped)
	}
	t.Logf("WPT selectors: %d cases, %d skipped, %d failures", len(entries), skipped, failures)
}
