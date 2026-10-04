package selectors

import (
	"strings"

	"gonetsurf/parsers/css/parser"
	"gonetsurf/parsers/css/tokenizer"
)

// AnB holds the coefficients of an An+B expression: A is the step and B the
// offset. It is used by :nth-child() and friends.
type AnB struct {
	A int
	B int
}

type anbToken struct {
	token  tokenizer.Token
	spaced bool
}

// ParseAnB parses an An+B expression from a string. See CSS Syntax 3 §6.
func ParseAnB(input string) (AnB, bool) {
	return ParseAnBValues(parser.ParseComponentValueList(input))
}

// ParseAnBValues parses an An+B expression from component values.
func ParseAnBValues(values []parser.ComponentValue) (AnB, bool) {
	tokens := normalizeAnB(values)
	if len(tokens) == 0 {
		return AnB{}, false
	}
	parserState := &anbParser{tokens: tokens}
	return parserState.parse()
}

func normalizeAnB(values []parser.ComponentValue) []anbToken {
	tokens := make([]anbToken, 0, len(values))
	spaced := true
	for _, value := range values {
		if parser.IsWhitespace(value) {
			spaced = true
			continue
		}
		token, ok := parser.AsToken(value)
		if !ok {
			return nil
		}
		tokens = append(tokens, anbToken{token: token, spaced: spaced})
		spaced = false
	}
	return tokens
}

type anbParser struct {
	tokens []anbToken
	pos    int
}

func (state *anbParser) parse() (AnB, bool) {
	first := state.tokens[0].token
	switch first.Type {
	case tokenizer.TokenIdent:
		return state.parseIdentStart()
	case tokenizer.TokenDimension:
		return state.parseDimensionStart()
	case tokenizer.TokenNumber:
		if len(state.tokens) == 1 && isIntegerToken(first) {
			return AnB{A: 0, B: int(first.Number)}, true
		}
		return AnB{}, false
	case tokenizer.TokenDelim:
		if first.Value == "+" {
			return state.parsePlusStart()
		}
		return AnB{}, false
	default:
		return AnB{}, false
	}
}

func (state *anbParser) parseIdentStart() (AnB, bool) {
	name := strings.ToLower(state.tokens[0].token.Value)
	switch name {
	case "odd":
		if len(state.tokens) != 1 {
			return AnB{}, false
		}
		return AnB{A: 2, B: 1}, true
	case "even":
		if len(state.tokens) != 1 {
			return AnB{}, false
		}
		return AnB{A: 2, B: 0}, true
	case "n":
		return state.parseBPart(1, 1)
	case "-n":
		return state.parseBPart(1, -1)
	case "n-":
		return state.parseNegatedSignless(1, 1)
	case "-n-":
		return state.parseNegatedSignless(1, -1)
	}
	if offset, ok := digitsSuffix(name, "n-"); ok {
		if len(state.tokens) != 1 {
			return AnB{}, false
		}
		return AnB{A: 1, B: -parseDecimal(offset)}, true
	}
	if offset, ok := digitsSuffix(name, "-n-"); ok {
		if len(state.tokens) != 1 {
			return AnB{}, false
		}
		return AnB{A: -1, B: -parseDecimal(offset)}, true
	}
	return AnB{}, false
}

func (state *anbParser) parseDimensionStart() (AnB, bool) {
	token := state.tokens[0].token
	if token.Type != tokenizer.TokenDimension || token.NumericKind != tokenizer.Integer {
		return AnB{}, false
	}
	unit := strings.ToLower(token.Value)
	a := int(token.Number)
	switch {
	case unit == "n":
		return state.parseBPart(1, a)
	case unit == "n-":
		return state.parseNegatedSignless(1, a)
	}
	if offset, ok := digitsSuffix(unit, "n-"); ok {
		if len(state.tokens) != 1 {
			return AnB{}, false
		}
		return AnB{A: a, B: -parseDecimal(offset)}, true
	}
	return AnB{}, false
}

func (state *anbParser) parsePlusStart() (AnB, bool) {
	if len(state.tokens) < 2 {
		return AnB{}, false
	}
	next := state.tokens[1]
	if next.token.Type != tokenizer.TokenIdent || next.spaced {
		return AnB{}, false
	}
	name := strings.ToLower(next.token.Value)
	switch {
	case name == "n":
		return state.parseBPart(2, 1)
	case name == "n-":
		return state.parseNegatedSignless(2, 1)
	}
	if offset, ok := digitsSuffix(name, "n-"); ok {
		if len(state.tokens) != 2 {
			return AnB{}, false
		}
		return AnB{A: 1, B: -parseDecimal(offset)}, true
	}
	return AnB{}, false
}

// parseBPart parses the optional B after an "n" occurrence at index.
func (state *anbParser) parseBPart(index, a int) (AnB, bool) {
	if index >= len(state.tokens) {
		return AnB{A: a, B: 0}, true
	}
	token := state.tokens[index].token
	if token.Type == tokenizer.TokenNumber && isIntegerToken(token) && isSignedInteger(token) {
		if index+1 != len(state.tokens) {
			return AnB{}, false
		}
		return AnB{A: a, B: int(token.Number)}, true
	}
	if token.Type == tokenizer.TokenDelim && (token.Value == "+" || token.Value == "-") {
		if index+1 != len(state.tokens)-1 {
			return AnB{}, false
		}
		integer := state.tokens[index+1].token
		if integer.Type != tokenizer.TokenNumber || !isIntegerToken(integer) || !isSignlessInteger(integer) {
			return AnB{}, false
		}
		b := int(integer.Number)
		if token.Value == "-" {
			b = -b
		}
		return AnB{A: a, B: b}, true
	}
	return AnB{}, false
}

// parseNegatedSignless parses "<n-> <signless-integer>".
func (state *anbParser) parseNegatedSignless(index, a int) (AnB, bool) {
	if index+1 != len(state.tokens) {
		return AnB{}, false
	}
	integer := state.tokens[index].token
	if integer.Type != tokenizer.TokenNumber || !isIntegerToken(integer) || !isSignlessInteger(integer) {
		return AnB{}, false
	}
	return AnB{A: a, B: -int(integer.Number)}, true
}

func isIntegerToken(token tokenizer.Token) bool {
	return token.Type == tokenizer.TokenNumber && token.NumericKind == tokenizer.Integer
}

func isSignedInteger(token tokenizer.Token) bool {
	return strings.HasPrefix(token.Representation, "+") || strings.HasPrefix(token.Representation, "-")
}

func isSignlessInteger(token tokenizer.Token) bool {
	if len(token.Representation) == 0 {
		return false
	}
	return token.Representation[0] >= '0' && token.Representation[0] <= '9'
}

// digitsSuffix returns the digit suffix after prefix, or ("", false).
func digitsSuffix(value, prefix string) (string, bool) {
	if !strings.HasPrefix(value, prefix) {
		return "", false
	}
	suffix := value[len(prefix):]
	if suffix == "" {
		return "", false
	}
	for index := 0; index < len(suffix); index++ {
		if suffix[index] < '0' || suffix[index] > '9' {
			return "", false
		}
	}
	return suffix, true
}

func parseDecimal(value string) int {
	result := 0
	for index := 0; index < len(value); index++ {
		result = result*10 + int(value[index]-'0')
	}
	return result
}
