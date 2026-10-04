package tokenizer_test

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"

	"gonetsurf/parsers/css/tokenizer"
)

// testdata/tokenizer-tests.json is converted from tabatkins/parse-css
// tests.js at revision 3dfe12a9bb63b8762c3bfd37db4e3593aa39bd5e, filtered to
// the tokenization cases (parser == ""). That repository is the reference
// implementation accompanying CSS Syntax 3 and is distributed under the MIT
// license. The expected values describe the preprocessed token stream,
// including the trailing EOF token.
type corpusCase struct {
	CSS      string        `json:"css"`
	Expected []corpusToken `json:"expected"`
}

type corpusToken struct {
	Type      string  `json:"type"`
	Value     any     `json:"value"`
	IsInteger *bool   `json:"isInteger"`
	IsIdent   *bool   `json:"isIdent"`
	Unit      *string `json:"unit"`
	Sign      *string `json:"sign"`
}

var corpusTokenTypes = map[string]tokenizer.TokenType{
	"EOF":          tokenizer.TokenEOF,
	"IDENT":        tokenizer.TokenIdent,
	"FUNCTION":     tokenizer.TokenFunction,
	"AT-KEYWORD":   tokenizer.TokenAtKeyword,
	"HASH":         tokenizer.TokenHash,
	"STRING":       tokenizer.TokenString,
	"BADSTRING":    tokenizer.TokenBadString,
	"URL":          tokenizer.TokenURL,
	"BADURL":       tokenizer.TokenBadURL,
	"DELIM":        tokenizer.TokenDelim,
	"NUMBER":       tokenizer.TokenNumber,
	"PERCENTAGE":   tokenizer.TokenPercentage,
	"DIMENSION":    tokenizer.TokenDimension,
	"WHITESPACE":   tokenizer.TokenWhitespace,
	"CDO":          tokenizer.TokenCDO,
	"CDC":          tokenizer.TokenCDC,
	"COLON":        tokenizer.TokenColon,
	"SEMICOLON":    tokenizer.TokenSemicolon,
	"COMMA":        tokenizer.TokenComma,
	"OPEN-PAREN":   tokenizer.TokenOpenParen,
	"CLOSE-PAREN":  tokenizer.TokenCloseParen,
	"OPEN-SQUARE":  tokenizer.TokenOpenSquare,
	"CLOSE-SQUARE": tokenizer.TokenCloseSquare,
	"OPEN-CURLY":   tokenizer.TokenOpenCurly,
	"CLOSE-CURLY":  tokenizer.TokenCloseCurly,
}

func TestTokenizerConformance(t *testing.T) {
	encoded, err := os.ReadFile("testdata/tokenizer-tests.json")
	if err != nil {
		t.Fatalf("read corpus: %v", err)
	}
	var cases []corpusCase
	if err := json.Unmarshal(encoded, &cases); err != nil {
		t.Fatalf("decode corpus: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("corpus is empty")
	}

	for index, test := range cases {
		tokens := tokenizer.Tokenize(test.CSS)
		if len(tokens) != len(test.Expected) {
			t.Errorf("case %d %q: got %d tokens, want %d\n got: %s",
				index, test.CSS, len(tokens), len(test.Expected), describeTokens(tokens))
			continue
		}
		for tokenIndex, expected := range test.Expected {
			if problem := compareToken(tokens[tokenIndex], expected); problem != "" {
				t.Errorf("case %d %q token %d: %s\n got: %s",
					index, test.CSS, tokenIndex, problem, describeToken(tokens[tokenIndex]))
			}
		}
	}
}

func compareToken(actual tokenizer.Token, expected corpusToken) string {
	expectedType, ok := corpusTokenTypes[expected.Type]
	if !ok {
		return fmt.Sprintf("unknown expected type %q", expected.Type)
	}
	if actual.Type != expectedType {
		return fmt.Sprintf("type = %s, want %s", actual.Type, expectedType)
	}
	if expected.Sign != nil && !strings.HasPrefix(actual.Representation, *expected.Sign) {
		return fmt.Sprintf("representation %q does not start with sign %q", actual.Representation, *expected.Sign)
	}
	switch expectedType {
	case tokenizer.TokenIdent, tokenizer.TokenFunction, tokenizer.TokenAtKeyword,
		tokenizer.TokenString, tokenizer.TokenURL, tokenizer.TokenBadURL, tokenizer.TokenDelim:
		if value, ok := expected.Value.(string); ok && actual.Value != value {
			return fmt.Sprintf("value = %q, want %q", actual.Value, value)
		}
	case tokenizer.TokenHash:
		if value, ok := expected.Value.(string); ok && actual.Value != value {
			return fmt.Sprintf("value = %q, want %q", actual.Value, value)
		}
		if expected.IsIdent != nil {
			wantID := *expected.IsIdent
			if (actual.HashFlag == tokenizer.HashID) != wantID {
				return fmt.Sprintf("id flag = %v, want %v", actual.HashFlag == tokenizer.HashID, wantID)
			}
		}
	case tokenizer.TokenNumber, tokenizer.TokenPercentage, tokenizer.TokenDimension:
		if value, ok := expected.Value.(float64); ok && !nearlyEqual(actual.Number, value) {
			return fmt.Sprintf("number = %v, want %v", actual.Number, value)
		}
		if expected.IsInteger != nil {
			wantNumber := !*expected.IsInteger
			if (actual.NumericKind == tokenizer.Number) != wantNumber {
				return fmt.Sprintf("numeric kind = %v, want integer=%v", actual.NumericKind, *expected.IsInteger)
			}
		}
		if expectedType == tokenizer.TokenDimension && expected.Unit != nil && actual.Value != *expected.Unit {
			return fmt.Sprintf("unit = %q, want %q", actual.Value, *expected.Unit)
		}
	}
	return ""
}

func nearlyEqual(actual, expected float64) bool {
	if actual == expected {
		return true
	}
	scale := math.Max(1, math.Abs(expected))
	return math.Abs(actual-expected) <= scale*1e-12
}

func describeTokens(tokens []tokenizer.Token) string {
	descriptions := make([]string, len(tokens))
	for index, token := range tokens {
		descriptions[index] = describeToken(token)
	}
	return strings.Join(descriptions, ", ")
}

func describeToken(token tokenizer.Token) string {
	return fmt.Sprintf("%s(%q)", token.Type, token.Representation)
}
