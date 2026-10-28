package main

import (
	"io/fs"
	"path/filepath"
	"strings"
)

type valueCall struct {
	Property   string
	Value      string
	Serialized []string
	Valid      bool
}

// wptValueDirectories is the stable parsing set used by the Phase 4
// conformance suite. Other parsing directories remain vendored for later
// phases but are not part of the supported property set yet.
var wptValueDirectories = []string{
	"css-animations",
	"css-backgrounds",
	"css-box",
	"css-color",
	"css-display",
	"css-fonts",
	"css-overflow",
	"css-position",
	"css-sizing",
	"css-text",
	"css-text-decor",
	"css-transitions",
	"selectors",
}

func generateWPTValues(upstream, root string) error {
	corpus := []any{}
	seen := map[string]bool{}
	for _, directory := range wptValueDirectories {
		base := filepath.Join(upstream, "wpt", "css", directory)
		err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			if !strings.Contains(filepath.ToSlash(path), "/parsing/") {
				return nil
			}
			name := entry.Name()
			if !strings.HasSuffix(name, "-valid.html") && !strings.HasSuffix(name, "-invalid.html") {
				return nil
			}
			source, err := readFileString(path)
			if err != nil {
				return err
			}
			for _, call := range extractValueCalls(source) {
				key := call.Property + "\x00" + call.Value + "\x00" + boolString(call.Valid) + "\x00" + strings.Join(call.Serialized, "\x01")
				if seen[key] {
					continue
				}
				seen[key] = true
				item := map[string]any{
					"property": call.Property,
					"value":    call.Value,
					"valid":    call.Valid,
				}
				if len(call.Serialized) > 0 {
					item["serialized"] = call.Serialized
				}
				corpus = append(corpus, item)
			}
			return nil
		})
		if err != nil {
			return err
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
	serialized, after := readSerializedArgument(source, after)
	return valueCall{Property: property, Value: value, Serialized: serialized, Valid: valid}, after, true
}

// readSerializedArgument reads the optional third test_valid_value argument.
// It is either one JavaScript string or an array of acceptable strings. When
// no third argument is present the position is returned unchanged.
func readSerializedArgument(source string, position int) ([]string, int) {
	cursor := position
	for cursor < len(source) && isJSSpace(source[cursor]) {
		cursor++
	}
	if cursor >= len(source) || source[cursor] != ',' {
		return nil, position
	}
	cursor++
	for cursor < len(source) && isJSSpace(source[cursor]) {
		cursor++
	}
	if cursor >= len(source) {
		return nil, position
	}
	if source[cursor] == '\'' || source[cursor] == '"' {
		value, next, ok := readJSString(source, cursor)
		if !ok {
			return nil, position
		}
		return []string{value}, next
	}
	if source[cursor] != '[' {
		return nil, position
	}
	cursor++
	values := []string{}
	for {
		for cursor < len(source) && (isJSSpace(source[cursor]) || source[cursor] == ',') {
			cursor++
		}
		if cursor >= len(source) {
			return nil, position
		}
		if source[cursor] == ']' {
			return values, cursor + 1
		}
		if source[cursor] != '\'' && source[cursor] != '"' {
			return nil, position
		}
		value, next, ok := readJSString(source, cursor)
		if !ok {
			return nil, position
		}
		values = append(values, value)
		cursor = next
	}
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
