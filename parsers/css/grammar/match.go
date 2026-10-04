package grammar

import (
	"gonetsurf/parsers/css/parser"
	"gonetsurf/parsers/css/tokenizer"
)

// TypeMatcher attempts to match a named type such as <length> at the start of
// the given component values. It returns the number of component values
// consumed, or zero when the values do not match.
type TypeMatcher func(values []parser.ComponentValue) int

// Context resolves types and property references while matching. Properties
// maps a property name to its grammar definition string.
type Context struct {
	Types      map[string]TypeMatcher
	Properties map[string]string

	cache map[string]Expression
	depth int
}

// NewContext returns an empty matching context.
func NewContext() *Context {
	return &Context{
		Types:      map[string]TypeMatcher{},
		Properties: map[string]string{},
		cache:      map[string]Expression{},
	}
}

// Match is the result of a successful grammar match.
type Match struct {
	// Values is the component value prefix consumed by the grammar.
	Values []parser.ComponentValue
	// Captures maps reference names to the values they consumed.
	Captures map[string][]parser.ComponentValue
}

// MatchDefinition parses definition and matches it against values. The match
// must consume every non-whitespace component value.
func MatchDefinition(definition string, values []parser.ComponentValue, context *Context) (*Match, error) {
	expression, err := Parse(definition)
	if err != nil {
		return nil, err
	}
	state := &state{values: values, context: context, captures: map[string][]parser.ComponentValue{}}
	if !matchExpression(expression, state) {
		return nil, nil
	}
	state.skipWhitespace()
	if state.position != len(values) {
		return nil, nil
	}
	return &Match{Values: values[:state.position], Captures: state.captures}, nil
}

// MatchExpression matches a pre-parsed expression against the start of values.
func (context *Context) MatchExpression(expression Expression, values []parser.ComponentValue) (*Match, error) {
	state := &state{values: values, context: context, captures: map[string][]parser.ComponentValue{}}
	if !matchExpression(expression, state) {
		return nil, nil
	}
	return &Match{Values: values[:state.position], Captures: state.captures}, nil
}

type state struct {
	values   []parser.ComponentValue
	position int
	context  *Context
	captures map[string][]parser.ComponentValue
}

func (state *state) skipWhitespace() {
	for state.position < len(state.values) && parser.IsWhitespace(state.values[state.position]) {
		state.position++
	}
}

func (state *state) next() parser.ComponentValue {
	if state.position < len(state.values) {
		return state.values[state.position]
	}
	return nil
}

func matchExpression(expression Expression, state *state) bool {
	switch typed := expression.(type) {
	case *Reference:
		return matchReference(typed, state)
	case *Keyword:
		return matchKeyword(typed, state)
	case *Literal:
		return matchLiteral(typed, state)
	case *Sequence:
		return matchSequence(typed, state)
	case *OneOf:
		return matchOneOf(typed, state)
	case *AnyOrder:
		return matchAnyOrder(typed, state)
	case *AllAnyOrder:
		return matchAllAnyOrder(typed, state)
	case *Repeat:
		return matchRepeat(typed, state)
	case *Required:
		return matchRequired(typed, state)
	default:
		return false
	}
}

func matchReference(reference *Reference, state *state) bool {
	state.skipWhitespace()
	start := state.position
	if reference.Property {
		definition, ok := state.context.Properties[reference.Name]
		if !ok {
			return false
		}
		expression, err := state.context.cachedExpression(reference.Name, definition)
		if err != nil {
			return false
		}
		if !matchExpression(expression, state) {
			return false
		}
		state.captures[reference.Name] = state.values[start:state.position]
		return true
	}
	matcher, ok := state.context.Types[reference.Name]
	if !ok {
		return false
	}
	consumed := matcher(state.values[state.position:])
	if consumed <= 0 {
		return false
	}
	state.position += consumed
	state.captures[reference.Name] = state.values[start:state.position]
	return true
}

func (context *Context) cachedExpression(name, definition string) (Expression, error) {
	if expression, ok := context.cache[name]; ok {
		return expression, nil
	}
	expression, err := Parse(definition)
	if err != nil {
		return nil, err
	}
	context.cache[name] = expression
	return expression, nil
}

func matchKeyword(keyword *Keyword, state *state) bool {
	state.skipWhitespace()
	token, ok := parser.AsToken(state.next())
	if !ok || token.Type != tokenizer.TokenIdent || !asciiEqualFold(token.Value, keyword.Name) {
		return false
	}
	state.position++
	return true
}

func matchLiteral(literal *Literal, state *state) bool {
	state.skipWhitespace()
	value := state.next()
	token, ok := parser.AsToken(value)
	if !ok {
		return false
	}
	switch literal.Value {
	case ",":
		if token.Type == tokenizer.TokenComma {
			state.position++
			return true
		}
		return false
	case "/":
		if token.Type == tokenizer.TokenDelim && token.Value == "/" {
			state.position++
			return true
		}
		return false
	default:
		if token.Type == tokenizer.TokenDelim && token.Value == literal.Value {
			state.position++
			return true
		}
		return false
	}
}

func matchSequence(sequence *Sequence, state *state) bool {
	start := state.position
	for _, item := range sequence.Items {
		if !matchExpression(item, state) {
			state.position = start
			return false
		}
	}
	return true
}

func matchOneOf(oneOf *OneOf, state *state) bool {
	for _, item := range oneOf.Items {
		start := state.position
		if matchExpression(item, state) {
			return true
		}
		state.position = start
	}
	return false
}

func matchAnyOrder(anyOrder *AnyOrder, state *state) bool {
	matched := make([]bool, len(anyOrder.Items))
	count := 0
	for {
		progressed := false
		for index, item := range anyOrder.Items {
			if matched[index] {
				continue
			}
			start := state.position
			if matchExpression(item, state) {
				matched[index] = true
				count++
				progressed = true
			} else {
				state.position = start
			}
		}
		if !progressed {
			break
		}
	}
	return count >= 1
}

func matchAllAnyOrder(all *AllAnyOrder, state *state) bool {
	start := state.position
	matched := make([]bool, len(all.Items))
	count := 0
	// First match every item that consumes component values, so optional items
	// are not marked before a required item has had a chance to consume.
	for {
		progressed := false
		for index, item := range all.Items {
			if matched[index] {
				continue
			}
			position := state.position
			if matchExpression(item, state) && state.position > position {
				matched[index] = true
				count++
				progressed = true
			} else {
				state.position = position
			}
		}
		if !progressed {
			break
		}
	}
	// Then allow the remaining (optional) items to match empty.
	for index, item := range all.Items {
		if matched[index] {
			continue
		}
		position := state.position
		if matchExpression(item, state) {
			matched[index] = true
			count++
		} else {
			state.position = position
		}
	}
	if count != len(all.Items) {
		state.position = start
		return false
	}
	return true
}

func matchRepeat(repeat *Repeat, state *state) bool {
	count := 0
	for repeat.Max < 0 || count < repeat.Max {
		if repeat.Separated && count > 0 {
			start := state.position
			state.skipWhitespace()
			token, ok := parser.AsToken(state.next())
			if !ok || token.Type != tokenizer.TokenComma {
				state.position = start
				break
			}
			state.position++
		}
		start := state.position
		if !matchExpression(repeat.Item, state) {
			state.position = start
			break
		}
		if state.position == start {
			break
		}
		count++
	}
	return count >= repeat.Min
}

func matchRequired(required *Required, state *state) bool {
	start := state.position
	if !matchExpression(required.Item, state) {
		return false
	}
	if state.position == start {
		state.position = start
		return false
	}
	return true
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
