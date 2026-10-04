package colors

import (
	"encoding/json"
	"math"
	"os"
	"regexp"
	"strconv"
	"testing"
)

var numberPattern = regexp.MustCompile(`[0-9]+(?:\.[0-9]+)?`)

// normalizeNumbers rounds every decimal number in a serialization to four
// decimal places so that last-digit floating point differences do not cause
// spurious conformance failures.
func normalizeNumbers(value string) string {
	return numberPattern.ReplaceAllStringFunc(value, func(match string) string {
		if !containsDot(match) {
			return match
		}
		number, err := strconv.ParseFloat(match, 64)
		if err != nil {
			return match
		}
		rounded := math.Round(number*1e4) / 1e4
		return strconv.FormatFloat(rounded, 'f', -1, 64)
	})
}

func containsDot(value string) bool {
	for index := 0; index < len(value); index++ {
		if value[index] == '.' {
			return true
		}
	}
	return false
}

func runCorpus(t *testing.T, filename string, tolerant bool) (passed, failed int) {
	t.Helper()
	encoded, err := os.ReadFile("testdata/" + filename)
	if err != nil {
		t.Fatalf("read %s: %v", filename, err)
	}
	var items []any
	if err := json.Unmarshal(encoded, &items); err != nil {
		t.Fatalf("decode %s: %v", filename, err)
	}
	for index := 0; index < len(items); index += 2 {
		input, ok := items[index].(string)
		if !ok {
			continue
		}
		expected := items[index+1]
		color, valid := Parse(input)
		if expected == nil {
			if valid {
				t.Errorf("%s: Parse(%q) = %q, want invalid", filename, input, color.Serialize())
				failed++
			} else {
				passed++
			}
			continue
		}
		want := expected.(string)
		if !valid {
			t.Errorf("%s: Parse(%q) failed, want %q", filename, input, want)
			failed++
			continue
		}
		got := color.Serialize()
		matches := got == want
		if !matches && tolerant {
			matches = normalizeNumbers(got) == normalizeNumbers(want)
		}
		if !matches {
			t.Errorf("%s: Serialize(Parse(%q)) = %q, want %q", filename, input, got, want)
			failed++
			continue
		}
		passed++
	}
	return passed, failed
}

func TestHexadecimalConformance(t *testing.T) {
	passed, failed := runCorpus(t, "color_hexadecimal_3.json", false)
	passed4, failed4 := runCorpus(t, "color_hexadecimal_4.json", false)
	t.Logf("hex3: %d passed, %d failed; hex4: %d passed, %d failed", passed, failed, passed4, failed4)
}

func TestKeywordConformance(t *testing.T) {
	passed, failed := runCorpus(t, "color_keywords_3.json", false)
	passed4, failed4 := runCorpus(t, "color_keywords_4.json", false)
	t.Logf("keywords3: %d passed, %d failed; keywords4: %d passed, %d failed", passed, failed, passed4, failed4)
}

func TestHSLConformance(t *testing.T) {
	passed, failed := runCorpus(t, "color_hsl_3.json", true)
	t.Logf("hsl3: %d passed, %d failed", passed, failed)
}

func TestHWBConformance(t *testing.T) {
	passed, failed := runCorpus(t, "color_hwb_4.json", true)
	t.Logf("hwb4: %d passed, %d failed", passed, failed)
}

func TestHSL4Conformance(t *testing.T) {
	passed, failed := runCorpus(t, "color_hsl_4.json", true)
	t.Logf("hsl4: %d passed, %d failed", passed, failed)
}

func TestLabFamilyConformance(t *testing.T) {
	for _, filename := range []string{"color_lab_4.json", "color_lch_4.json", "color_oklab_4.json", "color_oklch_4.json"} {
		passed, failed := runCorpus(t, filename, true)
		t.Logf("%s: %d passed, %d failed", filename, passed, failed)
	}
}

func TestColorFunctionConformance(t *testing.T) {
	passed, failed := runCorpus(t, "color_function_4.json", true)
	t.Logf("color_function_4.json: %d passed, %d failed", passed, failed)
}
