// Package conditions implements CSS conditional rules: @supports conditions
// and @import descriptors.
//
// See specifications/css/conditional-rules-3.html.
package conditions

import (
	"fmt"
	"strings"

	"gonetsurf/parsers/css/parser"
	"gonetsurf/parsers/css/selectors"
	"gonetsurf/parsers/css/tokenizer"
)

// SupportChecker answers whether a declaration or selector is supported.
type SupportChecker interface {
	SupportsDeclaration(property, value string) bool
	SupportsSelector(selector string) bool
}

// Condition is a parsed @supports condition.
type Condition interface {
	Evaluate(checker SupportChecker) bool
}

// DeclarationCondition tests a declaration form such as (display: grid).
type DeclarationCondition struct {
	Property string
	Value    string
}

// Evaluate reports whether the declaration is supported.
func (condition DeclarationCondition) Evaluate(checker SupportChecker) bool {
	return checker.SupportsDeclaration(condition.Property, condition.Value)
}

// SelectorCondition tests a selector() form.
type SelectorCondition struct {
	Selector string
}

// Evaluate reports whether the selector is supported.
func (condition SelectorCondition) Evaluate(checker SupportChecker) bool {
	return checker.SupportsSelector(condition.Selector)
}

// NotCondition negates a condition.
type NotCondition struct {
	Condition Condition
}

// Evaluate reports the negation of the inner condition.
func (condition NotCondition) Evaluate(checker SupportChecker) bool {
	return !condition.Condition.Evaluate(checker)
}

// AndCondition requires all conditions to hold.
type AndCondition struct {
	Conditions []Condition
}

// Evaluate reports whether all conditions hold.
func (condition AndCondition) Evaluate(checker SupportChecker) bool {
	for _, item := range condition.Conditions {
		if !item.Evaluate(checker) {
			return false
		}
	}
	return true
}

// OrCondition requires at least one condition to hold.
type OrCondition struct {
	Conditions []Condition
}

// Evaluate reports whether any condition holds.
func (condition OrCondition) Evaluate(checker SupportChecker) bool {
	for _, item := range condition.Conditions {
		if item.Evaluate(checker) {
			return true
		}
	}
	return false
}

// ParseSupports parses an @supports condition.
func ParseSupports(input string) (Condition, error) {
	return parseCondition(parser.ParseComponentValueList(input))
}

func parseCondition(values []parser.ComponentValue) (Condition, error) {
	trimmed := trimWhitespace(values)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("css conditions: empty condition")
	}
	// not <condition>
	if token, ok := parser.AsToken(trimmed[0]); ok && token.Type == tokenizer.TokenIdent && strings.EqualFold(token.Value, "not") {
		inner, err := parseCondition(trimmed[1:])
		if err != nil {
			return nil, err
		}
		return NotCondition{Condition: inner}, nil
	}
	// Split on top-level and/or.
	if conditions, operator, ok := splitLogical(trimmed); ok {
		parts := make([]Condition, 0, len(conditions))
		for _, group := range conditions {
			condition, err := parseInParens(group)
			if err != nil {
				return nil, err
			}
			parts = append(parts, condition)
		}
		if operator == "and" {
			return AndCondition{Conditions: parts}, nil
		}
		return OrCondition{Conditions: parts}, nil
	}
	return parseInParens(trimmed)
}

func parseInParens(values []parser.ComponentValue) (Condition, error) {
	trimmed := trimWhitespace(values)
	if len(trimmed) == 1 {
		if block, ok := trimmed[0].(*parser.Block); ok && block.Associated == tokenizer.TokenOpenParen {
			return parseInParensContent(block.Value)
		}
		if function, ok := trimmed[0].(*parser.Function); ok {
			if strings.EqualFold(function.Name, "selector") {
				return SelectorCondition{Selector: strings.TrimSpace(parser.SerializeValues(function.Value))}, nil
			}
			return nil, fmt.Errorf("css conditions: unsupported function %q", function.Name)
		}
	}
	// A declaration or selector without outer parentheses.
	return parseInParensContent(trimmed)
}

func parseInParensContent(values []parser.ComponentValue) (Condition, error) {
	trimmed := trimWhitespace(values)
	colon := findTopLevelColon(trimmed)
	if colon >= 0 {
		nameToken, ok := parser.AsToken(trimmed[0])
		if !ok || nameToken.Type != tokenizer.TokenIdent {
			return nil, fmt.Errorf("css conditions: expected property name")
		}
		return DeclarationCondition{
			Property: strings.ToLower(nameToken.Value),
			Value:    strings.TrimSpace(parser.SerializeValues(trimmed[colon+1:])),
		}, nil
	}
	if list, err := selectors.ParseValues(trimmed); err == nil && list != nil {
		return SelectorCondition{Selector: strings.TrimSpace(parser.SerializeValues(trimmed))}, nil
	}
	return nil, fmt.Errorf("css conditions: invalid condition")
}

func findTopLevelColon(values []parser.ComponentValue) int {
	for index, value := range values {
		if token, ok := parser.AsToken(value); ok && token.Type == tokenizer.TokenColon {
			return index
		}
	}
	return -1
}

func splitLogical(values []parser.ComponentValue) ([][]parser.ComponentValue, string, bool) {
	groups := [][]parser.ComponentValue{}
	current := []parser.ComponentValue{}
	operator := ""
	for _, value := range values {
		if token, ok := parser.AsToken(value); ok && token.Type == tokenizer.TokenIdent {
			lower := strings.ToLower(token.Value)
			if (lower == "and" || lower == "or") && len(current) > 0 {
				if operator == "" {
					operator = lower
				}
				if operator != lower {
					return nil, "", false
				}
				groups = append(groups, current)
				current = nil
				continue
			}
		}
		current = append(current, value)
	}
	if operator == "" {
		return nil, "", false
	}
	groups = append(groups, current)
	return groups, operator, true
}

// Import is a parsed @import rule.
type Import struct {
	URL          string
	Layer        string
	Supports     Condition
	MediaQueries string
}

// ParseImport parses the prelude of an @import rule.
func ParseImport(input string) (*Import, error) {
	values := trimWhitespace(parser.ParseComponentValueList(input))
	if len(values) == 0 {
		return nil, fmt.Errorf("css conditions: empty @import")
	}
	result := &Import{}
	index := 0
	switch typed := values[index].(type) {
	case parser.Token:
		if typed.Type == tokenizer.TokenString || typed.Type == tokenizer.TokenURL {
			result.URL = typed.Value
			index++
		}
	case *parser.Function:
		if strings.EqualFold(typed.Name, "url") {
			result.URL = strings.Trim(strings.TrimSpace(parser.SerializeValues(typed.Value)), "\"'")
			index++
		}
	}
	if result.URL == "" {
		return nil, fmt.Errorf("css conditions: @import requires a URL")
	}
	rest := trimWhitespace(values[index:])
	// layer, supports(), and media queries.
	for len(rest) > 0 {
		value := rest[0]
		if function, ok := value.(*parser.Function); ok && strings.EqualFold(function.Name, "layer") {
			result.Layer = strings.TrimSpace(parser.SerializeValues(function.Value))
			rest = trimWhitespace(rest[1:])
			continue
		}
		if token, ok := parser.AsToken(value); ok && token.Type == tokenizer.TokenIdent && strings.EqualFold(token.Value, "layer") {
			result.Layer = "anonymous"
			rest = trimWhitespace(rest[1:])
			if len(rest) > 0 {
				if next, ok := parser.AsToken(rest[0]); ok && next.Type == tokenizer.TokenIdent {
					result.Layer = next.Value
					rest = trimWhitespace(rest[1:])
				}
			}
			continue
		}
		if function, ok := value.(*parser.Function); ok && strings.EqualFold(function.Name, "supports") {
			condition, err := ParseSupports(parser.SerializeValues(function.Value))
			if err != nil {
				return nil, err
			}
			result.Supports = condition
			rest = trimWhitespace(rest[1:])
			continue
		}
		break
	}
	result.MediaQueries = strings.TrimSpace(parser.SerializeValues(rest))
	return result, nil
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
