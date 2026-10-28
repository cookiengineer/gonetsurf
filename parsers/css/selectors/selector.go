// Package selectors implements the Selectors Level 4 syntax.
//
// It parses selector lists into an abstract syntax tree, computes
// specificity, and provides the An+B microsyntax used by the structural
// pseudo-classes.
//
// See specifications/css/selectors-4.html and specifications/css/syntax-3.html
// §6.
package selectors

import (
	"fmt"
	"strings"

	"gonetsurf/parsers/css/parser"
	"gonetsurf/parsers/css/tokenizer"
)

// Combinator is the relationship between two compound selectors.
type Combinator uint8

const (
	// Descendant is the whitespace combinator.
	Descendant Combinator = iota
	// Child is the ">" combinator.
	Child
	// NextSibling is the "+" combinator.
	NextSibling
	// SubsequentSibling is the "~" combinator.
	SubsequentSibling
	// Column is the "||" combinator.
	Column
)

func (combinator Combinator) String() string {
	switch combinator {
	case Child:
		return ">"
	case NextSibling:
		return "+"
	case SubsequentSibling:
		return "~"
	case Column:
		return "||"
	default:
		return " "
	}
}

// ComplexSelector is a sequence of compound selectors joined by combinators.
// Leading is set for a relative selector (inside :has()) that begins with a
// combinator.
type ComplexSelector struct {
	Leading     *Combinator
	Compounds   []CompoundSelector
	Combinators []Combinator
}

// CompoundSelector is a sequence of simple selectors with no combinator.
type CompoundSelector struct {
	Parts []SimpleSelector
}

// SimpleSelector is one simple selector.
type SimpleSelector interface {
	simpleSelector()
}

// TypeSelector matches elements by (optionally namespaced) name.
type TypeSelector struct {
	Namespace string
	Name      string
	Universal bool
}

func (*TypeSelector) simpleSelector() {}

// IDSelector matches an element by id.
type IDSelector struct{ Name string }

func (*IDSelector) simpleSelector() {}

// ClassSelector matches an element by class.
type ClassSelector struct{ Name string }

func (*ClassSelector) simpleSelector() {}

// AttributeSelector matches an element by attribute.
type AttributeSelector struct {
	Namespace       string
	Name            string
	Operator        string
	Value           string
	CaseInsensitive bool
}

func (*AttributeSelector) simpleSelector() {}

// PseudoClass matches an element by pseudo-class.
type PseudoClass struct {
	Name       string
	Argument   string
	Selectors  *SelectorList
	AnB        *AnB
	OfSelector *SelectorList
}

func (*PseudoClass) simpleSelector() {}

// PseudoElement matches a pseudo-element.
type PseudoElement struct {
	Name     string
	Argument string
}

func (*PseudoElement) simpleSelector() {}

// NestingSelector is the & nesting selector (CSS Nesting).
type NestingSelector struct{}

func (*NestingSelector) simpleSelector() {}

// SelectorList is a comma-separated list of complex selectors.
type SelectorList struct {
	Selectors []ComplexSelector
}

// Parse parses a selector list from a string.
func Parse(input string) (*SelectorList, error) {
	return ParseValues(parser.ParseComponentValueList(input))
}

// ParseValues parses a selector list from component values.
func ParseValues(values []parser.ComponentValue) (*SelectorList, error) {
	return parseSelectorValues(values, false)
}

func parseSelectorValues(values []parser.ComponentValue, relative bool) (*SelectorList, error) {
	state := &selectorState{values: values}
	list, err := state.parseSelectorListRelative(relative)
	if err != nil {
		return nil, err
	}
	state.skipWhitespace()
	if !parser.IsEOF(state.next()) {
		return nil, fmt.Errorf("css selectors: unexpected input")
	}
	return list, nil
}

type selectorState struct {
	values []parser.ComponentValue
	pos    int
}

func (state *selectorState) next() parser.ComponentValue {
	if state.pos < len(state.values) {
		return state.values[state.pos]
	}
	return eofValue()
}

func (state *selectorState) advance() parser.ComponentValue {
	value := state.next()
	if state.pos < len(state.values) {
		state.pos++
	}
	return value
}

func (state *selectorState) skipWhitespace() {
	for parser.IsWhitespace(state.next()) {
		state.pos++
	}
}

func eofValue() parser.ComponentValue {
	return parser.Token{Token: tokenizer.Token{Type: tokenizer.TokenEOF}}
}

func (state *selectorState) parseSelectorListRelative(relative bool) (*SelectorList, error) {
	list := &SelectorList{}
	for {
		state.skipWhitespace()
		complexSelector, err := state.parseComplexSelector(relative)
		if err != nil {
			return nil, err
		}
		list.Selectors = append(list.Selectors, *complexSelector)
		state.skipWhitespace()
		token, ok := parser.AsToken(state.next())
		if !ok || token.Type != tokenizer.TokenComma {
			break
		}
		state.pos++
	}
	if len(list.Selectors) == 0 {
		return nil, fmt.Errorf("css selectors: empty selector list")
	}
	return list, nil
}

func (state *selectorState) parseComplexSelector(relative bool) (*ComplexSelector, error) {
	complexSelector := &ComplexSelector{}
	if relative {
		if combinator, ok := state.parseLeadingCombinator(); ok {
			complexSelector.Leading = &combinator
			state.skipWhitespace()
		}
	}
	compound, err := state.parseCompoundSelector()
	if err != nil {
		return nil, err
	}
	complexSelector.Compounds = append(complexSelector.Compounds, *compound)
	for {
		combinator, hasCombinator := state.parseCombinator()
		if !hasCombinator {
			break
		}
		if compoundEndsWithPseudoElement(complexSelector.Compounds[len(complexSelector.Compounds)-1]) {
			return nil, fmt.Errorf("css selectors: combinator is not allowed after a pseudo-element")
		}
		compound, err := state.parseCompoundSelector()
		if err != nil {
			return nil, err
		}
		complexSelector.Combinators = append(complexSelector.Combinators, combinator)
		complexSelector.Compounds = append(complexSelector.Compounds, *compound)
	}
	return complexSelector, nil
}

func compoundEndsWithPseudoElement(compound CompoundSelector) bool {
	if len(compound.Parts) == 0 {
		return false
	}
	_, ok := compound.Parts[len(compound.Parts)-1].(*PseudoElement)
	return ok
}

// parseLeadingCombinator consumes a combinator at the start of a relative
// selector, if present. It does not consume descendant whitespace.
func (state *selectorState) parseLeadingCombinator() (Combinator, bool) {
	token, ok := parser.AsToken(state.next())
	if !ok || token.Type != tokenizer.TokenDelim {
		return Descendant, false
	}
	switch {
	case token.Value == ">":
		state.pos++
		return Child, true
	case token.Value == "+":
		state.pos++
		return NextSibling, true
	case token.Value == "~":
		state.pos++
		return SubsequentSibling, true
	case token.Value == "|" && isDelimAt(state, "|"):
		state.pos += 2
		return Column, true
	default:
		return Descendant, false
	}
}

func (state *selectorState) parseCombinator() (Combinator, bool) {
	hadWhitespace := false
	for parser.IsWhitespace(state.next()) {
		state.pos++
		hadWhitespace = true
	}
	token, ok := parser.AsToken(state.next())
	if ok {
		switch {
		case token.Type == tokenizer.TokenDelim && token.Value == ">":
			state.pos++
			state.skipWhitespace()
			return Child, true
		case token.Type == tokenizer.TokenDelim && token.Value == "+":
			state.pos++
			state.skipWhitespace()
			return NextSibling, true
		case token.Type == tokenizer.TokenDelim && token.Value == "~":
			state.pos++
			state.skipWhitespace()
			return SubsequentSibling, true
		case token.Type == tokenizer.TokenDelim && token.Value == "|" && isDelimAt(state, "|"):
			state.pos += 2
			state.skipWhitespace()
			return Column, true
		}
	}
	if hadWhitespace && state.startsCompound() {
		return Descendant, true
	}
	return Descendant, false
}

func isDelimAt(state *selectorState, value string) bool {
	token, ok := parser.AsToken(state.nextAt(1))
	return ok && token.Type == tokenizer.TokenDelim && token.Value == value
}

func (state *selectorState) nextAt(offset int) parser.ComponentValue {
	index := state.pos + offset
	if index < len(state.values) {
		return state.values[index]
	}
	return eofValue()
}

func (state *selectorState) startsCompound() bool {
	token, ok := parser.AsToken(state.next())
	if !ok {
		_, block := state.next().(*parser.Block)
		return block
	}
	switch token.Type {
	case tokenizer.TokenIdent, tokenizer.TokenHash, tokenizer.TokenColon, tokenizer.TokenOpenSquare:
		return true
	case tokenizer.TokenDelim:
		switch token.Value {
		case ".", "*", "&", "|":
			return true
		}
	}
	return false
}

func (state *selectorState) parseCompoundSelector() (*CompoundSelector, error) {
	compound := &CompoundSelector{}
	pseudoElement := ""
loop:
	for {
		token, ok := parser.AsToken(state.next())
		if !ok {
			if block, isBlock := state.next().(*parser.Block); isBlock && block.Associated == tokenizer.TokenOpenSquare {
				attribute, err := parseAttributeSelector(block)
				if err != nil {
					return nil, err
				}
				compound.Parts = append(compound.Parts, attribute)
				state.pos++
				continue
			}
			break
		}
		switch {
		case token.Type == tokenizer.TokenIdent:
			if len(compound.Parts) > 0 {
				break loop
			}
			if isPipe(state.nextAt(1)) {
				state.pos += 2
				simple, err := state.parseTypeName(token.Value)
				if err != nil {
					return nil, err
				}
				compound.Parts = append(compound.Parts, simple)
				continue
			}
			compound.Parts = append(compound.Parts, &TypeSelector{Name: token.Value})
			state.pos++
		case token.Type == tokenizer.TokenDelim && token.Value == "*":
			if len(compound.Parts) > 0 {
				break loop
			}
			if isPipe(state.nextAt(1)) {
				state.pos += 2
				simple, err := state.parseTypeName("*")
				if err != nil {
					return nil, err
				}
				compound.Parts = append(compound.Parts, simple)
				continue
			}
			compound.Parts = append(compound.Parts, &TypeSelector{Universal: true})
			state.pos++
		case token.Type == tokenizer.TokenDelim && token.Value == "|":
			if len(compound.Parts) > 0 {
				break loop
			}
			state.pos++
			simple, err := state.parseTypeName("")
			if err != nil {
				return nil, err
			}
			compound.Parts = append(compound.Parts, simple)
			continue
		case token.Type == tokenizer.TokenDelim && token.Value == "&":
			compound.Parts = append(compound.Parts, &NestingSelector{})
			state.pos++
		case token.Type == tokenizer.TokenDelim && token.Value == ".":
			state.pos++
			nameToken, ok := parser.AsToken(state.next())
			if !ok || nameToken.Type != tokenizer.TokenIdent {
				return nil, fmt.Errorf("css selectors: expected class name")
			}
			compound.Parts = append(compound.Parts, &ClassSelector{Name: nameToken.Value})
			state.pos++
		case token.Type == tokenizer.TokenHash:
			if token.HashFlag != tokenizer.HashID {
				return nil, fmt.Errorf("css selectors: #%s is not a valid id selector", token.Value)
			}
			compound.Parts = append(compound.Parts, &IDSelector{Name: token.Value})
			state.pos++
		case token.Type == tokenizer.TokenColon:
			simple, err := state.parsePseudo()
			if err != nil {
				return nil, err
			}
			if pseudo, ok := simple.(*PseudoClass); ok && pseudoElement != "" {
				if pseudoElement != "part" || strings.EqualFold(pseudo.Name, "has") {
					return nil, fmt.Errorf("css selectors: pseudo-class :%s is not allowed after ::%s", pseudo.Name, pseudoElement)
				}
			}
			if element, ok := simple.(*PseudoElement); ok {
				pseudoElement = strings.ToLower(element.Name)
			}
			compound.Parts = append(compound.Parts, simple)
		default:
			break loop
		}
	}
	if len(compound.Parts) == 0 {
		return nil, fmt.Errorf("css selectors: empty compound selector")
	}
	return compound, nil
}

func (state *selectorState) parseTypeName(namespace string) (SimpleSelector, error) {
	token, ok := parser.AsToken(state.next())
	if !ok {
		return nil, fmt.Errorf("css selectors: expected type name after namespace")
	}
	switch {
	case token.Type == tokenizer.TokenIdent:
		state.pos++
		return &TypeSelector{Namespace: namespace, Name: token.Value}, nil
	case token.Type == tokenizer.TokenDelim && token.Value == "*":
		state.pos++
		return &TypeSelector{Namespace: namespace, Universal: true}, nil
	default:
		return nil, fmt.Errorf("css selectors: expected type name after namespace")
	}
}

func (state *selectorState) parsePseudo() (SimpleSelector, error) {
	state.pos++ // consume ':'
	isElement := false
	if token, ok := parser.AsToken(state.next()); ok && token.Type == tokenizer.TokenColon {
		isElement = true
		state.pos++
	}
	value := state.next()
	if token, ok := parser.AsToken(value); ok && token.Type == tokenizer.TokenIdent {
		state.pos++
		if isElement {
			if !isKnownPseudoElement(token.Value) {
				return nil, fmt.Errorf("css selectors: unknown pseudo-element ::%s", token.Value)
			}
			if isFunctionalPseudoElement(token.Value) {
				return nil, fmt.Errorf("css selectors: pseudo-element ::%s requires arguments", token.Value)
			}
			return &PseudoElement{Name: token.Value}, nil
		}
		if isLegacyPseudoElement(token.Value) {
			return &PseudoElement{Name: token.Value}, nil
		}
		if !isKnownPseudoClass(token.Value) || requiresFunctionPseudoClass(token.Value) {
			return nil, fmt.Errorf("css selectors: unknown pseudo-class :%s", token.Value)
		}
		return &PseudoClass{Name: token.Value}, nil
	}
	if function, ok := value.(*parser.Function); ok {
		state.pos++
		if isElement {
			if !isKnownPseudoElement(function.Name) {
				return nil, fmt.Errorf("css selectors: unknown pseudo-element ::%s()", function.Name)
			}
			if err := validatePseudoElementArgument(function); err != nil {
				return nil, err
			}
			return &PseudoElement{Name: function.Name, Argument: serialize(function.Value)}, nil
		}
		if !isKnownPseudoClass(function.Name) {
			return nil, fmt.Errorf("css selectors: unknown pseudo-class :%s()", function.Name)
		}
		return parseFunctionalPseudoClass(function)
	}
	return nil, fmt.Errorf("css selectors: expected pseudo name")
}

func validatePseudoElementArgument(function *parser.Function) error {
	switch strings.ToLower(function.Name) {
	case "part":
		trimmed := trimWhitespace(function.Value)
		if len(trimmed) == 0 {
			return fmt.Errorf("css selectors: ::part() requires an identifier")
		}
		for _, value := range trimmed {
			if parser.IsWhitespace(value) {
				continue
			}
			token, ok := parser.AsToken(value)
			if !ok || token.Type != tokenizer.TokenIdent {
				return fmt.Errorf("css selectors: ::part() requires identifiers")
			}
		}
	case "slotted":
		state := &selectorState{values: function.Value}
		state.skipWhitespace()
		compound, err := state.parseCompoundSelector()
		if err != nil {
			return err
		}
		state.skipWhitespace()
		if !parser.IsEOF(state.next()) {
			return fmt.Errorf("css selectors: ::slotted() requires a compound selector")
		}
		for _, part := range compound.Parts {
			if _, ok := part.(*PseudoElement); ok {
				return fmt.Errorf("css selectors: ::slotted() cannot contain a pseudo-element")
			}
		}
	}
	return nil
}

func isLegacyPseudoElement(name string) bool {
	switch strings.ToLower(name) {
	case "before", "after", "first-line", "first-letter":
		return true
	}
	return false
}

func parseFunctionalPseudoClass(function *parser.Function) (SimpleSelector, error) {
	name := strings.ToLower(function.Name)
	switch name {
	case "is", "where", "not":
		list, err := parseSelectorValues(function.Value, false)
		if err != nil {
			return nil, err
		}
		if containsPseudoElement(list) {
			return nil, fmt.Errorf("css selectors: :%s() cannot contain a pseudo-element", function.Name)
		}
		return &PseudoClass{Name: function.Name, Selectors: list}, nil
	case "has":
		list, err := parseSelectorValues(function.Value, true)
		if err != nil {
			return nil, err
		}
		if containsHas(list) {
			return nil, fmt.Errorf("css selectors: :has() cannot be nested")
		}
		if containsPseudoElement(list) {
			return nil, fmt.Errorf("css selectors: :has() cannot contain a pseudo-element")
		}
		return &PseudoClass{Name: function.Name, Selectors: list}, nil
	case "host":
		list, err := parseSelectorValues(function.Value, false)
		if err != nil {
			return nil, err
		}
		if !allCompoundSelectors(list) {
			return nil, fmt.Errorf("css selectors: :host() requires a compound selector")
		}
		return &PseudoClass{Name: function.Name, Selectors: list}, nil
	case "state":
		if !isIdentArgument(function.Value) {
			return nil, fmt.Errorf("css selectors: :state() requires an identifier")
		}
		return &PseudoClass{Name: function.Name, Argument: serialize(function.Value)}, nil
	case "nth-child", "nth-last-child", "nth-of-type", "nth-last-of-type", "nth-col", "nth-last-col":
		anbValues, ofValues := splitOf(function.Value)
		anb, ok := ParseAnBValues(anbValues)
		if !ok {
			return nil, fmt.Errorf("css selectors: invalid An+B in :%s", function.Name)
		}
		pseudo := &PseudoClass{Name: function.Name, AnB: &anb}
		if ofValues != nil {
			list, err := ParseValues(ofValues)
			if err != nil {
				return nil, err
			}
			pseudo.OfSelector = list
		}
		return pseudo, nil
	default:
		return &PseudoClass{Name: function.Name, Argument: serialize(function.Value)}, nil
	}
}

func containsPseudoElement(list *SelectorList) bool {
	if list == nil {
		return false
	}
	for _, complexSelector := range list.Selectors {
		for _, compound := range complexSelector.Compounds {
			for _, part := range compound.Parts {
				if _, ok := part.(*PseudoElement); ok {
					return true
				}
				if pseudo, ok := part.(*PseudoClass); ok && pseudo.Selectors != nil && containsPseudoElement(pseudo.Selectors) {
					return true
				}
			}
		}
	}
	return false
}

func allCompoundSelectors(list *SelectorList) bool {
	if list == nil {
		return false
	}
	for _, complexSelector := range list.Selectors {
		if len(complexSelector.Compounds) != 1 || len(complexSelector.Combinators) != 0 {
			return false
		}
		for _, part := range complexSelector.Compounds[0].Parts {
			if pseudo, ok := part.(*PseudoClass); ok && pseudo.Selectors != nil && !allCompoundSelectors(pseudo.Selectors) {
				return false
			}
		}
	}
	return true
}

// containsHas reports whether a selector list uses :has() at any depth.
func containsHas(list *SelectorList) bool {
	if list == nil {
		return false
	}
	for _, complexSelector := range list.Selectors {
		for _, compound := range complexSelector.Compounds {
			for _, part := range compound.Parts {
				pseudo, ok := part.(*PseudoClass)
				if !ok {
					continue
				}
				if strings.EqualFold(pseudo.Name, "has") {
					return true
				}
				if pseudo.Selectors != nil && containsHas(pseudo.Selectors) {
					return true
				}
			}
		}
	}
	return false
}

func splitOf(values []parser.ComponentValue) (anbValues, ofValues []parser.ComponentValue) {
	for index, value := range values {
		if token, ok := parser.AsToken(value); ok && token.Type == tokenizer.TokenIdent && strings.EqualFold(token.Value, "of") {
			return values[:index], values[index+1:]
		}
	}
	return values, nil
}

func parseAttributeSelector(block *parser.Block) (*AttributeSelector, error) {
	attribute := &AttributeSelector{}
	trimmed := trimWhitespace(block.Value)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("css selectors: empty attribute selector")
	}
	index := 0
	// Optional namespace: ns|name, *|name, |name.
	if token, ok := parser.AsToken(trimmed[index]); ok {
		switch {
		case token.Type == tokenizer.TokenIdent && index+1 < len(trimmed) && isNamespacePipe(trimmed, index+1):
			attribute.Namespace = token.Value
			index += 2
		case token.Type == tokenizer.TokenDelim && token.Value == "*" && index+1 < len(trimmed) && isNamespacePipe(trimmed, index+1):
			attribute.Namespace = "*"
			index += 2
		case token.Type == tokenizer.TokenDelim && token.Value == "|":
			index++
		}
	}
	nameToken, ok := parser.AsToken(trimmed[index])
	if !ok || nameToken.Type != tokenizer.TokenIdent {
		return nil, fmt.Errorf("css selectors: expected attribute name")
	}
	attribute.Name = nameToken.Value
	index = skipWhitespaceFrom(trimmed, index+1)
	if index >= len(trimmed) {
		return attribute, nil
	}
	operator, operatorLength, ok := matchOperator(trimmed[index:])
	if !ok {
		return nil, fmt.Errorf("css selectors: expected match operator")
	}
	attribute.Operator = operator
	index = skipWhitespaceFrom(trimmed, index+operatorLength)
	if index >= len(trimmed) {
		return nil, fmt.Errorf("css selectors: expected attribute value")
	}
	valueToken, ok := parser.AsToken(trimmed[index])
	if !ok || (valueToken.Type != tokenizer.TokenIdent && valueToken.Type != tokenizer.TokenString) {
		return nil, fmt.Errorf("css selectors: expected attribute value")
	}
	attribute.Value = valueToken.Value
	index = skipWhitespaceFrom(trimmed, index+1)
	if index < len(trimmed) {
		flagToken, ok := parser.AsToken(trimmed[index])
		if !ok || flagToken.Type != tokenizer.TokenIdent {
			return nil, fmt.Errorf("css selectors: expected attribute flag")
		}
		switch strings.ToLower(flagToken.Value) {
		case "i":
			attribute.CaseInsensitive = true
		case "s":
		default:
			return nil, fmt.Errorf("css selectors: unknown attribute flag %q", flagToken.Value)
		}
		index++
	}
	if index != len(trimmed) {
		return nil, fmt.Errorf("css selectors: trailing attribute tokens")
	}
	return attribute, nil
}

func isPipe(value parser.ComponentValue) bool {
	token, ok := parser.AsToken(value)
	return ok && token.Type == tokenizer.TokenDelim && token.Value == "|"
}

func isNamespacePipe(values []parser.ComponentValue, index int) bool {
	if index < 0 || index+1 >= len(values) {
		return false
	}
	if !isPipe(values[index]) {
		return false
	}
	next, ok := parser.AsToken(values[index+1])
	return !ok || next.Type != tokenizer.TokenDelim || next.Value != "="
}

// matchOperator recognizes attribute match operators. The tokenizer emits
// "~=", "|=", "^=", "$=", and "*=" as two delimiter tokens, so operators are
// matched over a token slice.
func matchOperator(values []parser.ComponentValue) (string, int, bool) {
	if len(values) == 0 {
		return "", 0, false
	}
	token, ok := parser.AsToken(values[0])
	if !ok || token.Type != tokenizer.TokenDelim {
		return "", 0, false
	}
	if token.Value == "=" {
		return "=", 1, true
	}
	switch token.Value {
	case "~", "|", "^", "$", "*":
		if len(values) >= 2 {
			if next, ok := parser.AsToken(values[1]); ok && next.Type == tokenizer.TokenDelim && next.Value == "=" {
				return token.Value + "=", 2, true
			}
		}
	}
	return "", 0, false
}

// isIdentArgument reports whether the values are a single identifier (as
// required by :state()).
func isIdentArgument(values []parser.ComponentValue) bool {
	trimmed := trimWhitespace(values)
	if len(trimmed) != 1 {
		return false
	}
	token, ok := parser.AsToken(trimmed[0])
	return ok && token.Type == tokenizer.TokenIdent
}

func skipWhitespaceFrom(values []parser.ComponentValue, index int) int {
	for index < len(values) && parser.IsWhitespace(values[index]) {
		index++
	}
	return index
}

func trimWhitespace(values []parser.ComponentValue) []parser.ComponentValue {
	start := 0
	for start < len(values) && parser.IsWhitespace(values[start]) {
		start++
	}
	end := len(values)
	for end > start && parser.IsWhitespace(values[end-1]) {
		end--
	}
	return values[start:end]
}

// serialize renders component values back to text for opaque pseudo arguments
// such as :lang(en) or :dir(ltr).
func serialize(values []parser.ComponentValue) string {
	var builder strings.Builder
	for _, value := range values {
		if token, ok := parser.AsToken(value); ok {
			switch token.Type {
			case tokenizer.TokenString:
				builder.WriteString(token.Value)
			case tokenizer.TokenWhitespace:
				builder.WriteString(" ")
			default:
				builder.WriteString(token.Representation)
			}
			continue
		}
		builder.WriteString("<value>")
	}
	return builder.String()
}
