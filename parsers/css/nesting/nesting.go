// Package nesting implements CSS Nesting: parsing nested style rules and
// resolving the & nesting selector.
//
// See specifications/css/css-nesting-1.html.
package nesting

import (
	"strings"

	"gonetsurf/parsers/css/parser"
	"gonetsurf/parsers/css/tokenizer"
)

// Declaration is a declaration with its raw value text.
type Declaration struct {
	Name      string
	Value     string
	Important bool
}

// Rule is a style rule that may contain nested rules. Selector is populated
// by Flatten with the resolved selector text.
type Rule struct {
	Prelude      []parser.ComponentValue
	Selector     string
	AtRule       bool
	AtName       string
	Declarations []Declaration
	Nested       []Rule
}

// Parse parses a stylesheet into nested-capable rules.
func Parse(input string) []Rule {
	var rules []Rule
	for _, node := range parser.ParseStylesheet(input) {
		switch typed := node.(type) {
		case *parser.QualifiedRule:
			rules = append(rules, parseStyleRule(typed.Prelude, typed.Block))
		case *parser.AtRule:
			rule := Rule{Prelude: typed.Prelude, AtRule: true, AtName: typed.Name}
			if typed.Block != nil {
				declarations, nested := parseBlock(typed.Block.Value)
				rule.Declarations = declarations
				rule.Nested = nested
			}
			rules = append(rules, rule)
		}
	}
	return rules
}

func parseStyleRule(prelude []parser.ComponentValue, block *parser.Block) Rule {
	rule := Rule{Prelude: prelude}
	if block != nil {
		declarations, nested := parseBlock(block.Value)
		rule.Declarations = declarations
		rule.Nested = nested
	}
	return rule
}

func parseBlock(values []parser.ComponentValue) ([]Declaration, []Rule) {
	var declarations []Declaration
	var rules []Rule
	index := 0
	for index < len(values) {
		for index < len(values) && (parser.IsWhitespace(values[index]) || isSemicolon(values[index])) {
			index++
		}
		if index >= len(values) {
			break
		}
		blockIndex := -1
		scan := index
		for scan < len(values) {
			if isSemicolon(values[scan]) {
				break
			}
			if block, ok := values[scan].(*parser.Block); ok && block.Associated == tokenizer.TokenOpenCurly {
				blockIndex = scan
				break
			}
			scan++
		}
		if blockIndex >= 0 {
			prelude := values[index:blockIndex]
			block := values[blockIndex].(*parser.Block)
			rule := Rule{Prelude: prelude}
			if isAtKeyword(prelude) {
				rule.AtRule = true
				rule.AtName = atName(prelude)
			}
			declarationsInner, nested := parseBlock(block.Value)
			rule.Declarations = declarationsInner
			rule.Nested = nested
			rules = append(rules, rule)
			index = blockIndex + 1
			continue
		}
		segment := values[index:scan]
		declarations = append(declarations, parseDeclarationSegment(segment)...)
		index = scan + 1
	}
	return declarations, rules
}

func isSemicolon(value parser.ComponentValue) bool {
	token, ok := parser.AsToken(value)
	return ok && token.Type == tokenizer.TokenSemicolon
}

func isAtKeyword(values []parser.ComponentValue) bool {
	if len(values) == 0 {
		return false
	}
	token, ok := parser.AsToken(values[0])
	return ok && token.Type == tokenizer.TokenAtKeyword
}

func atName(values []parser.ComponentValue) string {
	token, _ := parser.AsToken(values[0])
	return token.Value
}

func parseDeclarationSegment(values []parser.ComponentValue) []Declaration {
	var declarations []Declaration
	for _, node := range parser.ParseListOfDeclarationsValues(values) {
		declaration, ok := node.(*parser.Declaration)
		if !ok {
			continue
		}
		declarations = append(declarations, Declaration{
			Name:      declaration.Name,
			Value:     strings.TrimSpace(parser.SerializeValues(declaration.Value)),
			Important: declaration.Important,
		})
	}
	return declarations
}

// Flatten expands nested rules into a flat list, resolving & against the
// parent selector.
func Flatten(rules []Rule) []Rule {
	return flattenAll(rules, "")
}

func flattenAll(rules []Rule, parent string) []Rule {
	var result []Rule
	for _, rule := range rules {
		selector := strings.TrimSpace(parser.SerializeValues(rule.Prelude))
		resolved := selector
		if !rule.AtRule && parent != "" {
			resolved = ResolveSelector(selector, parent)
		}
		flat := Rule{
			Prelude:      rule.Prelude,
			Selector:     resolved,
			AtRule:       rule.AtRule,
			AtName:       rule.AtName,
			Declarations: rule.Declarations,
		}
		result = append(result, flat)
		if len(rule.Nested) > 0 {
			childParent := parent
			if !rule.AtRule {
				childParent = resolved
			}
			result = append(result, flattenAll(rule.Nested, childParent)...)
		}
	}
	return result
}

// ResolveSelector substitutes the parent selector for & in a nested selector.
// Without an explicit &, the nested selector is treated as a descendant.
func ResolveSelector(nested, parent string) string {
	nestedParts := splitSelectorList(nested)
	parentParts := splitSelectorList(parent)
	var resolved []string
	for _, nestedPart := range nestedParts {
		nestedPart = strings.TrimSpace(nestedPart)
		for _, parentPart := range parentParts {
			parentPart = strings.TrimSpace(parentPart)
			if strings.Contains(nestedPart, "&") {
				resolved = append(resolved, strings.ReplaceAll(nestedPart, "&", parentPart))
			} else {
				resolved = append(resolved, parentPart+" "+nestedPart)
			}
		}
	}
	return strings.Join(resolved, ", ")
}

func splitSelectorList(selector string) []string {
	var parts []string
	depth := 0
	current := strings.Builder{}
	for _, char := range selector {
		switch char {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, current.String())
				current.Reset()
				continue
			}
		}
		current.WriteRune(char)
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}
