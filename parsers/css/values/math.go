package values

import (
	"strings"

	"gonetsurf/parsers/css/parser"
	"gonetsurf/parsers/css/tokenizer"
)

// MathKind identifies a node in a math expression tree.
type MathKind uint8

const (
	// MathValue is a literal number, percentage, or dimension.
	MathValue MathKind = iota
	// MathSum is addition or subtraction.
	MathSum
	// MathProduct is multiplication or division.
	MathProduct
	// MathFunction is a min(), max(), or clamp().
	MathFunction
)

// MathExpression is a parsed calc() expression tree.
type MathExpression struct {
	Kind     MathKind
	Value    Value
	Left     *MathExpression
	Right    *MathExpression
	Subtract bool
	Divide   bool
	Name     string
	Args     []*MathExpression
}

// Simplify evaluates the expression to a single Value where possible.
func (expression *MathExpression) Simplify() (Value, bool) {
	switch expression.Kind {
	case MathValue:
		return expression.Value, true
	case MathSum:
		left, ok := expression.Left.Simplify()
		if !ok {
			return Value{}, false
		}
		right, ok := expression.Right.Simplify()
		if !ok {
			return Value{}, false
		}
		return addValues(left, right, expression.Subtract)
	case MathProduct:
		left, ok := expression.Left.Simplify()
		if !ok {
			return Value{}, false
		}
		right, ok := expression.Right.Simplify()
		if !ok {
			return Value{}, false
		}
		return multiplyValues(left, right, expression.Divide)
	case MathFunction:
		return simplifyFunction(expression)
	default:
		return Value{}, false
	}
}

func addValues(left, right Value, subtract bool) (Value, bool) {
	if left.Unit == "" && right.Unit == "" && left.Kind == Other && right.Kind == Other {
		value := left.Number + right.Number
		if subtract {
			value = left.Number - right.Number
		}
		return Value{Kind: Other, Number: value}, true
	}
	if left.Kind == right.Kind && left.Unit == right.Unit {
		value := left.Number + right.Number
		if subtract {
			value = left.Number - right.Number
		}
		return Value{Kind: left.Kind, Number: value, Unit: left.Unit}, true
	}
	if left.Kind == Length && right.Kind == Length {
		leftPixels, leftOK := absoluteToPixels(left)
		rightPixels, rightOK := absoluteToPixels(right)
		if leftOK && rightOK {
			value := leftPixels + rightPixels
			if subtract {
				value = leftPixels - rightPixels
			}
			return Value{Kind: Length, Number: value, Unit: "px"}, true
		}
	}
	return Value{}, false
}

func multiplyValues(left, right Value, divide bool) (Value, bool) {
	if left.Unit == "" && right.Unit == "" {
		value := left.Number * right.Number
		if divide {
			if right.Number == 0 {
				return Value{}, false
			}
			value = left.Number / right.Number
		}
		return Value{Kind: Other, Number: value}, true
	}
	// A dimension may be scaled by a plain number.
	if left.Unit == "" && right.Unit != "" {
		if divide {
			return Value{}, false
		}
		return Value{Kind: right.Kind, Number: left.Number * right.Number, Unit: right.Unit}, true
	}
	if left.Unit != "" && right.Unit == "" {
		value := left.Number * right.Number
		if divide {
			if right.Number == 0 {
				return Value{}, false
			}
			value = left.Number / right.Number
		}
		return Value{Kind: left.Kind, Number: value, Unit: left.Unit}, true
	}
	return Value{}, false
}

func absoluteToPixels(value Value) (float64, bool) {
	switch value.Unit {
	case "px", "cm", "mm", "q", "in", "pc", "pt":
		return ToPixels(value, 0, 0)
	default:
		return 0, false
	}
}

func simplifyFunction(expression *MathExpression) (Value, bool) {
	values := make([]Value, len(expression.Args))
	for index, argument := range expression.Args {
		value, ok := argument.Simplify()
		if !ok {
			return Value{}, false
		}
		values[index] = value
	}
	switch strings.ToLower(expression.Name) {
	case "min":
		result := values[0]
		for _, value := range values[1:] {
			compared, ok := compareValues(result, value)
			if !ok {
				return Value{}, false
			}
			if compared > 0 {
				result = value
			}
		}
		return result, true
	case "max":
		result := values[0]
		for _, value := range values[1:] {
			compared, ok := compareValues(result, value)
			if !ok {
				return Value{}, false
			}
			if compared < 0 {
				result = value
			}
		}
		return result, true
	case "clamp":
		if len(values) != 3 {
			return Value{}, false
		}
		minimum, maximum, preferred := values[0], values[1], values[2]
		value, ok := compareValues(preferred, minimum)
		if !ok {
			return Value{}, false
		}
		if value < 0 {
			return minimum, true
		}
		value, ok = compareValues(preferred, maximum)
		if !ok {
			return Value{}, false
		}
		if value > 0 {
			return maximum, true
		}
		return preferred, true
	default:
		return Value{}, false
	}
}

func compareValues(left, right Value) (int, bool) {
	leftNumber, leftOK := comparableNumber(left)
	rightNumber, rightOK := comparableNumber(right)
	if !leftOK || !rightOK {
		return 0, false
	}
	switch {
	case leftNumber < rightNumber:
		return -1, true
	case leftNumber > rightNumber:
		return 1, true
	default:
		return 0, true
	}
}

func comparableNumber(value Value) (float64, bool) {
	if value.Unit == "" {
		return value.Number, true
	}
	if value.Kind == Length {
		return absoluteToPixels(value)
	}
	return 0, false
}

// ParseMath parses a calc(), min(), max(), or clamp() function.
func ParseMath(function *parser.Function) (*MathExpression, bool) {
	name := strings.ToLower(function.Name)
	switch name {
	case "calc":
		mathParser := &mathState{values: function.Value}
		expression, ok := mathParser.parseSum()
		if !ok {
			return nil, false
		}
		mathParser.skipWhitespace()
		if !parser.IsEOF(mathParser.next()) {
			return nil, false
		}
		return expression, true
	case "min", "max":
		args, ok := parseMathArguments(function.Value)
		if !ok || len(args) == 0 {
			return nil, false
		}
		return &MathExpression{Kind: MathFunction, Name: name, Args: args}, true
	case "clamp":
		args, ok := parseMathArguments(function.Value)
		if !ok || len(args) != 3 {
			return nil, false
		}
		return &MathExpression{Kind: MathFunction, Name: name, Args: args}, true
	default:
		return nil, false
	}
}

func parseMathArguments(values []parser.ComponentValue) ([]*MathExpression, bool) {
	groups := splitCommas(values)
	args := make([]*MathExpression, 0, len(groups))
	for _, group := range groups {
		state := &mathState{values: group}
		expression, ok := state.parseSum()
		if !ok {
			return nil, false
		}
		state.skipWhitespace()
		if !parser.IsEOF(state.next()) {
			return nil, false
		}
		args = append(args, expression)
	}
	return args, true
}

func splitCommas(values []parser.ComponentValue) [][]parser.ComponentValue {
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
	return groups
}

type mathState struct {
	values []parser.ComponentValue
	pos    int
}

func (state *mathState) next() parser.ComponentValue {
	if state.pos < len(state.values) {
		return state.values[state.pos]
	}
	return parser.Token{Token: tokenizer.Token{Type: tokenizer.TokenEOF}}
}

func (state *mathState) skipWhitespace() {
	for parser.IsWhitespace(state.next()) {
		state.pos++
	}
}

func (state *mathState) parseSum() (*MathExpression, bool) {
	left, ok := state.parseProduct()
	if !ok {
		return nil, false
	}
	for {
		state.skipWhitespace()
		token, ok := parser.AsToken(state.next())
		if !ok || token.Type != tokenizer.TokenDelim || (token.Value != "+" && token.Value != "-") {
			return left, true
		}
		state.pos++
		right, ok := state.parseProduct()
		if !ok {
			return nil, false
		}
		left = &MathExpression{Kind: MathSum, Left: left, Right: right, Subtract: token.Value == "-"}
	}
}

func (state *mathState) parseProduct() (*MathExpression, bool) {
	left, ok := state.parseFactor()
	if !ok {
		return nil, false
	}
	for {
		state.skipWhitespace()
		token, ok := parser.AsToken(state.next())
		if !ok || token.Type != tokenizer.TokenDelim || (token.Value != "*" && token.Value != "/") {
			return left, true
		}
		state.pos++
		right, ok := state.parseFactor()
		if !ok {
			return nil, false
		}
		left = &MathExpression{Kind: MathProduct, Left: left, Right: right, Divide: token.Value == "/"}
	}
}

func (state *mathState) parseFactor() (*MathExpression, bool) {
	state.skipWhitespace()
	value := state.next()
	if _, ok := parser.AsToken(value); ok {
		if numeric, valid := ParseNumeric(value); valid {
			state.pos++
			return &MathExpression{Kind: MathValue, Value: numeric}, true
		}
	}
	if block, ok := value.(*parser.Block); ok && block.Associated == tokenizer.TokenOpenParen {
		state.pos++
		inner := &mathState{values: block.Value}
		expression, ok := inner.parseSum()
		if !ok {
			return nil, false
		}
		inner.skipWhitespace()
		if !parser.IsEOF(inner.next()) {
			return nil, false
		}
		return expression, true
	}
	if function, ok := value.(*parser.Function); ok {
		state.pos++
		return ParseMath(function)
	}
	return nil, false
}
