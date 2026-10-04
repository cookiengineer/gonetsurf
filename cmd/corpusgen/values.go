package main

import (
	"path/filepath"
	"strings"
)

type valueCall struct {
	Property string
	Value    string
	Valid    bool
}

func generateWPTValues(upstream, root string) error {
	patterns := []string{
		filepath.Join(upstream, "wpt", "css", "*", "parsing", "*-valid.html"),
		filepath.Join(upstream, "wpt", "css", "*", "parsing", "*-invalid.html"),
	}
	corpus := []any{}
	seen := map[string]bool{}
	for _, pattern := range patterns {
		files, err := filepath.Glob(pattern)
		if err != nil {
			return err
		}
		for _, file := range files {
			source, err := readFileString(file)
			if err != nil {
				return err
			}
			for _, call := range extractValueCalls(source) {
				key := call.Property + "\x00" + call.Value + "\x00" + boolString(call.Valid)
				if seen[key] {
					continue
				}
				seen[key] = true
				corpus = append(corpus, map[string]any{
					"property": call.Property,
					"value":    call.Value,
					"valid":    call.Valid,
				})
			}
		}
	}
	return writeJSON(filepath.Join(root, "parsers/css/properties/testdata/wpt-values.json"), corpus)
}

func boolString(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func extractValueCalls(source string) []valueCall {
	calls := []valueCall{}
	for index := 0; index < len(source); {
		if strings.HasPrefix(source[index:], "test_valid_value(") {
			if call, next, ok := readValueCall(source, index+len("test_valid_value("), true); ok {
				calls = append(calls, call)
				index = next
				continue
			}
		}
		if strings.HasPrefix(source[index:], "test_invalid_value(") {
			if call, next, ok := readValueCall(source, index+len("test_invalid_value("), false); ok {
				calls = append(calls, call)
				index = next
				continue
			}
		}
		index++
	}
	return calls
}

func readValueCall(source string, position int, valid bool) (valueCall, int, bool) {
	property, next, ok := readStringArgument(source, position)
	if !ok {
		return valueCall{}, position, false
	}
	value, after, ok := readStringArgument(source, next)
	if !ok {
		return valueCall{}, position, false
	}
	return valueCall{Property: property, Value: value, Valid: valid}, after, true
}

// readStringArgument skips whitespace and commas then reads one JS string.
func readStringArgument(source string, position int) (string, int, bool) {
	for position < len(source) && (isJSSpace(source[position]) || source[position] == ',') {
		position++
	}
	if position >= len(source) || (source[position] != '\'' && source[position] != '"') {
		return "", position, false
	}
	return readJSString(source, position)
}
