// Package grammar implements the CSS value-definition syntax: parsing property
// grammar strings into an expression tree and matching component values
// against it.
//
// See specifications/css/values-and-units-4.html §2.
package grammar

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Expression is a node in a value-definition grammar.
type Expression interface {
	expression()
}

// Reference is a type reference such as <length> or a property reference such
// as <'color'>.
type Reference struct {
	Name     string
	Property bool
	Range    *NumericRange
}

func (*Reference) expression() {}

// NumericRange is an optional numeric range attached to a type reference,
// such as <length [0,∞]> or <angle [-90deg,90deg]>.
type NumericRange struct {
	Min          float64
	Max          float64
	HasMin       bool
	HasMax       bool
	MinInclusive bool
	MaxInclusive bool
	Unit         string
}

// Keyword is a literal identifier, such as auto or none.
type Keyword struct {
	Name string
}

func (*Keyword) expression() {}

// Literal is a literal delimiter such as "," or "/".
type Literal struct {
	Value string
}

func (*Literal) expression() {}

// Sequence is a juxtaposition: every item must match in order.
type Sequence struct {
	Items []Expression
}

func (*Sequence) expression() {}

// OneOf is a single-bar alternation: exactly one item must match.
type OneOf struct {
	Items []Expression
}

func (*OneOf) expression() {}

// AnyOrder is a double-bar combinator: one or more items match in any order.
type AnyOrder struct {
	Items []Expression
}

func (*AnyOrder) expression() {}

// AllAnyOrder is a double-ampersand combinator: every item matches in any
// order.
type AllAnyOrder struct {
	Items []Expression
}

func (*AllAnyOrder) expression() {}

// Repeat is a multiplier. Max is -1 for an unbounded count. Separated marks a
// comma-separated (#) list.
type Repeat struct {
	Item      Expression
	Min       int
	Max       int
	Separated bool
}

func (*Repeat) expression() {}

// Required marks a required group ([ ... ]!).
type Required struct {
	Item Expression
}

func (*Required) expression() {}

// Parse parses a value-definition grammar string into an Expression.
func Parse(definition string) (Expression, error) {
	parser := &definitionParser{input: []rune(definition)}
	parser.skipSpace()
	expression, err := parser.parseOneOf()
	if err != nil {
		return nil, err
	}
	parser.skipSpace()
	if parser.position < len(parser.input) {
		return nil, fmt.Errorf("css grammar: unexpected %q at %d", string(parser.input[parser.position]), parser.position)
	}
	return expression, nil
}

type definitionParser struct {
	input    []rune
	position int
}

func (parser *definitionParser) skipSpace() {
	for parser.position < len(parser.input) && isSpace(parser.input[parser.position]) {
		parser.position++
	}
}

func (parser *definitionParser) peek() rune {
	if parser.position < len(parser.input) {
		return parser.input[parser.position]
	}
	return 0
}

func (parser *definitionParser) parseOneOf() (Expression, error) {
	first, err := parser.parseAnyOrder()
	if err != nil {
		return nil, err
	}
	items := []Expression{first}
	for {
		parser.skipSpace()
		if parser.peek() != '|' || parser.peekAt(1) == '|' {
			break
		}
		parser.position++
		next, err := parser.parseAnyOrder()
		if err != nil {
			return nil, err
		}
		items = append(items, next)
	}
	if len(items) == 1 {
		return items[0], nil
	}
	return &OneOf{Items: items}, nil
}

func (parser *definitionParser) parseAnyOrder() (Expression, error) {
	first, err := parser.parseAllAnyOrder()
	if err != nil {
		return nil, err
	}
	items := []Expression{first}
	for {
		parser.skipSpace()
		if parser.peek() != '|' || parser.peekAt(1) != '|' {
			break
		}
		parser.position += 2
		next, err := parser.parseAllAnyOrder()
		if err != nil {
			return nil, err
		}
		items = append(items, next)
	}
	if len(items) == 1 {
		return items[0], nil
	}
	return &AnyOrder{Items: items}, nil
}

func (parser *definitionParser) parseAllAnyOrder() (Expression, error) {
	first, err := parser.parseSequence()
	if err != nil {
		return nil, err
	}
	items := []Expression{first}
	for {
		parser.skipSpace()
		if parser.peek() != '&' || parser.peekAt(1) != '&' {
			break
		}
		parser.position += 2
		next, err := parser.parseSequence()
		if err != nil {
			return nil, err
		}
		items = append(items, next)
	}
	if len(items) == 1 {
		return items[0], nil
	}
	return &AllAnyOrder{Items: items}, nil
}

func (parser *definitionParser) parseSequence() (Expression, error) {
	var items []Expression
	for {
		parser.skipSpace()
		current := parser.peek()
		if current == 0 || current == '|' || current == ']' || current == ')' {
			break
		}
		if current == '&' && parser.peekAt(1) == '&' {
			break
		}
		item, err := parser.parseAtom()
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("css grammar: expected a term at %d", parser.position)
	}
	if len(items) == 1 {
		return items[0], nil
	}
	return &Sequence{Items: items}, nil
}

func (parser *definitionParser) parseAtom() (Expression, error) {
	parser.skipSpace()
	current := parser.peek()
	var item Expression
	switch {
	case current == '<':
		reference, err := parser.parseReference()
		if err != nil {
			return nil, err
		}
		item = reference
	case current == '[':
		parser.position++
		inner, err := parser.parseOneOf()
		if err != nil {
			return nil, err
		}
		parser.skipSpace()
		if parser.peek() != ']' {
			return nil, fmt.Errorf("css grammar: missing ] at %d", parser.position)
		}
		parser.position++
		item = inner
	case current == '(':
		parser.position++
		inner, err := parser.parseOneOf()
		if err != nil {
			return nil, err
		}
		parser.skipSpace()
		if parser.peek() != ')' {
			return nil, fmt.Errorf("css grammar: missing ) at %d", parser.position)
		}
		parser.position++
		item = inner
	case isNameStart(current):
		item = &Keyword{Name: parser.parseName()}
	default:
		parser.position++
		item = &Literal{Value: string(current)}
	}
	return parser.parseMultiplier(item)
}

func (parser *definitionParser) parseReference() (Expression, error) {
	parser.position++
	start := parser.position
	for parser.position < len(parser.input) && parser.input[parser.position] != '>' {
		parser.position++
	}
	if parser.peek() != '>' {
		return nil, fmt.Errorf("css grammar: missing > at %d", parser.position)
	}
	raw := strings.TrimSpace(string(parser.input[start:parser.position]))
	parser.position++
	property := false
	if strings.HasPrefix(raw, "'") && strings.HasSuffix(raw, "'") && len(raw) >= 2 {
		property = true
		raw = strings.Trim(raw, "'")
	}
	if property {
		return &Reference{Name: strings.TrimSpace(raw), Property: true}, nil
	}
	name, numericRange, err := splitReferenceRange(raw)
	if err != nil {
		return nil, err
	}
	return &Reference{Name: name, Range: numericRange}, nil
}

func splitReferenceRange(raw string) (string, *NumericRange, error) {
	index := -1
	for position, current := range raw {
		if current == '[' || current == '(' {
			index = position
			break
		}
	}
	if index < 0 {
		return raw, nil, nil
	}
	name := strings.TrimSpace(raw[:index])
	numericRange, err := parseNumericRange(strings.TrimSpace(raw[index:]))
	if err != nil {
		return "", nil, err
	}
	return name, numericRange, nil
}

func parseNumericRange(text string) (*NumericRange, error) {
	if len(text) < 2 {
		return nil, fmt.Errorf("css grammar: invalid range %q", text)
	}
	open := text[0]
	close := text[len(text)-1]
	if (open != '[' && open != '(') || (close != ']' && close != ')') {
		return nil, fmt.Errorf("css grammar: invalid range %q", text)
	}
	body := text[1 : len(text)-1]
	comma := strings.Index(body, ",")
	if comma < 0 {
		return nil, fmt.Errorf("css grammar: invalid range %q", text)
	}
	numericRange := &NumericRange{
		MinInclusive: open == '[',
		MaxInclusive: close == ']',
	}
	minimum, ok, err := parseRangeBound(strings.TrimSpace(body[:comma]))
	if err != nil {
		return nil, err
	}
	if ok {
		numericRange.Min = minimum.number
		numericRange.HasMin = true
		numericRange.Unit = minimum.unit
	}
	maximum, ok, err := parseRangeBound(strings.TrimSpace(body[comma+1:]))
	if err != nil {
		return nil, err
	}
	if ok {
		numericRange.Max = maximum.number
		numericRange.HasMax = true
		if numericRange.Unit == "" {
			numericRange.Unit = maximum.unit
		}
	}
	return numericRange, nil
}

type rangeBound struct {
	number float64
	unit   string
}

func parseRangeBound(text string) (rangeBound, bool, error) {
	switch text {
	case "∞", "+∞":
		return rangeBound{}, false, nil
	case "-∞":
		return rangeBound{}, false, nil
	}
	index := 0
	if index < len(text) && (text[index] == '+' || text[index] == '-') {
		index++
	}
	for index < len(text) && (unicode.IsDigit(rune(text[index])) || text[index] == '.') {
		index++
	}
	if index == 0 {
		return rangeBound{}, false, fmt.Errorf("css grammar: invalid range bound %q", text)
	}
	number, err := strconv.ParseFloat(text[:index], 64)
	if err != nil {
		return rangeBound{}, false, fmt.Errorf("css grammar: invalid range bound %q", text)
	}
	return rangeBound{number: number, unit: strings.TrimSpace(text[index:])}, true, nil
}

func (parser *definitionParser) parseMultiplier(item Expression) (Expression, error) {
	parser.skipSpace()
	switch parser.peek() {
	case '?':
		parser.position++
		return &Repeat{Item: item, Min: 0, Max: 1}, nil
	case '*':
		parser.position++
		return &Repeat{Item: item, Min: 0, Max: -1}, nil
	case '+':
		parser.position++
		return &Repeat{Item: item, Min: 1, Max: -1}, nil
	case '#':
		parser.position++
		repeat := &Repeat{Item: item, Min: 1, Max: -1, Separated: true}
		return parser.maybeRange(repeat)
	case '{':
		return parser.maybeRange(&Repeat{Item: item})
	case '!':
		parser.position++
		return &Required{Item: item}, nil
	default:
		return item, nil
	}
}

func (parser *definitionParser) maybeRange(repeat *Repeat) (Expression, error) {
	parser.skipSpace()
	if parser.peek() != '{' {
		return repeat, nil
	}
	parser.position++
	parser.skipSpace()
	min, err := parser.parseInteger()
	if err != nil {
		return nil, err
	}
	repeat.Min = min
	repeat.Max = min
	parser.skipSpace()
	if parser.peek() == ',' {
		parser.position++
		parser.skipSpace()
		if parser.peek() == '}' {
			repeat.Max = -1
		} else {
			max, err := parser.parseInteger()
			if err != nil {
				return nil, err
			}
			repeat.Max = max
		}
	}
	parser.skipSpace()
	if parser.peek() != '}' {
		return nil, fmt.Errorf("css grammar: missing } at %d", parser.position)
	}
	if repeat.Max >= 0 && repeat.Max < repeat.Min {
		return nil, fmt.Errorf("css grammar: invalid range {%d,%d}", repeat.Min, repeat.Max)
	}
	parser.position++
	return repeat, nil
}

func (parser *definitionParser) parseInteger() (int, error) {
	start := parser.position
	for parser.position < len(parser.input) && parser.input[parser.position] >= '0' && parser.input[parser.position] <= '9' {
		parser.position++
	}
	if parser.position == start {
		return 0, fmt.Errorf("css grammar: expected a number at %d", parser.position)
	}
	value := 0
	for _, digit := range parser.input[start:parser.position] {
		value = value*10 + int(digit-'0')
	}
	return value, nil
}

func (parser *definitionParser) parseName() string {
	var name strings.Builder
	for parser.position < len(parser.input) {
		current := parser.input[parser.position]
		if !isNameRune(current) {
			break
		}
		name.WriteRune(current)
		parser.position++
	}
	return name.String()
}

func (parser *definitionParser) peekAt(offset int) rune {
	index := parser.position + offset
	if index < len(parser.input) {
		return parser.input[index]
	}
	return 0
}

func isSpace(current rune) bool {
	return current == ' ' || current == '\t' || current == '\n' || current == '\r' || current == '\f'
}

func isNameStart(current rune) bool {
	return (current >= 'a' && current <= 'z') || (current >= 'A' && current <= 'Z') || current == '_' || current == '-'
}

func isNameRune(current rune) bool {
	return isNameStart(current) || (current >= '0' && current <= '9')
}
