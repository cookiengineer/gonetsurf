package parser

import (
	"errors"
	"strings"

	"gonetsurf/parsers/css/tokenizer"
)

// Sentinel errors reported by the parser entry points.
var (
	// ErrEmpty reports that the input contained no rule, declaration, or
	// component value.
	ErrEmpty = errors.New("css: empty input")
	// ErrInvalid reports that the input was not a valid rule or declaration.
	ErrInvalid = errors.New("css: invalid input")
	// ErrExtraInput reports that input remained after a complete construct.
	ErrExtraInput = errors.New("css: extra input")
)

// Node is a parsed CSS construct: a qualified rule, an at-rule, or a
// declaration.
type Node interface {
	node()
}

// QualifiedRule is a style rule: a prelude followed by a {} block.
type QualifiedRule struct {
	Prelude []ComponentValue
	Block   *Block
}

func (*QualifiedRule) node() {}

// AtRule is an at-rule. Block is nil for statement at-rules.
type AtRule struct {
	Name    string
	Prelude []ComponentValue
	Block   *Block
}

func (*AtRule) node() {}

// Declaration is a property declaration. Important records whether it ended
// with "!important".
type Declaration struct {
	Name      string
	Value     []ComponentValue
	Important bool
}

func (*Declaration) node() {}

// ParseComponentValueList parses input into a list of component values. It
// never fails. See CSS Syntax 3 §5.3.9.
func ParseComponentValueList(input string) []ComponentValue {
	return buildComponentValues(tokenizer.Tokenize(input))
}

// ParseComponentValue parses input into a single component value, returning
// ErrEmpty for whitespace-only input and ErrExtraInput when more than one
// component value is present. See CSS Syntax 3 §5.3.8.
func ParseComponentValue(input string) (ComponentValue, error) {
	reader := newValueReader(ParseComponentValueList(input))
	reader.skipWhitespace()
	if IsEOF(reader.next()) {
		return nil, ErrEmpty
	}
	value := reader.consume()
	reader.skipWhitespace()
	if !IsEOF(reader.next()) {
		return nil, ErrExtraInput
	}
	return value, nil
}

// ParseCommaSeparatedComponentValueList parses input into a list of
// comma-separated component value lists. See CSS Syntax 3 §5.3.10.
func ParseCommaSeparatedComponentValueList(input string) [][]ComponentValue {
	reader := newValueReader(ParseComponentValueList(input))
	lists := make([][]ComponentValue, 0, 1)
	current := []ComponentValue{}
	for {
		value := reader.consume()
		if IsEOF(value) {
			lists = append(lists, current)
			return lists
		}
		if token, ok := AsToken(value); ok && token.Type == tokenizer.TokenComma {
			lists = append(lists, current)
			current = []ComponentValue{}
			continue
		}
		current = append(current, value)
	}
}

// ParseStylesheet parses input into a list of rules, ignoring CDO and CDC
// tokens between rules. See CSS Syntax 3 §5.3.3.
func ParseStylesheet(input string) []Node {
	return consumeListOfRules(newValueReader(ParseComponentValueList(input)), true)
}

// ParseListOfRules parses input into a list of rules without treating CDO and
// CDC specially. See CSS Syntax 3 §5.3.4.
func ParseListOfRules(input string) []Node {
	return consumeListOfRules(newValueReader(ParseComponentValueList(input)), false)
}

// ParseRule parses input into a single rule. See CSS Syntax 3 §5.3.5.
func ParseRule(input string) (Node, error) {
	reader := newValueReader(ParseComponentValueList(input))
	reader.skipWhitespace()
	if IsEOF(reader.next()) {
		return nil, ErrEmpty
	}
	var rule Node
	if token, ok := AsToken(reader.next()); ok && token.Type == tokenizer.TokenAtKeyword {
		rule = consumeAtRule(reader)
	} else {
		qualified := consumeQualifiedRule(reader)
		if qualified == nil {
			return nil, ErrInvalid
		}
		rule = qualified
	}
	reader.skipWhitespace()
	if !IsEOF(reader.next()) {
		return nil, ErrExtraInput
	}
	return rule, nil
}

// ParseDeclaration parses input into a single declaration. See CSS Syntax 3
// §5.3.6.
func ParseDeclaration(input string) (*Declaration, error) {
	reader := newValueReader(ParseComponentValueList(input))
	reader.skipWhitespace()
	if IsEOF(reader.next()) {
		return nil, ErrEmpty
	}
	token, ok := AsToken(reader.next())
	if !ok || token.Type != tokenizer.TokenIdent {
		return nil, ErrInvalid
	}
	declaration := consumeDeclaration(reader)
	if declaration == nil {
		return nil, ErrInvalid
	}
	return declaration, nil
}

// ParseListOfDeclarations parses input into a mixed list of declarations and
// at-rules. See CSS Syntax 3 §5.3.7.
func ParseListOfDeclarations(input string) []Node {
	return ParseListOfDeclarationsValues(ParseComponentValueList(input))
}

// ParseListOfDeclarationsValues parses declarations and at-rules from
// component values.
func ParseListOfDeclarationsValues(values []ComponentValue) []Node {
	return consumeListOfDeclarations(newValueReader(values))
}

// ParseBlockContents parses the contents of a {} block: declarations, at-rules,
// and qualified rules (CSS Nesting). See CSS Syntax 3 §5.5.5.
func ParseBlockContents(input string) []Node {
	return ParseBlockContentsValues(ParseComponentValueList(input))
}

// ParseBlockContentsValues parses block contents from component values.
func ParseBlockContentsValues(values []ComponentValue) []Node {
	return consumeBlockContents(newValueReader(values))
}

func consumeBlockContents(reader *valueReader) []Node {
	nodes := []Node{}
	for {
		reader.skipWhitespace()
		value := reader.next()
		if IsEOF(value) {
			return nodes
		}
		if token, ok := AsToken(value); ok && token.Type == tokenizer.TokenSemicolon {
			reader.consume()
			continue
		}
		if token, ok := AsToken(value); ok && token.Type == tokenizer.TokenAtKeyword {
			nodes = append(nodes, consumeAtRule(reader))
			continue
		}
		scan := reader.index
		blockIndex := -1
		for scan < len(reader.values) {
			if token, ok := AsToken(reader.values[scan]); ok && token.Type == tokenizer.TokenSemicolon {
				break
			}
			if block, ok := reader.values[scan].(*Block); ok && block.Associated == tokenizer.TokenOpenCurly {
				blockIndex = scan
				break
			}
			scan++
		}
		if blockIndex >= 0 {
			if rule := consumeQualifiedRule(reader); rule != nil {
				nodes = append(nodes, rule)
			}
			continue
		}
		segment := reader.values[reader.index:scan]
		declarationReader := newValueReader(segment)
		if declaration := consumeDeclaration(declarationReader); declaration != nil {
			nodes = append(nodes, declaration)
		}
		reader.index = scan
		if scan < len(reader.values) {
			reader.index = scan + 1
		}
	}
}

// SerializeValues reconstructs CSS text from component values. Tokens use
// their preserved Representation, so the result reproduces the source modulo
// comments.
func SerializeValues(values []ComponentValue) string {
	var builder strings.Builder
	for _, value := range values {
		serializeValue(&builder, value)
	}
	return builder.String()
}

func serializeValue(builder *strings.Builder, value ComponentValue) {
	switch typed := value.(type) {
	case Token:
		builder.WriteString(typed.Representation)
	case *Function:
		builder.WriteString(typed.Name)
		builder.WriteByte('(')
		for _, argument := range typed.Value {
			serializeValue(builder, argument)
		}
		builder.WriteByte(')')
	case *Block:
		switch typed.Associated {
		case tokenizer.TokenOpenCurly:
			builder.WriteByte('{')
		case tokenizer.TokenOpenSquare:
			builder.WriteByte('[')
		case tokenizer.TokenOpenParen:
			builder.WriteByte('(')
		}
		for _, item := range typed.Value {
			serializeValue(builder, item)
		}
		switch typed.Associated {
		case tokenizer.TokenOpenCurly:
			builder.WriteByte('}')
		case tokenizer.TokenOpenSquare:
			builder.WriteByte(']')
		case tokenizer.TokenOpenParen:
			builder.WriteByte(')')
		}
	}
}

func buildComponentValues(tokens []tokenizer.Token) []ComponentValue {
	reader := &tokenReader{tokens: tokens}
	var values []ComponentValue
	for {
		token := reader.consume()
		if token.Type == tokenizer.TokenEOF {
			return values
		}
		values = append(values, buildFromToken(reader, token))
	}
}

func buildFromToken(reader *tokenReader, token tokenizer.Token) ComponentValue {
	switch token.Type {
	case tokenizer.TokenOpenCurly, tokenizer.TokenOpenSquare, tokenizer.TokenOpenParen:
		return consumeSimpleBlock(reader, token)
	case tokenizer.TokenFunction:
		return consumeFunction(reader, token)
	default:
		return Token{token}
	}
}

func consumeSimpleBlock(reader *tokenReader, open tokenizer.Token) *Block {
	ending := mirror(open.Type)
	block := &Block{Associated: open.Type}
	for {
		token := reader.consume()
		if token.Type == ending || token.Type == tokenizer.TokenEOF {
			return block
		}
		block.Value = append(block.Value, buildFromToken(reader, token))
	}
}

func consumeFunction(reader *tokenReader, open tokenizer.Token) *Function {
	function := &Function{Name: open.Value}
	for {
		token := reader.consume()
		if token.Type == tokenizer.TokenCloseParen || token.Type == tokenizer.TokenEOF {
			return function
		}
		function.Value = append(function.Value, buildFromToken(reader, token))
	}
}

func consumeListOfRules(reader *valueReader, topLevel bool) []Node {
	rules := []Node{}
	for {
		value := reader.consume()
		if IsEOF(value) {
			return rules
		}
		if IsWhitespace(value) {
			continue
		}
		if token, ok := AsToken(value); ok {
			if token.Type == tokenizer.TokenCDO || token.Type == tokenizer.TokenCDC {
				if topLevel {
					continue
				}
				reader.requestReconsume()
				if rule := consumeQualifiedRule(reader); rule != nil {
					rules = append(rules, rule)
				}
				continue
			}
			if token.Type == tokenizer.TokenAtKeyword {
				reader.requestReconsume()
				rules = append(rules, consumeAtRule(reader))
				continue
			}
		}
		reader.requestReconsume()
		if rule := consumeQualifiedRule(reader); rule != nil {
			rules = append(rules, rule)
		}
	}
}

func consumeAtRule(reader *valueReader) *AtRule {
	token, _ := AsToken(reader.consume())
	atRule := &AtRule{Name: token.Value, Prelude: []ComponentValue{}}
	for {
		value := reader.consume()
		if IsEOF(value) {
			return atRule
		}
		if token, ok := AsToken(value); ok && token.Type == tokenizer.TokenSemicolon {
			return atRule
		}
		if block, ok := IsBlock(value, tokenizer.TokenOpenCurly); ok {
			atRule.Block = block
			return atRule
		}
		reader.requestReconsume()
		atRule.Prelude = append(atRule.Prelude, reader.consume())
	}
}

func consumeQualifiedRule(reader *valueReader) *QualifiedRule {
	rule := &QualifiedRule{Prelude: []ComponentValue{}}
	for {
		value := reader.consume()
		if IsEOF(value) {
			return nil
		}
		if block, ok := IsBlock(value, tokenizer.TokenOpenCurly); ok {
			rule.Block = block
			return rule
		}
		reader.requestReconsume()
		rule.Prelude = append(rule.Prelude, reader.consume())
	}
}

func consumeListOfDeclarations(reader *valueReader) []Node {
	declarations := []Node{}
	for {
		value := reader.consume()
		if IsEOF(value) {
			return declarations
		}
		if IsWhitespace(value) {
			continue
		}
		token, ok := AsToken(value)
		if !ok {
			reader.requestReconsume()
			reader.discardUntilSemicolon()
			continue
		}
		switch token.Type {
		case tokenizer.TokenSemicolon:
			continue
		case tokenizer.TokenAtKeyword:
			reader.requestReconsume()
			declarations = append(declarations, consumeAtRule(reader))
		case tokenizer.TokenIdent:
			temporary := []ComponentValue{value}
			for {
				next := reader.next()
				if IsEOF(next) {
					break
				}
				if nextToken, ok := AsToken(next); ok && nextToken.Type == tokenizer.TokenSemicolon {
					break
				}
				temporary = append(temporary, reader.consume())
			}
			declarationReader := newValueReader(temporary)
			if declaration := consumeDeclaration(declarationReader); declaration != nil {
				declarations = append(declarations, declaration)
			}
		default:
			reader.requestReconsume()
			reader.discardUntilSemicolon()
		}
	}
}

func consumeDeclaration(reader *valueReader) *Declaration {
	token, ok := AsToken(reader.consume())
	if !ok || token.Type != tokenizer.TokenIdent {
		return nil
	}
	declaration := &Declaration{Name: token.Value, Value: []ComponentValue{}}
	reader.skipWhitespace()
	if nextToken, ok := AsToken(reader.next()); !ok || nextToken.Type != tokenizer.TokenColon {
		return nil
	}
	reader.consume()
	reader.skipWhitespace()
	for !IsEOF(reader.next()) {
		declaration.Value = append(declaration.Value, reader.consume())
	}
	value, important := extractImportant(declaration.Value)
	declaration.Value = value
	declaration.Important = important
	return declaration
}

func extractImportant(value []ComponentValue) ([]ComponentValue, bool) {
	last := -1
	secondLast := -1
	for index := len(value) - 1; index >= 0; index-- {
		if IsWhitespace(value[index]) {
			continue
		}
		if last == -1 {
			last = index
			continue
		}
		secondLast = index
		break
	}
	if last == -1 || secondLast == -1 {
		return value, false
	}
	if !isDelim(value[secondLast], '!') || !isIdentFold(value[last], "important") {
		return value, false
	}
	result := make([]ComponentValue, 0, len(value))
	for index, item := range value {
		if index == secondLast || index == last {
			continue
		}
		result = append(result, item)
	}
	for len(result) > 0 && IsWhitespace(result[len(result)-1]) {
		result = result[:len(result)-1]
	}
	return result, true
}

func isDelim(value ComponentValue, delim rune) bool {
	token, ok := AsToken(value)
	return ok && token.Type == tokenizer.TokenDelim && token.Value == string(delim)
}

func isIdentFold(value ComponentValue, name string) bool {
	token, ok := AsToken(value)
	if !ok || token.Type != tokenizer.TokenIdent {
		return false
	}
	return asciiEqualFold(token.Value, name)
}

func asciiEqualFold(left, right string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := 0; index < len(left); index++ {
		leftByte := left[index]
		rightByte := right[index]
		if leftByte == rightByte {
			continue
		}
		if leftByte >= 'A' && leftByte <= 'Z' {
			leftByte += 'a' - 'A'
		}
		if rightByte >= 'A' && rightByte <= 'Z' {
			rightByte += 'a' - 'A'
		}
		if leftByte != rightByte {
			return false
		}
	}
	return true
}

// tokenReader consumes tokens while building component values.
type tokenReader struct {
	tokens []tokenizer.Token
	index  int
}

func (reader *tokenReader) consume() tokenizer.Token {
	if reader.index < len(reader.tokens) {
		token := reader.tokens[reader.index]
		reader.index++
		return token
	}
	return tokenizer.Token{Type: tokenizer.TokenEOF}
}

// valueReader consumes a prebuilt list of component values.
type valueReader struct {
	values    []ComponentValue
	index     int
	reconsume bool
	current   ComponentValue
}

func newValueReader(values []ComponentValue) *valueReader {
	return &valueReader{values: values}
}

func (reader *valueReader) next() ComponentValue {
	if reader.reconsume {
		return reader.current
	}
	if reader.index < len(reader.values) {
		return reader.values[reader.index]
	}
	return eofValue()
}

func (reader *valueReader) consume() ComponentValue {
	if reader.reconsume {
		reader.reconsume = false
		return reader.current
	}
	if reader.index < len(reader.values) {
		reader.current = reader.values[reader.index]
		reader.index++
	} else {
		reader.current = eofValue()
	}
	return reader.current
}

func (reader *valueReader) requestReconsume() {
	reader.reconsume = true
}

func (reader *valueReader) skipWhitespace() {
	for IsWhitespace(reader.next()) {
		reader.consume()
	}
}

func (reader *valueReader) discardUntilSemicolon() {
	for {
		value := reader.consume()
		if IsEOF(value) {
			return
		}
		if token, ok := AsToken(value); ok && token.Type == tokenizer.TokenSemicolon {
			return
		}
	}
}
