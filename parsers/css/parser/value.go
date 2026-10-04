package parser

import "gonetsurf/parsers/css/tokenizer"

// ComponentValue is a CSS component value: a token, a function, or a simple
// block. See CSS Syntax 3 §5.2.
type ComponentValue interface {
	componentValue()
}

// Token is a component value that consists of a single token.
type Token struct{ tokenizer.Token }

func (Token) componentValue() {}

// Function is a <function> component value. Value holds its arguments.
type Function struct {
	Name  string
	Value []ComponentValue
}

func (*Function) componentValue() {}

// Block is a simple block component value. Associated is the opening token
// type ({, [, or (); Value holds the block's contents.
type Block struct {
	Associated tokenizer.TokenType
	Value      []ComponentValue
}

func (*Block) componentValue() {}

// AsToken returns the underlying token if the component value is a Token.
func AsToken(value ComponentValue) (tokenizer.Token, bool) {
	token, ok := value.(Token)
	return token.Token, ok
}

// IsEOF reports whether the component value is an EOF token.
func IsEOF(value ComponentValue) bool {
	token, ok := AsToken(value)
	return ok && token.Type == tokenizer.TokenEOF
}

// IsWhitespace reports whether the component value is a whitespace token.
func IsWhitespace(value ComponentValue) bool {
	token, ok := AsToken(value)
	return ok && token.Type == tokenizer.TokenWhitespace
}

// EOFValue is the sentinel returned when a component value list is exhausted.
func eofValue() ComponentValue {
	return Token{tokenizer.Token{Type: tokenizer.TokenEOF}}
}

func mirror(open tokenizer.TokenType) tokenizer.TokenType {
	switch open {
	case tokenizer.TokenOpenCurly:
		return tokenizer.TokenCloseCurly
	case tokenizer.TokenOpenSquare:
		return tokenizer.TokenCloseSquare
	case tokenizer.TokenOpenParen:
		return tokenizer.TokenCloseParen
	default:
		return tokenizer.TokenEOF
	}
}

// IsBlock reports whether the value is a simple block opened by the given
// token type.
func IsBlock(value ComponentValue, open tokenizer.TokenType) (*Block, bool) {
	block, ok := value.(*Block)
	if !ok || block.Associated != open {
		return nil, false
	}
	return block, true
}
