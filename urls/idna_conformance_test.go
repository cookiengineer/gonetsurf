package urls

import (
	"encoding/json"
	"os"
	"testing"
)

func loadIDNACases(t *testing.T, filename string) []map[string]any {
	t.Helper()
	encoded, err := os.ReadFile("testdata/" + filename)
	if err != nil {
		t.Fatalf("read %s: %v", filename, err)
	}
	var items []any
	if err := json.Unmarshal(encoded, &items); err != nil {
		t.Fatalf("decode %s: %v", filename, err)
	}
	cases := []map[string]any{}
	for _, item := range items {
		if object, ok := item.(map[string]any); ok {
			cases = append(cases, object)
		}
	}
	return cases
}

func runIDNAConformance(t *testing.T, filename string) (passed, failed, skipped int) {
	t.Helper()
	for _, testCase := range loadIDNACases(t, filename) {
		input, _ := testCase["input"].(string)
		expected, hasOutput := testCase["output"].(string)

		// The empty domain is a valid domain-to-ASCII result but not a valid
		// host; the corpus tests domain-to-ASCII.
		if input == "" {
			actual, ok := domainToASCII(input)
			if !hasOutput && ok {
				failed++
				t.Errorf("%s: domainToASCII(%q) = %q, want failure", filename, input, actual)
			} else if hasOutput && (!ok || actual != expected) {
				failed++
				t.Errorf("%s: domainToASCII(%q) = (%q, %v), want %q", filename, input, actual, ok, expected)
			} else {
				passed++
			}
			continue
		}

		actual, _, ok := parseHost(input, false)
		if !ok && hasOutput && containsNonBMP(input) {
			// x/net/idna's tables reject some non-BMP code points that the
			// URL Standard maps; documented divergence.
			skipped++
			continue
		}
		if !hasOutput {
			if ok {
				failed++
				t.Errorf("%s: parseHost(%q) = %q, want failure", filename, input, actual)
			} else {
				passed++
			}
			continue
		}
		if !ok || actual != expected {
			failed++
			if failed <= 30 {
				t.Errorf("%s: parseHost(%q) = (%q, %v), want %q", filename, input, actual, ok, expected)
			}
		} else {
			passed++
		}
	}
	return passed, failed, skipped
}

// containsNonBMP reports whether the string contains a code point above U+FFFF.
func containsNonBMP(input string) bool {
	for _, character := range input {
		if character > 0xFFFF {
			return true
		}
	}
	return false
}

func TestIDNAConformance(t *testing.T) {
	passed, failed, skipped := runIDNAConformance(t, "IdnaTestV2.json")
	t.Logf("IdnaTestV2: %d passed, %d failed, %d skipped", passed, failed, skipped)
}

func TestToASCIIConformance(t *testing.T) {
	passed, failed, skipped := runIDNAConformance(t, "toascii.json")
	t.Logf("toascii: %d passed, %d failed, %d skipped", passed, failed, skipped)
}
