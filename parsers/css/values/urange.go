package values

import (
	"strconv"
	"strings"

	"gonetsurf/parsers/css/parser"
	"gonetsurf/parsers/css/tokenizer"
)

// URange is a contiguous range of Unicode code points. See CSS Syntax 3 §7.
type URange struct {
	Start int
	End   int
}

const maximumCodePoint = 0x10FFFF

// ParseURange parses a single <urange> from a string.
func ParseURange(input string) (*URange, bool) {
	return ParseURangeValues(parser.ParseComponentValueList(input))
}

// ParseURangeValues parses a single <urange> from component values.
func ParseURangeValues(values []parser.ComponentValue) (*URange, bool) {
	tokens, ok := urangeTokens(values)
	if !ok || len(tokens) < 2 {
		return nil, false
	}
	first := tokens[0]
	if first.Type != tokenizer.TokenIdent || !strings.EqualFold(first.Value, "u") {
		return nil, false
	}
	var text strings.Builder
	for _, token := range tokens[1:] {
		text.WriteString(token.Representation)
	}
	return interpretURange(text.String())
}

// ParseURangeList parses a comma-separated list of <urange> values. A nil
// entry marks an invalid range.
func ParseURangeList(input string) []*URange {
	values := parser.ParseComponentValueList(input)
	groups := [][]parser.ComponentValue{}
	current := []parser.ComponentValue{}
	for _, value := range values {
		if token, ok := parser.AsToken(value); ok && token.Type == tokenizer.TokenComma {
			groups = append(groups, current)
			current = nil
			continue
		}
		current = append(current, value)
	}
	groups = append(groups, current)
	result := make([]*URange, 0, len(groups))
	for _, group := range groups {
		rangeValue, ok := ParseURangeValues(group)
		if !ok {
			result = append(result, nil)
			continue
		}
		result = append(result, rangeValue)
	}
	return result
}

// urangeTokens returns the non-EOF tokens, rejecting whitespace (which is not
// permitted inside a <urange>).
func urangeTokens(values []parser.ComponentValue) ([]tokenizer.Token, bool) {
	start := 0
	for start < len(values) && parser.IsWhitespace(values[start]) {
		start++
	}
	end := len(values)
	for end > start && parser.IsWhitespace(values[end-1]) {
		end--
	}
	tokens := make([]tokenizer.Token, 0, end-start)
	for _, value := range values[start:end] {
		if parser.IsWhitespace(value) {
			return nil, false
		}
		token, ok := parser.AsToken(value)
		if !ok {
			return nil, false
		}
		tokens = append(tokens, token)
	}
	return tokens, true
}

func interpretURange(text string) (*URange, bool) {
	codePoints := []rune(text)
	index := 0
	if index >= len(codePoints) || codePoints[index] != '+' {
		return nil, false
	}
	index++
	digitsStart := index
	for index < len(codePoints) && isHexDigit(codePoints[index]) {
		index++
	}
	digitsEnd := index
	questionsStart := index
	for index < len(codePoints) && codePoints[index] == '?' {
		index++
	}
	consumed := index - digitsStart
	if consumed == 0 || consumed > 6 {
		return nil, false
	}
	if index > questionsStart {
		if index != len(codePoints) {
			return nil, false
		}
		digits := string(codePoints[digitsStart:digitsEnd])
		questionCount := index - questionsStart
		start := parseHex(digits + strings.Repeat("0", questionCount))
		end := parseHex(digits + strings.Repeat("F", questionCount))
		return validateURange(start, end)
	}
	start := parseHex(string(codePoints[digitsStart:digitsEnd]))
	if index == len(codePoints) {
		return validateURange(start, start)
	}
	if codePoints[index] != '-' {
		return nil, false
	}
	index++
	endStart := index
	for index < len(codePoints) && isHexDigit(codePoints[index]) {
		index++
	}
	endDigits := index - endStart
	if endDigits == 0 || endDigits > 6 || index != len(codePoints) {
		return nil, false
	}
	end := parseHex(string(codePoints[endStart:index]))
	return validateURange(start, end)
}

func validateURange(start, end int) (*URange, bool) {
	if end > maximumCodePoint || start > end {
		return nil, false
	}
	return &URange{Start: start, End: end}, true
}

func isHexDigit(value rune) bool {
	return (value >= '0' && value <= '9') ||
		(value >= 'a' && value <= 'f') ||
		(value >= 'A' && value <= 'F')
}

func parseHex(text string) int {
	value, err := strconv.ParseInt(text, 16, 32)
	if err != nil {
		return -1
	}
	return int(value)
}
