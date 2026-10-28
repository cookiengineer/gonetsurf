package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func generateCorpora(upstream, root, corpus string) error {
	if corpus == "all" || corpus == "css-parsing-tests" {
		if err := generateCSSParsing(upstream, root); err != nil {
			return err
		}
	}
	if corpus == "all" || corpus == "url" {
		if err := generateURL(upstream, root); err != nil {
			return err
		}
	}
	if corpus == "all" || corpus == "html5lib" {
		if err := generateHTML5lib(upstream, root); err != nil {
			return err
		}
	}
	if corpus == "all" || corpus == "wpt" {
		if err := generateWPTSelectors(upstream, root); err != nil {
			return err
		}
		if err := generateWPTValues(upstream, root); err != nil {
			return err
		}
	}
	return nil
}

// cssParsingCopies maps source files to their package testdata destinations.
// These corpora need no structural changes.
var cssParsingCopies = map[string]string{
	"An+B.json":                "parsers/css/selectors/testdata/anb.json",
	"color_function_4.json":    "parsers/css/colors/testdata/color_function_4.json",
	"color_functions_5.json":   "parsers/css/colors/testdata/color_functions_5.json",
	"color_hexadecimal_3.json": "parsers/css/colors/testdata/color_hexadecimal_3.json",
	"color_hexadecimal_4.json": "parsers/css/colors/testdata/color_hexadecimal_4.json",
	"color_hsl_3.json":         "parsers/css/colors/testdata/color_hsl_3.json",
	"color_hsl_4.json":         "parsers/css/colors/testdata/color_hsl_4.json",
	"color_hwb_4.json":         "parsers/css/colors/testdata/color_hwb_4.json",
	"color_keywords_3.json":    "parsers/css/colors/testdata/color_keywords_3.json",
	"color_keywords_4.json":    "parsers/css/colors/testdata/color_keywords_4.json",
	"color_lab_4.json":         "parsers/css/colors/testdata/color_lab_4.json",
	"color_lch_4.json":         "parsers/css/colors/testdata/color_lch_4.json",
	"color_oklab_4.json":       "parsers/css/colors/testdata/color_oklab_4.json",
	"color_oklch_4.json":       "parsers/css/colors/testdata/color_oklch_4.json",
}

// cssParsingNormalized files are copied but have their expected values
// rewritten to match the current CSS Syntax 3 token set.
var cssParsingNormalized = []string{
	"component_value_list.json",
	"one_component_value.json",
	"declaration_list.json",
	"one_declaration.json",
	"one_rule.json",
	"rule_list.json",
	"stylesheet.json",
	"blocks_contents.json",
}

func generateCSSParsing(upstream, root string) error {
	source := filepath.Join(upstream, "css-parsing-tests")
	for name, destination := range cssParsingCopies {
		items, err := readJSON(filepath.Join(source, name))
		if err != nil {
			return err
		}
		if err := writeJSON(filepath.Join(root, destination), items); err != nil {
			return err
		}
	}
	for _, name := range cssParsingNormalized {
		items, err := readJSON(filepath.Join(source, name))
		if err != nil {
			return err
		}
		normalized, err := normalizeCorpus(items)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		destination := filepath.Join(root, "parsers/css/parser/testdata", name)
		if err := writeJSON(destination, normalized); err != nil {
			return err
		}
	}
	return nil
}

func readJSON(path string) (any, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var value any
	if err := json.Unmarshal(encoded, &value); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return value, nil
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(value, "", " ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	return os.WriteFile(path, encoded, 0o644)
}

var matchOperators = map[string][]string{
	"~=": {"~", "="},
	"|=": {"|", "="},
	"^=": {"^", "="},
	"$=": {"$", "="},
	"*=": {"*", "="},
	"||": {"|", "|"},
}

// normalizeCorpus rewrites each [input, expected] pair so the expected value
// reflects the current CSS Syntax 3 tokenization: match operators become
// delimiter sequences and the tinycss2 EOF error markers are dropped. Pairs
// that rely on the unicode-range token are removed, because normal
// tokenization does not produce it.
func normalizeCorpus(value any) (any, error) {
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("corpus is not a JSON array")
	}
	if len(items)%2 != 0 {
		return nil, fmt.Errorf("corpus has an odd number of items (%d)", len(items))
	}
	result := make([]any, 0, len(items))
	for index := 0; index < len(items); index += 2 {
		expected, keep := normalizeExpected(items[index+1])
		if !keep {
			continue
		}
		result = append(result, items[index], normalizeDeclarationWhitespace(expected))
	}
	return result, nil
}

func normalizeExpected(value any) (any, bool) {
	if isUnicodeRangeComponent(value) {
		return nil, false
	}
	items, ok := value.([]any)
	if !ok {
		return value, true
	}
	result := make([]any, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok {
			if replacement, ok := matchOperators[text]; ok {
				for _, part := range replacement {
					result = append(result, part)
				}
				continue
			}
		}
		if isEOFError(item) {
			continue
		}
		if isUnicodeRangeComponent(item) {
			return nil, false
		}
		normalized, keep := normalizeExpected(item)
		if !keep {
			return nil, false
		}
		result = append(result, normalized)
	}
	return result, true
}

// normalizeDeclarationWhitespace trims the leading and trailing whitespace
// markers the 2021 corpus preserves around declaration values. Current CSS
// Syntax 3 discards that whitespace when parsing a declaration.
func normalizeDeclarationWhitespace(value any) any {
	items, ok := value.([]any)
	if !ok {
		return value
	}
	if len(items) == 4 && items[0] == "declaration" {
		if values, ok := items[2].([]any); ok {
			items[2] = trimWhitespaceMarkers(values)
		}
	}
	for index, item := range items {
		items[index] = normalizeDeclarationWhitespace(item)
	}
	return items
}

func trimWhitespaceMarkers(values []any) []any {
	start := 0
	for start < len(values) && values[start] == " " {
		start++
	}
	end := len(values)
	for end > start && values[end-1] == " " {
		end--
	}
	return values[start:end]
}

func isEOFError(value any) bool {
	items, ok := value.([]any)
	if !ok || len(items) != 2 {
		return false
	}
	tag, ok := items[0].(string)
	if !ok || tag != "error" {
		return false
	}
	reason, ok := items[1].(string)
	return ok && (reason == "eof-in-string" || reason == "eof-in-url")
}

func isUnicodeRangeComponent(value any) bool {
	items, ok := value.([]any)
	if !ok || len(items) != 3 {
		return false
	}
	tag, ok := items[0].(string)
	return ok && tag == "unicode-range"
}
