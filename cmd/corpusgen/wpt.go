package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// selectorCall is one extracted test_valid_selector/test_invalid_selector case.
type selectorCall struct {
	Selector string
	Valid    bool
}

func generateWPTSelectors(upstream, root string) error {
	base := filepath.Join(upstream, "wpt", "css", "selectors", "parsing")
	corpus := []any{}
	err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
			return nil
		}
		encoded, err := readFileString(path)
		if err != nil {
			return err
		}
		for _, call := range extractSelectorCalls(encoded) {
			corpus = append(corpus, call.Selector, call.Valid)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return writeJSON(filepath.Join(root, "parsers/css/selectors/testdata/wpt-selectors.json"), corpus)
}

func extractSelectorCalls(source string) []selectorCall {
	calls := []selectorCall{}
	for index := 0; index < len(source); {
		if strings.HasPrefix(source[index:], "test_valid_selector(") {
			if call, next, ok := readSelectorCall(source, index+len("test_valid_selector("), true); ok {
				calls = append(calls, call)
				index = next
				continue
			}
		}
		if strings.HasPrefix(source[index:], "test_invalid_selector(") {
			if call, next, ok := readSelectorCall(source, index+len("test_invalid_selector("), false); ok {
				calls = append(calls, call)
				index = next
				continue
			}
		}
		index++
	}
	return calls
}

func readSelectorCall(source string, position int, valid bool) (selectorCall, int, bool) {
	for position < len(source) && isJSSpace(source[position]) {
		position++
	}
	if position >= len(source) || (source[position] != '\'' && source[position] != '"') {
		return selectorCall{}, position, false
	}
	value, end, ok := readJSString(source, position)
	if !ok {
		return selectorCall{}, position, false
	}
	return selectorCall{Selector: value, Valid: valid}, end, true
}

// readJSString reads a single- or double-quoted JavaScript string literal,
// decoding common escapes. It returns the decoded value and the index just
// past the closing quote.
func readJSString(source string, position int) (string, int, bool) {
	quote := source[position]
	var builder strings.Builder
	index := position + 1
	for index < len(source) {
		current := source[index]
		if current == quote {
			return builder.String(), index + 1, true
		}
		if current != '\\' {
			builder.WriteByte(current)
			index++
			continue
		}
		if index+1 >= len(source) {
			return "", index, false
		}
		escaped := source[index+1]
		switch escaped {
		case 'n':
			builder.WriteByte('\n')
			index += 2
		case 't':
			builder.WriteByte('\t')
			index += 2
		case 'r':
			builder.WriteByte('\r')
			index += 2
		case 'b':
			builder.WriteByte('\b')
			index += 2
		case 'f':
			builder.WriteByte('\f')
			index += 2
		case 'v':
			builder.WriteByte('\v')
			index += 2
		case '0':
			builder.WriteByte('\x00')
			index += 2
		case 'u':
			if index+5 < len(source) {
				if value, err := strconv.ParseUint(source[index+2:index+6], 16, 32); err == nil {
					builder.WriteRune(rune(value))
					index += 6
					continue
				}
			}
			builder.WriteByte(escaped)
			index += 2
		case 'x':
			if index+3 < len(source) {
				if value, err := strconv.ParseUint(source[index+2:index+4], 16, 32); err == nil {
					builder.WriteRune(rune(value))
					index += 4
					continue
				}
			}
			builder.WriteByte(escaped)
			index += 2
		default:
			builder.WriteByte(escaped)
			index += 2
		}
	}
	return "", index, false
}

func isJSSpace(current byte) bool {
	return current == ' ' || current == '\t' || current == '\n' || current == '\r'
}

func readFileString(path string) (string, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}
