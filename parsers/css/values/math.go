package values

import (
	"math"
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
	Strategy string
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
		return pickExtreme(values, false)
	case "max":
		return pickExtreme(values, true)
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
	case "round":
		if len(values) != 2 {
			return Value{}, false
		}
		number, step, result, ok := compatibleNumbers(values[0], values[1])
		if !ok || step == 0 {
			return Value{}, false
		}
		result.Number = roundNumber(number, step, expression.Strategy)
		return result, true
	case "mod":
		if len(values) != 2 {
			return Value{}, false
		}
		left, right, result, ok := compatibleNumbers(values[0], values[1])
		if !ok || right == 0 {
			return Value{}, false
		}
		result.Number = left - right*math.Floor(left/right)
		return result, true
	case "rem":
		if len(values) != 2 {
			return Value{}, false
		}
		left, right, result, ok := compatibleNumbers(values[0], values[1])
		if !ok || right == 0 {
			return Value{}, false
		}
		result.Number = math.Mod(left, right)
		return result, true
	case "abs":
		if len(values) != 1 {
			return Value{}, false
		}
		value := values[0]
		value.Number = math.Abs(value.Number)
		return value, true
	case "sign":
		if len(values) != 1 {
			return Value{}, false
		}
		if values[0].Kind == Other && values[0].Unit != "" {
			return Value{}, false
		}
		return Value{Kind: Other, Number: compareSign(values[0].Number)}, true
	case "hypot":
		if len(values) == 0 {
			return Value{}, false
		}
		return hypotValues(values)
	case "pow":
		if len(values) != 2 {
			return Value{}, false
		}
		base, baseOK := numericValue(values[0])
		exponent, exponentOK := numericValue(values[1])
		if !baseOK || !exponentOK {
			return Value{}, false
		}
		return Value{Kind: Other, Number: math.Pow(base, exponent)}, true
	case "sqrt":
		if len(values) != 1 {
			return Value{}, false
		}
		number, ok := numericValue(values[0])
		if !ok || number < 0 {
			return Value{}, false
		}
		return Value{Kind: Other, Number: math.Sqrt(number)}, true
	case "exp":
		if len(values) != 1 {
			return Value{}, false
		}
		number, ok := numericValue(values[0])
		if !ok {
			return Value{}, false
		}
		return Value{Kind: Other, Number: math.Exp(number)}, true
	case "log":
		return simplifyLog(values)
	case "sin", "cos", "tan":
		if len(values) != 1 {
			return Value{}, false
		}
		radians, ok := radiansValue(values[0])
		if !ok {
			return Value{}, false
		}
		var result float64
		switch strings.ToLower(expression.Name) {
		case "sin":
			result = math.Sin(radians)
		case "cos":
			result = math.Cos(radians)
		default:
			result = math.Tan(radians)
		}
		return Value{Kind: Other, Number: result}, true
	case "asin", "acos", "atan":
		if len(values) != 1 {
			return Value{}, false
		}
		number, ok := numericValue(values[0])
		if !ok {
			return Value{}, false
		}
		var radians float64
		switch strings.ToLower(expression.Name) {
		case "asin":
			radians = math.Asin(number)
		case "acos":
			radians = math.Acos(number)
		default:
			radians = math.Atan(number)
		}
		return angleValue(radians), true
	case "atan2":
		if len(values) != 2 {
			return Value{}, false
		}
		y, yOK := numericValue(values[0])
		x, xOK := numericValue(values[1])
		if !yOK || !xOK {
			return Value{}, false
		}
		return angleValue(math.Atan2(y, x)), true
	default:
		return Value{}, false
	}
}

func pickExtreme(values []Value, maximum bool) (Value, bool) {
	if len(values) == 0 {
		return Value{}, false
	}
	result := values[0]
	for _, value := range values[1:] {
		compared, ok := compareValues(result, value)
		if !ok {
			return Value{}, false
		}
		if (maximum && compared < 0) || (!maximum && compared > 0) {
			result = value
		}
	}
	return result, true
}

func numericValue(value Value) (float64, bool) {
	if value.Kind == Other && value.Unit == "" {
		return value.Number, true
	}
	return 0, false
}

func radiansValue(value Value) (float64, bool) {
	if value.Kind == Angle {
		degrees, ok := ToDegrees(value)
		if !ok {
			return 0, false
		}
		return degrees * math.Pi / 180, true
	}
	return numericValue(value)
}

func angleValue(radians float64) Value {
	return Value{Kind: Angle, Unit: "deg", Number: radians * 180 / math.Pi}
}

func compareSign(number float64) float64 {
	switch {
	case number < 0:
		return -1
	case number > 0:
		return 1
	default:
		return 0
	}
}

func roundNumber(number, step float64, strategy string) float64 {
	quotient := number / step
	switch strings.ToLower(strategy) {
	case "up":
		return math.Ceil(quotient) * step
	case "down":
		return math.Floor(quotient) * step
	case "to-zero":
		return math.Trunc(quotient) * step
	default:
		return math.Floor(quotient+0.5) * step
	}
}

func hypotValues(values []Value) (Value, bool) {
	first := values[0]
	if first.Kind == Other && first.Unit == "" {
		sum := 0.0
		for _, value := range values {
			number, ok := numericValue(value)
			if !ok {
				return Value{}, false
			}
			sum += number * number
		}
		return Value{Kind: Other, Number: math.Sqrt(sum)}, true
	}
	if first.Kind == Length || first.Kind == Angle || first.Kind == Time || first.Kind == Frequency || first.Kind == Resolution {
		sum := 0.0
		for _, value := range values {
			if value.Kind != first.Kind || value.Unit != first.Unit {
				return Value{}, false
			}
			sum += value.Number * value.Number
		}
		return Value{Kind: first.Kind, Unit: first.Unit, Number: math.Sqrt(sum)}, true
	}
	return Value{}, false
}

func simplifyLog(values []Value) (Value, bool) {
	if len(values) != 1 && len(values) != 2 {
		return Value{}, false
	}
	number, ok := numericValue(values[0])
	if !ok || number <= 0 {
		return Value{}, false
	}
	if len(values) == 1 {
		return Value{Kind: Other, Number: math.Log(number)}, true
	}
	base, ok := numericValue(values[1])
	if !ok || base <= 0 || base == 1 {
		return Value{}, false
	}
	return Value{Kind: Other, Number: math.Log(number) / math.Log(base)}, true
}

func compareValues(left, right Value) (int, bool) {
	leftNumber, rightNumber, _, ok := compatibleNumbers(left, right)
	if !ok {
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

// compatibleNumbers returns the numeric parts of two values that can be
// compared or combined, along with a value carrying the result's kind and
// unit.
func compatibleNumbers(left, right Value) (float64, float64, Value, bool) {
	if left.Kind == right.Kind && left.Unit == right.Unit {
		return left.Number, right.Number, Value{Kind: left.Kind, Unit: left.Unit}, true
	}
	if left.Kind == Length && right.Kind == Length {
		leftPixels, leftOK := absoluteToPixels(left)
		rightPixels, rightOK := absoluteToPixels(right)
		if leftOK && rightOK {
			return leftPixels, rightPixels, Value{Kind: Length, Unit: "px"}, true
		}
	}
	return 0, 0, Value{}, false
}

// ParseMath parses a CSS math function.
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
	case "min", "max", "hypot":
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
	case "round":
		args, strategy, ok := parseRoundArguments(function.Value)
		if !ok {
			return nil, false
		}
		return &MathExpression{Kind: MathFunction, Name: name, Strategy: strategy, Args: args}, true
	case "log":
		args, ok := parseMathArguments(function.Value)
		if !ok || len(args) < 1 || len(args) > 2 {
			return nil, false
		}
		return &MathExpression{Kind: MathFunction, Name: name, Args: args}, true
	case "mod", "rem", "pow", "atan2":
		args, ok := parseMathArguments(function.Value)
		if !ok || len(args) != 2 {
			return nil, false
		}
		return &MathExpression{Kind: MathFunction, Name: name, Args: args}, true
	case "abs", "sign", "sqrt", "exp", "sin", "cos", "tan", "asin", "acos", "atan":
		args, ok := parseMathArguments(function.Value)
		if !ok || len(args) != 1 {
			return nil, false
		}
		return &MathExpression{Kind: MathFunction, Name: name, Args: args}, true
	default:
		return nil, false
	}
}

func parseRoundArguments(values []parser.ComponentValue) ([]*MathExpression, string, bool) {
	groups := splitCommas(values)
	strategy := "nearest"
	if len(groups) > 0 {
		if token, ok := singleIdent(groups[0]); ok {
			switch strings.ToLower(token.Value) {
			case "nearest", "up", "down", "to-zero":
				strategy = strings.ToLower(token.Value)
				groups = groups[1:]
			}
		}
	}
	if len(groups) != 2 {
		return nil, "", false
	}
	args := make([]*MathExpression, 0, len(groups))
	for _, group := range groups {
		state := &mathState{values: group}
		expression, ok := state.parseSum()
		if !ok {
			return nil, "", false
		}
		state.skipWhitespace()
		if !parser.IsEOF(state.next()) {
			return nil, "", false
		}
		args = append(args, expression)
	}
	return args, strategy, true
}

func singleIdent(values []parser.ComponentValue) (tokenizer.Token, bool) {
	trimmed := []parser.ComponentValue{}
	for _, value := range values {
		if parser.IsWhitespace(value) {
			continue
		}
		trimmed = append(trimmed, value)
	}
	if len(trimmed) != 1 {
		return tokenizer.Token{}, false
	}
	token, ok := parser.AsToken(trimmed[0])
	if !ok || token.Type != tokenizer.TokenIdent {
		return tokenizer.Token{}, false
	}
	return token, true
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
