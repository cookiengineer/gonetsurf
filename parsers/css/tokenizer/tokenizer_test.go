package tokenizer

import (
	"math"
	"testing"
)

func TestPreprocess(t *testing.T) {
	tests := []struct {
		input []rune
		want  string
	}{
		{[]rune("plain"), "plain"},
		{[]rune("a\rb"), "a\nb"},
		{[]rune("a\r\nb"), "a\nb"},
		{[]rune("a\fb"), "a\nb"},
		{[]rune{'a', 0, 'b'}, "a\uFFFDb"},
		{[]rune{'a', 0xD800, 'b'}, "a\uFFFDb"},
		{[]rune{'a', 0xDFFF, 'b'}, "a\uFFFDb"},
		{[]rune("café \U0001F600"), "café \U0001F600"},
	}
	for _, test := range tests {
		if got := string(Preprocess(test.input)); got != test.want {
			t.Errorf("Preprocess(%q) = %q, want %q", string(test.input), got, test.want)
		}
	}
}

func TestPreprocessDoesNotAliasInput(t *testing.T) {
	input := []rune("a\r\nb")
	result := Preprocess(input)
	result[0] = 'z'
	if input[0] != 'a' {
		t.Error("Preprocess modified its input")
	}
}

func TestTokenizeRepresentation(t *testing.T) {
	tests := []struct {
		input          string
		tokenType      TokenType
		value          string
		number         float64
		representation string
	}{
		{".009", TokenNumber, "", 0.009, ".009"},
		{"9e-3", TokenNumber, "", 0.009, "9e-3"},
		{"1e3", TokenNumber, "", 1000, "1e3"},
		{"+42", TokenNumber, "", 42, "+42"},
		{"2n+1", TokenDimension, "n", 2, "2n"},
		{"12.5px", TokenDimension, "px", 12.5, "12.5px"},
		{"50%", TokenPercentage, "", 50, "50%"},
		{"\\31 23", TokenIdent, "123", 0, "\\31 23"},
		{"url(foo)", TokenURL, "foo", 0, "url(foo)"},
		{"'he\\6c lo'", TokenString, "hello", 0, "'he\\6c lo'"},
	}
	for _, test := range tests {
		tokens := Tokenize(test.input)
		token := tokens[0]
		if token.Type != test.tokenType {
			t.Errorf("Tokenize(%q)[0].Type = %s, want %s", test.input, token.Type, test.tokenType)
			continue
		}
		if token.Value != test.value {
			t.Errorf("Tokenize(%q)[0].Value = %q, want %q", test.input, token.Value, test.value)
		}
		if test.number != 0 && math.Abs(token.Number-test.number) > 1e-12 {
			t.Errorf("Tokenize(%q)[0].Number = %v, want %v", test.input, token.Number, test.number)
		}
		if token.Representation != test.representation {
			t.Errorf("Tokenize(%q)[0].Representation = %q, want %q", test.input, token.Representation, test.representation)
		}
		if token.Start != 0 || token.End != len([]rune(test.representation)) {
			t.Errorf("Tokenize(%q)[0] offsets = [%d,%d), want [0,%d)", test.input, token.Start, token.End, len([]rune(test.representation)))
		}
	}
}

func TestTokenizeNumericKinds(t *testing.T) {
	tests := []struct {
		input string
		kind  NumericKind
	}{
		{"10", Integer},
		{"12.0", Number},
		{"10e0", Number},
		{"100", Integer},
	}
	for _, test := range tests {
		token := Tokenize(test.input)[0]
		if token.NumericKind != test.kind {
			t.Errorf("Tokenize(%q)[0].NumericKind = %v, want %v", test.input, token.NumericKind, test.kind)
		}
	}
}

func TestHashFlag(t *testing.T) {
	tests := []struct {
		input string
		flag  HashFlag
		value string
	}{
		{"#id-selector", HashID, "id-selector"},
		{"#123", HashUnrestricted, "123"},
		{"#-a", HashID, "-a"},
	}
	for _, test := range tests {
		token := Tokenize(test.input)[0]
		if token.Type != TokenHash {
			t.Errorf("Tokenize(%q)[0].Type = %s, want hash", test.input, token.Type)
			continue
		}
		if token.HashFlag != test.flag {
			t.Errorf("Tokenize(%q)[0].HashFlag = %v, want %v", test.input, token.HashFlag, test.flag)
		}
		if token.Value != test.value {
			t.Errorf("Tokenize(%q)[0].Value = %q, want %q", test.input, token.Value, test.value)
		}
	}
}

func TestCommentsAreSkipped(t *testing.T) {
	tokens := Tokenize("a/* comment */b")
	if len(tokens) != 3 {
		t.Fatalf("got %d tokens, want 3: %v", len(tokens), tokens)
	}
	if tokens[0].Value != "a" || tokens[1].Value != "b" {
		t.Errorf("identifiers = %q, %q; want a, b", tokens[0].Value, tokens[1].Value)
	}
	if tokens[1].Start != 14 {
		t.Errorf("second token starts at %d, want 14", tokens[1].Start)
	}
}

func TestBadStringStopsAtNewline(t *testing.T) {
	tokens := Tokenize("\"unterminated\nnext")
	if tokens[0].Type != TokenBadString {
		t.Fatalf("first token = %s, want bad-string", tokens[0].Type)
	}
	if tokens[0].Value != "unterminated" {
		t.Errorf("bad string value = %q", tokens[0].Value)
	}
}

func TestNextAtEOFIsStable(t *testing.T) {
	tokenizer := New("")
	for attempt := 0; attempt < 3; attempt++ {
		token := tokenizer.Next()
		if token.Type != TokenEOF {
			t.Fatalf("attempt %d: got %s, want EOF", attempt, token.Type)
		}
		if token.Representation != "" {
			t.Fatalf("attempt %d: EOF representation = %q", attempt, token.Representation)
		}
	}
}

func TestUnicodeRangeToken(t *testing.T) {
	tests := []struct {
		input string
		start int
		end   int
		repr  string
	}{
		{"u+1", 1, 1, "u+1"},
		{"U+1-2", 1, 2, "U+1-2"},
		{"u+??", 0, 255, "u+??"},
		{"U+10????", 0x100000, 0x10FFFF, "U+10????"},
		{"u+1?-50", 0x10, 0x1F, "u+1?"},
	}
	for _, test := range tests {
		tokens := TokenizeWithOptions(test.input, Options{UnicodeRangesAllowed: true})
		token := tokens[0]
		if token.Type != TokenUnicodeRange {
			t.Errorf("TokenizeWithOptions(%q)[0].Type = %s, want unicode-range", test.input, token.Type)
			continue
		}
		if token.RangeStart != test.start || token.RangeEnd != test.end {
			t.Errorf("TokenizeWithOptions(%q)[0] range = [%d,%d], want [%d,%d]", test.input, token.RangeStart, token.RangeEnd, test.start, test.end)
		}
		if token.Representation != test.repr {
			t.Errorf("TokenizeWithOptions(%q)[0].Representation = %q, want %q", test.input, token.Representation, test.repr)
		}
	}
}

func TestUnicodeRangeDisabledByDefault(t *testing.T) {
	tokens := Tokenize("u+1-2")
	if tokens[0].Type == TokenUnicodeRange {
		t.Error("unicode range should not be produced by default tokenization")
	}
}

func TestTokenTypeString(t *testing.T) {
	tests := map[TokenType]string{
		TokenEOF:        "EOF",
		TokenIdent:      "ident",
		TokenDimension:  "dimension",
		TokenCloseCurly: "close-curly",
	}
	for tokenType, want := range tests {
		if got := tokenType.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", tokenType, got, want)
		}
	}
}

func TestConvertStringToNumber(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"12", 12},
		{"-7", -7},
		{"+45.6", 45.6},
		{".7", 0.7},
		{"12E-1", 1.2},
		{"3e+1", 30},
	}
	for _, test := range tests {
		if got := convertStringToNumber(test.input); got != test.want {
			t.Errorf("convertStringToNumber(%q) = %v, want %v", test.input, got, test.want)
		}
	}
}
