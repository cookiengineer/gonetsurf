package microsyntaxes

import "strings"

// SplitSpaceSeparated implements the rules for splitting a string on ASCII
// whitespace. Empty fields are dropped.
func SplitSpaceSeparated(input string) []string {
	return strings.FieldsFunc(input, func(character rune) bool {
		return character == '\t' || character == '\n' || character == '\f' ||
			character == '\r' || character == ' '
	})
}

// SplitCommaSeparated implements the rules for splitting a string on commas,
// trimming surrounding ASCII whitespace from each token. Empty tokens are
// preserved.
func SplitCommaSeparated(input string) []string {
	parts := strings.Split(input, ",")
	for index, part := range parts {
		parts[index] = strings.Trim(part, " \t\n\f\r")
	}
	return parts
}

// UniqueSpaceSeparated returns the space-separated tokens with duplicates
// removed, preserving the first occurrence order.
func UniqueSpaceSeparated(input string) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, token := range SplitSpaceSeparated(input) {
		if _, exists := seen[token]; exists {
			continue
		}
		seen[token] = struct{}{}
		result = append(result, token)
	}
	return result
}

// UniqueCommaSeparated returns the comma-separated tokens with duplicates
// removed, preserving the first occurrence order.
func UniqueCommaSeparated(input string) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, token := range SplitCommaSeparated(input) {
		if _, exists := seen[token]; exists {
			continue
		}
		seen[token] = struct{}{}
		result = append(result, token)
	}
	return result
}
