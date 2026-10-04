package encodings

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestEncodingConformance runs the html5lib-tests encoding sniffing corpus.
// Each case is a byte stream (U+0000..U+00FF map to bytes) and the expected
// canonical encoding name.
func TestEncodingConformance(t *testing.T) {
	files := []string{"tests1.dat", "tests2.dat"}
	for _, filename := range files {
		runEncodingConformance(t, filename)
	}
}

func runEncodingConformance(t *testing.T, filename string) {
	t.Helper()
	encoded, err := os.ReadFile("../testdata/html5lib-tests/encoding/" + filename)
	if err != nil {
		t.Fatalf("read %s: %v", filename, err)
	}
	passed := 0
	failed := 0
	for _, testCase := range parseEncodingCorpus(encoded) {
		actual := Sniff("", testCase.data).Name
		if strings.EqualFold(actual, testCase.expected) {
			passed++
			continue
		}
		failed++
		if failed <= 20 {
			t.Errorf("%s: sniff(%q) = %q, want %q", filename, string(testCase.data), actual, testCase.expected)
		}
	}
	t.Logf("%s: %d passed, %d failed", filename, passed, failed)
}

type encodingCase struct {
	data     []byte
	expected string
}

func parseEncodingCorpus(source []byte) []encodingCase {
	const (
		outside = iota
		readingData
		readingEncoding
	)
	lines := bytes.Split(source, []byte("\n"))
	cases := []encodingCase{}
	var data []byte
	expected := ""
	state := outside
	for _, line := range lines {
		switch string(line) {
		case "#data":
			if state == readingEncoding && expected != "" {
				cases = append(cases, encodingCase{data: data, expected: expected})
			}
			data = nil
			expected = ""
			state = readingData
		case "#encoding":
			state = readingEncoding
		default:
			switch state {
			case readingData:
				data = append(data, line...)
				data = append(data, '\n')
			case readingEncoding:
				if expected == "" && len(strings.TrimSpace(string(line))) > 0 {
					expected = strings.TrimSpace(string(line))
				}
			}
		}
	}
	if state == readingEncoding && expected != "" {
		cases = append(cases, encodingCase{data: data, expected: expected})
	}
	return cases
}
