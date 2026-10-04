package urls

import (
	"encoding/json"
	"os"
	"testing"
)

type urlTestCase struct {
	Input    string  `json:"input"`
	Base     *string `json:"base"`
	Failure  bool    `json:"failure"`
	Href     string  `json:"href"`
	Origin   string  `json:"origin"`
	Protocol string  `json:"protocol"`
	Username string  `json:"username"`
	Password string  `json:"password"`
	Host     string  `json:"host"`
	Hostname string  `json:"hostname"`
	Port     string  `json:"port"`
	Pathname string  `json:"pathname"`
	Search   string  `json:"search"`
	Hash     string  `json:"hash"`
}

func TestURLParserConformance(t *testing.T) {
	data, err := os.ReadFile("testdata/urltestdata.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("unmarshal testdata: %v", err)
	}

	passed := 0
	failed := 0
	for _, entry := range entries {
		if len(entry) > 0 && entry[0] == '"' {
			continue
		}
		var testCase urlTestCase
		if err := json.Unmarshal(entry, &testCase); err != nil {
			t.Fatalf("unmarshal case: %v", err)
		}

		var base *URL
		if testCase.Base != nil {
			parsedBase, baseErr := Parse(*testCase.Base)
			if baseErr != nil {
				t.Errorf("base %q failed: %v", *testCase.Base, baseErr)
				continue
			}
			base = parsedBase
		}

		var parsed *URL
		var parseErr error
		if base != nil {
			parsed, parseErr = ParseReference(base, testCase.Input)
		} else {
			parsed, parseErr = Parse(testCase.Input)
		}

		if testCase.Failure {
			if parseErr == nil {
				failed++
				t.Errorf("input %q base %v: expected failure, got %q", testCase.Input, baseString(testCase.Base), parsed.String())
			} else {
				passed++
			}
			continue
		}
		if parseErr != nil {
			failed++
			t.Errorf("input %q base %v: unexpected failure: %v", testCase.Input, baseString(testCase.Base), parseErr)
			continue
		}

		if diff := compareCase(parsed, &testCase); diff != "" {
			failed++
			t.Errorf("input %q base %v:\n%s", testCase.Input, baseString(testCase.Base), diff)
			continue
		}
		passed++
	}

	t.Logf("url conformance: %d passed, %d failed", passed, failed)
}

func baseString(base *string) string {
	if base == nil {
		return "<nil>"
	}
	return *base
}

func compareCase(parsed *URL, testCase *urlTestCase) string {
	checks := []struct {
		name     string
		got      string
		expected string
	}{
		{"href", parsed.String(), testCase.Href},
		{"protocol", parsed.Protocol(), testCase.Protocol},
		{"username", parsed.Username, testCase.Username},
		{"password", parsed.Password, testCase.Password},
		{"host", parsed.HostWithPort(), testCase.Host},
		{"hostname", parsed.Hostname(), testCase.Hostname},
		{"port", parsed.Port, testCase.Port},
		{"pathname", parsed.Pathname(), testCase.Pathname},
		{"search", parsed.Search(), testCase.Search},
		{"hash", parsed.Hash(), testCase.Hash},
	}
	output := ""
	for _, check := range checks {
		if check.got != check.expected {
			output += "  " + check.name + ": got " + quote(check.got) + " want " + quote(check.expected) + "\n"
		}
	}
	if testCase.Origin != "" && parsed.OriginString() != testCase.Origin {
		output += "  origin: got " + quote(parsed.OriginString()) + " want " + quote(testCase.Origin) + "\n"
	}
	return output
}

func quote(value string) string {
	return "\"" + value + "\""
}
