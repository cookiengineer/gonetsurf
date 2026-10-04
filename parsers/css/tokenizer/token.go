package tokenizer

import "strconv"

// TokenType identifies one of the token kinds produced by the CSS Syntax 3
// tokenizer. See CSS Syntax 3 §4.
type TokenType uint8

const (
	// TokenEOF marks the end of the input stream.
	TokenEOF TokenType = iota
	// TokenIdent is an <ident-token>.
	TokenIdent
	// TokenFunction is a <function-token>.
	TokenFunction
	// TokenAtKeyword is an <at-keyword-token>.
	TokenAtKeyword
	// TokenHash is a <hash-token>.
	TokenHash
	// TokenString is a <string-token>.
	TokenString
	// TokenBadString is a <bad-string-token>.
	TokenBadString
	// TokenURL is a <url-token>.
	TokenURL
	// TokenBadURL is a <bad-url-token>.
	TokenBadURL
	// TokenDelim is a <delim-token>.
	TokenDelim
	// TokenNumber is a <number-token>.
	TokenNumber
	// TokenPercentage is a <percentage-token>.
	TokenPercentage
	// TokenDimension is a <dimension-token>.
	TokenDimension
	// TokenWhitespace is a <whitespace-token>.
	TokenWhitespace
	// TokenCDO is a <CDO-token> ("<!--").
	TokenCDO
	// TokenCDC is a <CDC-token> ("-->").
	TokenCDC
	// TokenColon is a <colon-token> (":").
	TokenColon
	// TokenSemicolon is a <semicolon-token> (";").
	TokenSemicolon
	// TokenComma is a <comma-token> (",").
	TokenComma
	// TokenOpenParen is a <(-token>.
	TokenOpenParen
	// TokenCloseParen is a <)-token>.
	TokenCloseParen
	// TokenOpenSquare is a <[-token>.
	TokenOpenSquare
	// TokenCloseSquare is a <]-token>.
	TokenCloseSquare
	// TokenOpenCurly is a <{-token>.
	TokenOpenCurly
	// TokenCloseCurly is a <}-token>.
	TokenCloseCurly
	// TokenUnicodeRange is a <unicode-range-token>. It is only produced when
	// tokenizing with Unicode ranges enabled (the unicode-range descriptor).
	TokenUnicodeRange
)

// NumericKind records whether a <number-token> or <dimension-token> was written
// with an integer or a real-number syntax. The spec calls this the token's
// "type flag".
type NumericKind uint8

const (
	// Integer means the token had no decimal point or exponent.
	Integer NumericKind = iota
	// Number means the token contained a decimal point or exponent.
	Number
)

// HashFlag records whether a <hash-token> is the "id" or "unrestricted" type.
// Only "id" hashes are valid ID selectors. See CSS Syntax 3 §4.
type HashFlag uint8

const (
	// HashUnrestricted is the default hash type.
	HashUnrestricted HashFlag = iota
	// HashID marks a hash whose contents would start an identifier.
	HashID
)

// Token is a single CSS token. Only the fields relevant to Type are
// populated; the others hold their zero value.
//
// Representation preserves the exact subsequence of the preprocessed input
// consumed to produce the token. It is required by algorithms (such as the
// An+B and <urange> microsyntaxes) that depend on details a re-serialization
// might disturb. Start and End are the rune offsets of Representation in the
// preprocessed input.
type Token struct {
	Type TokenType
	// Value is the decoded content of ident, function, at-keyword, hash,
	// string, and url tokens, and the unit of a dimension token.
	Value string
	// Representation is the raw source text consumed for this token.
	Representation string
	// Number is the numeric value of number, percentage, and dimension tokens.
	Number float64
	// NumberRepresentation is the numeric portion of a number, percentage, or
	// dimension token, without the percent sign or unit.
	NumberRepresentation string
	// NumericKind distinguishes integer and real-number syntax.
	NumericKind NumericKind
	// HashFlag distinguishes ID hashes from unrestricted hashes.
	HashFlag HashFlag
	// Start is the rune offset of the token in the preprocessed input.
	Start int
	// End is the exclusive rune offset of the token.
	End int
	// RangeStart and RangeEnd are the code points of a unicode-range token.
	RangeStart int
	RangeEnd   int
}

// String returns the canonical name of the token kind.
func (tokenType TokenType) String() string {
	switch tokenType {
	case TokenEOF:
		return "EOF"
	case TokenIdent:
		return "ident"
	case TokenFunction:
		return "function"
	case TokenAtKeyword:
		return "at-keyword"
	case TokenHash:
		return "hash"
	case TokenString:
		return "string"
	case TokenBadString:
		return "bad-string"
	case TokenURL:
		return "url"
	case TokenBadURL:
		return "bad-url"
	case TokenDelim:
		return "delim"
	case TokenNumber:
		return "number"
	case TokenPercentage:
		return "percentage"
	case TokenDimension:
		return "dimension"
	case TokenWhitespace:
		return "whitespace"
	case TokenCDO:
		return "CDO"
	case TokenCDC:
		return "CDC"
	case TokenColon:
		return "colon"
	case TokenSemicolon:
		return "semicolon"
	case TokenComma:
		return "comma"
	case TokenOpenParen:
		return "open-paren"
	case TokenCloseParen:
		return "close-paren"
	case TokenOpenSquare:
		return "open-square"
	case TokenCloseSquare:
		return "close-square"
	case TokenOpenCurly:
		return "open-curly"
	case TokenCloseCurly:
		return "close-curly"
	case TokenUnicodeRange:
		return "unicode-range"
	default:
		return "token(" + strconv.Itoa(int(tokenType)) + ")"
	}
}
