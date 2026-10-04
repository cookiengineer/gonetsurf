package cssom

import (
	"strings"

	"gonetsurf/parsers/css/parser"
	"gonetsurf/parsers/css/selectors"
	"gonetsurf/parsers/css/tokenizer"
)

// Stylesheet is a parsed list of CSS rules.
type Stylesheet struct {
	Rules []Rule
}

// Rule is a stylesheet rule: a style rule or an at-rule.
type Rule interface {
	rule()
}

// StyleRule is a qualified rule with a selector prelude and a declaration
// block.
type StyleRule struct {
	Selectors    *selectors.SelectorList
	Prelude      []parser.ComponentValue
	Declarations []*Declaration
	// Nested holds nested rules (CSS Nesting).
	Nested []Rule
}

func (*StyleRule) rule() {}

// AtRule is an at-rule such as @media or @import.
type AtRule struct {
	Name         string
	Prelude      []parser.ComponentValue
	Block        []parser.ComponentValue
	Declarations []*Declaration
	Rules        []Rule
}

func (*AtRule) rule() {}

// Declaration is a property declaration.
type Declaration struct {
	Name      string
	Value     []parser.ComponentValue
	Important bool
}

// Parse parses a stylesheet, ignoring CDO and CDC tokens between rules.
func Parse(input string) *Stylesheet {
	stylesheet := &Stylesheet{Rules: []Rule{}}
	for _, node := range parser.ParseStylesheet(input) {
		if rule := convertRule(node); rule != nil {
			stylesheet.Rules = append(stylesheet.Rules, rule)
		}
	}
	return stylesheet
}

// ParseRule parses a single rule.
func ParseRule(input string) (Rule, error) {
	node, err := parser.ParseRule(input)
	if err != nil {
		return nil, err
	}
	return convertRule(node), nil
}

func convertRule(node parser.Node) Rule {
	switch typed := node.(type) {
	case *parser.QualifiedRule:
		rule := &StyleRule{Prelude: typed.Prelude}
		if list, err := selectors.ParseValues(typed.Prelude); err == nil {
			rule.Selectors = list
		}
		if typed.Block != nil {
			rule.Declarations = convertDeclarations(typed.Block.Value)
		}
		return rule
	case *parser.AtRule:
		rule := &AtRule{Name: typed.Name, Prelude: typed.Prelude}
		if typed.Block != nil {
			rule.Block = typed.Block.Value
			rule.Declarations = convertDeclarations(typed.Block.Value)
		}
		return rule
	default:
		return nil
	}
}

func convertDeclarations(values []parser.ComponentValue) []*Declaration {
	var declarations []*Declaration
	for _, node := range parser.ParseListOfDeclarationsValues(values) {
		declaration, ok := node.(*parser.Declaration)
		if !ok {
			continue
		}
		declarations = append(declarations, &Declaration{
			Name:      declaration.Name,
			Value:     declaration.Value,
			Important: declaration.Important,
		})
	}
	return declarations
}

// Serialize returns the stylesheet as CSS text.
func (stylesheet *Stylesheet) Serialize() string {
	var builder strings.Builder
	for _, rule := range stylesheet.Rules {
		builder.WriteString(serializeRule(rule))
	}
	return builder.String()
}

func serializeRule(rule Rule) string {
	switch typed := rule.(type) {
	case *StyleRule:
		return typed.Serialize()
	case *AtRule:
		return typed.Serialize()
	default:
		return ""
	}
}

// Serialize returns the style rule as CSS text.
func (rule *StyleRule) Serialize() string {
	var builder strings.Builder
	builder.WriteString(strings.TrimSpace(parser.SerializeValues(rule.Prelude)))
	builder.WriteString(" {")
	for _, declaration := range rule.Declarations {
		builder.WriteString(" ")
		builder.WriteString(declaration.Serialize())
	}
	for _, nested := range rule.Nested {
		builder.WriteString(" ")
		builder.WriteString(strings.TrimSpace(serializeRule(nested)))
	}
	builder.WriteString(" }")
	return builder.String()
}

// Serialize returns the at-rule as CSS text.
func (rule *AtRule) Serialize() string {
	var builder strings.Builder
	builder.WriteString("@")
	builder.WriteString(rule.Name)
	prelude := strings.TrimSpace(parser.SerializeValues(rule.Prelude))
	if prelude != "" {
		builder.WriteString(" ")
		builder.WriteString(prelude)
	}
	if rule.Block == nil {
		builder.WriteString(";")
		return builder.String()
	}
	body := ""
	for _, declaration := range rule.Declarations {
		body += " " + declaration.Serialize()
	}
	if body == "" {
		body = parser.SerializeValues(rule.Block)
	}
	body = strings.TrimSpace(body)
	builder.WriteString(" {")
	if body != "" {
		builder.WriteString(" ")
		builder.WriteString(body)
	}
	builder.WriteString(" }")
	return builder.String()
}

// Serialize returns the declaration as CSS text, including the trailing
// semicolon.
func (declaration *Declaration) Serialize() string {
	value := strings.TrimSpace(parser.SerializeValues(declaration.Value))
	var builder strings.Builder
	builder.WriteString(declaration.Name)
	builder.WriteString(": ")
	builder.WriteString(value)
	if declaration.Important {
		builder.WriteString(" !important")
	}
	builder.WriteString(";")
	return builder.String()
}

// ExpectIdent reports whether a token is an identifier with the given
// ASCII-case-insensitive value.
func ExpectIdent(token tokenizer.Token, name string) bool {
	if token.Type != tokenizer.TokenIdent {
		return false
	}
	return strings.EqualFold(token.Value, name)
}
