package tokenizer

import (
	"errors"
	"strconv"
)

// Options controls optional tokenizer behavior.
type Options struct {
	// UnicodeRangesAllowed enables <unicode-range-token> consumption. It is
	// false during normal tokenization and true only for the unicode-range
	// descriptor. See CSS Syntax 3 §4.3.14.
	UnicodeRangesAllowed bool
}

// Tokenizer consumes code points and produces tokens on demand.
type Tokenizer struct {
	input    []rune
	position int
	current  rune
	options  Options
}

// New returns a tokenizer over input. The input is preprocessed first.
func New(input string) *Tokenizer {
	return NewWithOptions(input, Options{})
}

// NewWithOptions returns a tokenizer over input with the given options.
func NewWithOptions(input string, options Options) *Tokenizer {
	return NewFromRunesWithOptions(Preprocess([]rune(input)), options)
}

// NewFromRunes returns a tokenizer over already preprocessed code points.
// Callers that pass raw code points should use New or Preprocess first.
func NewFromRunes(input []rune) *Tokenizer {
	return NewFromRunesWithOptions(input, Options{})
}

// NewFromRunesWithOptions returns a tokenizer over already preprocessed code
// points with the given options.
func NewFromRunesWithOptions(input []rune, options Options) *Tokenizer {
	return &Tokenizer{input: input, options: options}
}

// Tokenize returns every token in input, including the trailing EOF token.
func Tokenize(input string) []Token {
	return TokenizeWithOptions(input, Options{})
}

// TokenizeWithOptions returns every token in input with the given options.
func TokenizeWithOptions(input string, options Options) []Token {
	tokenizer := NewWithOptions(input, options)
	tokens := make([]Token, 0, len(input)/2+1)
	for {
		token := tokenizer.Next()
		tokens = append(tokens, token)
		if token.Type == TokenEOF {
			return tokens
		}
	}
}

// Next consumes and returns the next token from the input stream. At the end
// of the stream it returns an EOF token on every call.
func (tokenizer *Tokenizer) Next() Token {
	tokenizer.consumeComments()
	start := tokenizer.position
	current := tokenizer.nextInput()

	switch {
	case current == eofCodePoint:
		return tokenizer.newToken(TokenEOF, start)
	case isWhitespace(current):
		tokenizer.consumeWhitespace()
		return tokenizer.newToken(TokenWhitespace, start)
	case current == '"':
		return tokenizer.finishString(start, '"')
	case current == '\'':
		return tokenizer.finishString(start, '\'')
	case current == '#':
		if isIdentifierCodePoint(tokenizer.peek(0)) || isValidEscape(tokenizer.peek(0), tokenizer.peek(1)) {
			hashFlag := HashUnrestricted
			if wouldStartIdentifier(tokenizer.peek(0), tokenizer.peek(1), tokenizer.peek(2)) {
				hashFlag = HashID
			}
			value := tokenizer.consumeIdentifier()
			token := tokenizer.newToken(TokenHash, start)
			token.Value = value
			token.HashFlag = hashFlag
			return token
		}
		return tokenizer.newDelim(start, current)
	case current == '(':
		return tokenizer.newToken(TokenOpenParen, start)
	case current == ')':
		return tokenizer.newToken(TokenCloseParen, start)
	case current == '+':
		if wouldStartNumber(current, tokenizer.peek(0), tokenizer.peek(1)) {
			tokenizer.reconsume()
			return tokenizer.consumeNumericToken(start)
		}
		return tokenizer.newDelim(start, current)
	case current == ',':
		return tokenizer.newToken(TokenComma, start)
	case current == '-':
		if wouldStartNumber(current, tokenizer.peek(0), tokenizer.peek(1)) {
			tokenizer.reconsume()
			return tokenizer.consumeNumericToken(start)
		}
		if tokenizer.peek(0) == '-' && tokenizer.peek(1) == '>' {
			tokenizer.nextInput()
			tokenizer.nextInput()
			return tokenizer.newToken(TokenCDC, start)
		}
		if wouldStartIdentifier(current, tokenizer.peek(0), tokenizer.peek(1)) {
			tokenizer.reconsume()
			return tokenizer.consumeIdentLike(start)
		}
		return tokenizer.newDelim(start, current)
	case current == '.':
		if wouldStartNumber(current, tokenizer.peek(0), tokenizer.peek(1)) {
			tokenizer.reconsume()
			return tokenizer.consumeNumericToken(start)
		}
		return tokenizer.newDelim(start, current)
	case current == ':':
		return tokenizer.newToken(TokenColon, start)
	case current == ';':
		return tokenizer.newToken(TokenSemicolon, start)
	case current == '<':
		if tokenizer.peek(0) == '!' && tokenizer.peek(1) == '-' && tokenizer.peek(2) == '-' {
			tokenizer.nextInput()
			tokenizer.nextInput()
			tokenizer.nextInput()
			return tokenizer.newToken(TokenCDO, start)
		}
		return tokenizer.newDelim(start, current)
	case current == '@':
		if wouldStartIdentifier(tokenizer.peek(0), tokenizer.peek(1), tokenizer.peek(2)) {
			value := tokenizer.consumeIdentifier()
			token := tokenizer.newToken(TokenAtKeyword, start)
			token.Value = value
			return token
		}
		return tokenizer.newDelim(start, current)
	case current == '[':
		return tokenizer.newToken(TokenOpenSquare, start)
	case current == '\\':
		if isValidEscape(current, tokenizer.peek(0)) {
			tokenizer.reconsume()
			return tokenizer.consumeIdentLike(start)
		}
		return tokenizer.newDelim(start, current)
	case current == ']':
		return tokenizer.newToken(TokenCloseSquare, start)
	case current == '{':
		return tokenizer.newToken(TokenOpenCurly, start)
	case current == '}':
		return tokenizer.newToken(TokenCloseCurly, start)
	case isDigit(current):
		tokenizer.reconsume()
		return tokenizer.consumeNumericToken(start)
	case (current == 'U' || current == 'u') && tokenizer.options.UnicodeRangesAllowed &&
		wouldStartUnicodeRange(current, tokenizer.peek(0), tokenizer.peek(1)):
		tokenizer.reconsume()
		return tokenizer.consumeUnicodeRangeToken(start)
	case isIdentifierStart(current):
		tokenizer.reconsume()
		return tokenizer.consumeIdentLike(start)
	default:
		return tokenizer.newDelim(start, current)
	}
}

func (tokenizer *Tokenizer) finishString(start int, ending rune) Token {
	tokenType, value := tokenizer.consumeString(ending)
	token := tokenizer.newToken(tokenType, start)
	token.Value = value
	return token
}

func (tokenizer *Tokenizer) consumeIdentLike(start int) Token {
	name := tokenizer.consumeIdentifier()
	if asciiEqualFold(name, "url") && tokenizer.peek(0) == '(' {
		tokenizer.nextInput()
		for isWhitespace(tokenizer.peek(0)) && isWhitespace(tokenizer.peek(1)) {
			tokenizer.nextInput()
		}
		next := tokenizer.peek(0)
		afterNext := tokenizer.peek(1)
		isQuoted := next == '"' || next == '\'' || (isWhitespace(next) && (afterNext == '"' || afterNext == '\''))
		if isQuoted {
			token := tokenizer.newToken(TokenFunction, start)
			token.Value = name
			return token
		}
		return tokenizer.consumeURLToken(start)
	}
	if tokenizer.peek(0) == '(' {
		tokenizer.nextInput()
		token := tokenizer.newToken(TokenFunction, start)
		token.Value = name
		return token
	}
	token := tokenizer.newToken(TokenIdent, start)
	token.Value = name
	return token
}

func (tokenizer *Tokenizer) consumeNumericToken(start int) Token {
	value, numericKind, representation := tokenizer.consumeNumber()
	if wouldStartIdentifier(tokenizer.peek(0), tokenizer.peek(1), tokenizer.peek(2)) {
		unit := tokenizer.consumeIdentifier()
		token := tokenizer.newToken(TokenDimension, start)
		token.Number = value
		token.NumberRepresentation = representation
		token.NumericKind = numericKind
		token.Value = unit
		return token
	}
	if tokenizer.peek(0) == '%' {
		tokenizer.nextInput()
		token := tokenizer.newToken(TokenPercentage, start)
		token.Number = value
		token.NumberRepresentation = representation
		token.NumericKind = numericKind
		return token
	}
	token := tokenizer.newToken(TokenNumber, start)
	token.Number = value
	token.NumberRepresentation = representation
	token.NumericKind = numericKind
	return token
}

// consumeUnicodeRangeToken consumes a <unicode-range-token>. The current input
// code point has already been reconsumed. See CSS Syntax 3 §4.3.14.
func (tokenizer *Tokenizer) consumeUnicodeRangeToken(start int) Token {
	tokenizer.nextInput() // U
	tokenizer.nextInput() // +
	digits := make([]rune, 0, 6)
	for len(digits) < 6 && isHexDigit(tokenizer.peek(0)) {
		digits = append(digits, tokenizer.nextInput())
	}
	if len(digits) < 6 {
		for len(digits) < 6 && tokenizer.peek(0) == '?' {
			digits = append(digits, tokenizer.nextInput())
		}
	}
	hasQuestion := false
	for _, digit := range digits {
		if digit == '?' {
			hasQuestion = true
			break
		}
	}
	rangeStart := 0
	rangeEnd := 0
	if hasQuestion {
		rangeStart = parseHexRun(digits, '0')
		rangeEnd = parseHexRun(digits, 'F')
	} else {
		rangeStart = parseHexRun(digits, 0)
		rangeEnd = rangeStart
		if tokenizer.peek(0) == '-' && isHexDigit(tokenizer.peek(1)) {
			tokenizer.nextInput()
			endDigits := make([]rune, 0, 6)
			for len(endDigits) < 6 && isHexDigit(tokenizer.peek(0)) {
				endDigits = append(endDigits, tokenizer.nextInput())
			}
			rangeEnd = parseHexRun(endDigits, 0)
		}
	}
	token := tokenizer.newToken(TokenUnicodeRange, start)
	token.RangeStart = rangeStart
	token.RangeEnd = rangeEnd
	return token
}

// parseHexRun interprets a run of hex digits, substituting replacement for '?'.
func parseHexRun(digits []rune, replacement rune) int {
	value := 0
	for _, digit := range digits {
		if digit == '?' {
			digit = replacement
		}
		value = value*16 + hexValue(digit)
	}
	return value
}

func hexValue(digit rune) int {
	switch {
	case digit >= '0' && digit <= '9':
		return int(digit - '0')
	case digit >= 'a' && digit <= 'f':
		return int(digit-'a') + 10
	case digit >= 'A' && digit <= 'F':
		return int(digit-'A') + 10
	default:
		return 0
	}
}

// consumeString consumes a string token that ends at ending, defaulting to the
// current input code point. It returns TokenString or TokenBadString.
func (tokenizer *Tokenizer) consumeString(ending rune) (TokenType, string) {
	result := make([]rune, 0, 16)
	for {
		current := tokenizer.nextInput()
		switch {
		case current == ending:
			return TokenString, string(result)
		case current == eofCodePoint:
			return TokenString, string(result)
		case current == newlineCodePoint:
			tokenizer.reconsume()
			return TokenBadString, string(result)
		case current == '\\':
			next := tokenizer.peek(0)
			if next == eofCodePoint {
				continue
			}
			if next == newlineCodePoint {
				tokenizer.nextInput()
				continue
			}
			result = append(result, tokenizer.consumeEscapedCodePoint())
		default:
			result = append(result, current)
		}
	}
}

// consumeURLToken assumes the initial "url(" has already been consumed.
func (tokenizer *Tokenizer) consumeURLToken(start int) Token {
	result := make([]rune, 0, 16)
	tokenizer.consumeWhitespace()
	for {
		current := tokenizer.nextInput()
		switch {
		case current == ')':
			return tokenizer.urlToken(TokenURL, start, result)
		case current == eofCodePoint:
			return tokenizer.urlToken(TokenURL, start, result)
		case isWhitespace(current):
			tokenizer.consumeWhitespace()
			if tokenizer.peek(0) == ')' {
				tokenizer.nextInput()
				return tokenizer.urlToken(TokenURL, start, result)
			}
			if tokenizer.peek(0) == eofCodePoint {
				return tokenizer.urlToken(TokenURL, start, result)
			}
			tokenizer.consumeRemnantsOfBadURL()
			return tokenizer.urlToken(TokenBadURL, start, result)
		case current == '"' || current == '\'' || current == '(' || isNonPrintable(current):
			tokenizer.consumeRemnantsOfBadURL()
			return tokenizer.urlToken(TokenBadURL, start, result)
		case current == '\\':
			if isValidEscape(current, tokenizer.peek(0)) {
				result = append(result, tokenizer.consumeEscapedCodePoint())
				continue
			}
			tokenizer.consumeRemnantsOfBadURL()
			return tokenizer.urlToken(TokenBadURL, start, result)
		default:
			result = append(result, current)
		}
	}
}

func (tokenizer *Tokenizer) urlToken(tokenType TokenType, start int, value []rune) Token {
	token := tokenizer.newToken(tokenType, start)
	token.Value = string(value)
	return token
}

func (tokenizer *Tokenizer) consumeComments() {
	for tokenizer.peek(0) == '/' && tokenizer.peek(1) == '*' {
		tokenizer.nextInput()
		tokenizer.nextInput()
		for {
			if tokenizer.peek(0) == eofCodePoint {
				return
			}
			if tokenizer.peek(0) == '*' && tokenizer.peek(1) == '/' {
				tokenizer.nextInput()
				tokenizer.nextInput()
				break
			}
			tokenizer.nextInput()
		}
	}
}

func (tokenizer *Tokenizer) consumeWhitespace() {
	for isWhitespace(tokenizer.peek(0)) {
		tokenizer.nextInput()
	}
}

// consumeIdentifier returns the largest identifier that can be formed at the
// current position. Verification of the first code points is the caller's
// responsibility.
func (tokenizer *Tokenizer) consumeIdentifier() string {
	result := make([]rune, 0, 16)
	for {
		current := tokenizer.peek(0)
		switch {
		case isIdentifierCodePoint(current):
			result = append(result, tokenizer.nextInput())
		case isValidEscape(current, tokenizer.peek(1)):
			tokenizer.nextInput()
			result = append(result, tokenizer.consumeEscapedCodePoint())
		default:
			return string(result)
		}
	}
}

// consumeEscapedCodePoint assumes the backslash has been consumed and the next
// code point is part of a valid escape.
func (tokenizer *Tokenizer) consumeEscapedCodePoint() rune {
	current := tokenizer.nextInput()
	if isHexDigit(current) {
		digits := []rune{current}
		for len(digits) < 6 && isHexDigit(tokenizer.peek(0)) {
			digits = append(digits, tokenizer.nextInput())
		}
		if isWhitespace(tokenizer.peek(0)) {
			tokenizer.nextInput()
		}
		value, err := strconv.ParseInt(string(digits), 16, 32)
		if err != nil || value == 0 || isSurrogate(rune(value)) || value > 0x10FFFF {
			return replacementCodePoint
		}
		return rune(value)
	}
	if current == eofCodePoint {
		return replacementCodePoint
	}
	return current
}

// consumeNumber returns the numeric value, integer/number kind, and the source
// representation of the number at the current position. Verification that the
// input starts with a number is the caller's responsibility.
func (tokenizer *Tokenizer) consumeNumber() (float64, NumericKind, string) {
	numericKind := Integer
	representation := make([]rune, 0, 16)

	if current := tokenizer.peek(0); current == '+' || current == '-' {
		representation = append(representation, tokenizer.nextInput())
	}
	for isDigit(tokenizer.peek(0)) {
		representation = append(representation, tokenizer.nextInput())
	}
	if tokenizer.peek(0) == '.' && isDigit(tokenizer.peek(1)) {
		representation = append(representation, tokenizer.nextInput())
		representation = append(representation, tokenizer.nextInput())
		numericKind = Number
		for isDigit(tokenizer.peek(0)) {
			representation = append(representation, tokenizer.nextInput())
		}
	}
	if isExponentIndicator(tokenizer.peek(0)) {
		next := tokenizer.peek(1)
		afterNext := tokenizer.peek(2)
		hasExponent := isDigit(next) || ((next == '+' || next == '-') && isDigit(afterNext))
		if hasExponent {
			representation = append(representation, tokenizer.nextInput())
			representation = append(representation, tokenizer.nextInput())
			if next == '+' || next == '-' {
				representation = append(representation, tokenizer.nextInput())
			}
			numericKind = Number
			for isDigit(tokenizer.peek(0)) {
				representation = append(representation, tokenizer.nextInput())
			}
		}
	}
	return convertStringToNumber(string(representation)), numericKind, string(representation)
}

// convertStringToNumber implements CSS Syntax 3 §4.3.13. It assumes the string
// is a valid CSS number.
func convertStringToNumber(representation string) float64 {
	value, err := strconv.ParseFloat(representation, 64)
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return 0
	}
	return value
}

func (tokenizer *Tokenizer) consumeRemnantsOfBadURL() {
	for {
		current := tokenizer.nextInput()
		if current == ')' || current == eofCodePoint {
			return
		}
		if isValidEscape(current, tokenizer.peek(0)) {
			tokenizer.consumeEscapedCodePoint()
		}
	}
}

func (tokenizer *Tokenizer) nextInput() rune {
	if tokenizer.position < len(tokenizer.input) {
		tokenizer.current = tokenizer.input[tokenizer.position]
		tokenizer.position++
		return tokenizer.current
	}
	tokenizer.current = eofCodePoint
	return eofCodePoint
}

func (tokenizer *Tokenizer) reconsume() {
	if tokenizer.current != eofCodePoint {
		tokenizer.position--
	}
}

func (tokenizer *Tokenizer) peek(offset int) rune {
	index := tokenizer.position + offset
	if index < 0 || index >= len(tokenizer.input) {
		return eofCodePoint
	}
	return tokenizer.input[index]
}

func (tokenizer *Tokenizer) newToken(tokenType TokenType, start int) Token {
	return Token{
		Type:           tokenType,
		Representation: string(tokenizer.input[start:tokenizer.position]),
		Start:          start,
		End:            tokenizer.position,
	}
}

func (tokenizer *Tokenizer) newDelim(start int, value rune) Token {
	token := tokenizer.newToken(TokenDelim, start)
	token.Value = string(value)
	return token
}
