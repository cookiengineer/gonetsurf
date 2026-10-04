package mediaqueries

import (
	"fmt"
	"strings"

	"gonetsurf/parsers/css/parser"
	"gonetsurf/parsers/css/tokenizer"
	"gonetsurf/parsers/css/values"
)

// Environment describes the device and user preferences a media query is
// evaluated against.
type Environment struct {
	MediaType            string
	Width                float64
	Height               float64
	Resolution           float64
	PrefersColorScheme   string
	PrefersReducedMotion string
	ForcedColors         bool
}

// DefaultEnvironment returns a reasonable desktop environment.
func DefaultEnvironment() Environment {
	return Environment{
		MediaType:  "screen",
		Width:      1280,
		Height:     720,
		Resolution: 1,
	}
}

// QueryList is a comma-separated list of media queries.
type QueryList struct {
	Queries []Query
}

// Query is a single media query.
type Query struct {
	Not      bool
	Only     bool
	Type     string
	Features []Feature
}

// Feature is a parenthesized media feature.
type Feature struct {
	Name     string
	Operator string
	Value    string
}

// Parse parses a media query list.
func Parse(input string) (*QueryList, error) {
	values := parser.ParseComponentValueList(input)
	groups := splitCommas(values)
	list := &QueryList{}
	for _, group := range groups {
		query, err := parseQuery(group)
		if err != nil {
			return nil, err
		}
		list.Queries = append(list.Queries, *query)
	}
	if len(list.Queries) == 0 {
		return nil, fmt.Errorf("css mediaqueries: empty query list")
	}
	return list, nil
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

func parseQuery(values []parser.ComponentValue) (*Query, error) {
	query := &Query{}
	index := 0
	// Trim leading whitespace.
	for index < len(values) && parser.IsWhitespace(values[index]) {
		index++
	}
	if index < len(values) {
		if token, ok := parser.AsToken(values[index]); ok && token.Type == tokenizer.TokenIdent {
			switch strings.ToLower(token.Value) {
			case "not":
				query.Not = true
				index++
			case "only":
				query.Only = true
				index++
			}
		}
	}
	for index < len(values) && parser.IsWhitespace(values[index]) {
		index++
	}
	if index < len(values) {
		if token, ok := parser.AsToken(values[index]); ok && token.Type == tokenizer.TokenIdent {
			lower := strings.ToLower(token.Value)
			if lower != "and" {
				query.Type = lower
				index++
			}
		}
	}
	for index < len(values) {
		value := values[index]
		if parser.IsWhitespace(value) {
			index++
			continue
		}
		if token, ok := parser.AsToken(value); ok && token.Type == tokenizer.TokenIdent && strings.EqualFold(token.Value, "and") {
			index++
			continue
		}
		block, ok := value.(*parser.Block)
		if !ok || block.Associated != tokenizer.TokenOpenParen {
			return nil, fmt.Errorf("css mediaqueries: expected a feature")
		}
		feature, err := parseFeature(block.Value)
		if err != nil {
			return nil, err
		}
		query.Features = append(query.Features, *feature)
		index++
	}
	if query.Type == "" {
		query.Type = "all"
	}
	return query, nil
}

func parseFeature(values []parser.ComponentValue) (*Feature, error) {
	trimmed := trimWhitespace(values)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("css mediaqueries: empty feature")
	}
	nameToken, ok := parser.AsToken(trimmed[0])
	if !ok || nameToken.Type != tokenizer.TokenIdent {
		return nil, fmt.Errorf("css mediaqueries: expected feature name")
	}
	feature := &Feature{Name: strings.ToLower(nameToken.Value)}
	rest := trimWhitespace(trimmed[1:])
	if len(rest) == 0 {
		return feature, nil
	}
	// Handle "(width >= 600px)" by detecting an operator delimiter.
	if token, isToken := parser.AsToken(rest[0]); isToken && token.Type == tokenizer.TokenDelim {
		operator := token.Value
		rest = rest[1:]
		if len(rest) > 0 {
			if next, isNext := parser.AsToken(rest[0]); isNext && next.Type == tokenizer.TokenDelim && next.Value == "=" {
				operator += "="
				rest = rest[1:]
			}
		}
		feature.Operator = operator
		feature.Value = strings.TrimSpace(parser.SerializeValues(rest))
		return feature, nil
	}
	if token, isToken := parser.AsToken(rest[0]); isToken && token.Type == tokenizer.TokenColon {
		rest = rest[1:]
		feature.Value = strings.TrimSpace(parser.SerializeValues(rest))
		return feature, nil
	}
	feature.Value = strings.TrimSpace(parser.SerializeValues(rest))
	return feature, nil
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

// Matches reports whether any query in the list matches the environment.
func (list *QueryList) Matches(environment Environment) bool {
	for _, query := range list.Queries {
		if query.matches(environment) {
			return true
		}
	}
	return false
}

func (query Query) matches(environment Environment) bool {
	result := query.matchesType(environment) && query.matchesFeatures(environment)
	if query.Not {
		return !result
	}
	return result
}

func (query Query) matchesType(environment Environment) bool {
	if query.Type == "all" || query.Type == "" {
		return true
	}
	if environment.MediaType == "" {
		return query.Type == "screen"
	}
	return query.Type == environment.MediaType
}

func (query Query) matchesFeatures(environment Environment) bool {
	for _, feature := range query.Features {
		if !feature.matches(environment) {
			return false
		}
	}
	return true
}

func (feature Feature) matches(environment Environment) bool {
	name := feature.Name
	switch name {
	case "width", "height":
		dimension := environment.Width
		if name == "height" {
			dimension = environment.Height
		}
		value, ok := parseLength(feature.Value)
		if !ok {
			return false
		}
		return compare(dimension, value, feature.Operator)
	case "min-width":
		value, ok := parseLength(feature.Value)
		return ok && environment.Width >= value
	case "max-width":
		value, ok := parseLength(feature.Value)
		return ok && environment.Width <= value
	case "min-height":
		value, ok := parseLength(feature.Value)
		return ok && environment.Height >= value
	case "max-height":
		value, ok := parseLength(feature.Value)
		return ok && environment.Height <= value
	case "orientation":
		orientation := "landscape"
		if environment.Height > environment.Width {
			orientation = "portrait"
		}
		return strings.EqualFold(feature.Value, orientation)
	case "resolution", "min-resolution", "max-resolution":
		value, ok := parseResolution(feature.Value)
		if !ok {
			return false
		}
		switch name {
		case "min-resolution":
			return environment.Resolution >= value
		case "max-resolution":
			return environment.Resolution <= value
		default:
			return compare(environment.Resolution, value, feature.Operator)
		}
	case "prefers-color-scheme":
		return strings.EqualFold(feature.Value, environment.PrefersColorScheme)
	case "prefers-reduced-motion":
		return strings.EqualFold(feature.Value, environment.PrefersReducedMotion)
	case "forced-colors":
		if strings.EqualFold(feature.Value, "active") {
			return environment.ForcedColors
		}
		return !environment.ForcedColors
	default:
		return false
	}
}

func compare(left, right float64, operator string) bool {
	switch operator {
	case "<":
		return left < right
	case "<=":
		return left <= right
	case ">":
		return left > right
	case ">=":
		return left >= right
	case "=":
		return left == right
	default:
		return left == right
	}
}

func parseLength(text string) (float64, bool) {
	componentValues := parser.ParseComponentValueList(text)
	trimmed := trimWhitespace(componentValues)
	if len(trimmed) != 1 {
		return 0, false
	}
	numeric, ok := values.ParseNumeric(trimmed[0])
	if !ok || numeric.Unit == "%" {
		return 0, false
	}
	if numeric.Kind == values.Length {
		return values.ToPixels(numeric, 0, 0)
	}
	return numeric.Number, true
}

func parseResolution(text string) (float64, bool) {
	componentValues := parser.ParseComponentValueList(text)
	trimmed := trimWhitespace(componentValues)
	if len(trimmed) != 1 {
		return 0, false
	}
	numeric, ok := values.ParseNumeric(trimmed[0])
	if !ok {
		return 0, false
	}
	switch strings.ToLower(numeric.Unit) {
	case "dppx", "x":
		return numeric.Number, true
	case "dpi":
		return numeric.Number / 96, true
	case "dpcm":
		return numeric.Number / (96 * 2.54), true
	default:
		return numeric.Number, numeric.Unit == ""
	}
}
