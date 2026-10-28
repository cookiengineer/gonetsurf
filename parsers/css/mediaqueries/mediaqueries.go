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
	MediaType                  string
	Width                      float64
	Height                     float64
	Resolution                 float64
	Color                      int
	ColorIndex                 int
	Monochrome                 int
	Grid                       bool
	Scan                       string
	Update                     string
	OverflowBlock              string
	OverflowInline             string
	Pointer                    string
	AnyPointer                 string
	Hover                      string
	AnyHover                   string
	PrefersColorScheme         string
	PrefersReducedMotion       string
	PrefersContrast            string
	PrefersReducedData         string
	PrefersReducedTransparency string
	ForcedColors               bool
	InvertedColors             bool
	DisplayMode                string
	DynamicRange               string
	VideoDynamicRange          string
	Scripting                  string
	ColorGamut                 string
	VideoColorGamut            string
	EnvironmentBlending        string
	DevicePosture              string
	NavControls                string
	HorizontalViewportSegments int
	VerticalViewportSegments   int
}

// DefaultEnvironment returns a reasonable desktop environment.
func DefaultEnvironment() Environment {
	return Environment{
		MediaType:                  "screen",
		Width:                      1280,
		Height:                     720,
		Resolution:                 1,
		Color:                      8,
		Grid:                       false,
		Scan:                       "progressive",
		Update:                     "fast",
		OverflowBlock:              "scroll",
		OverflowInline:             "scroll",
		Pointer:                    "fine",
		AnyPointer:                 "fine",
		Hover:                      "hover",
		AnyHover:                   "hover",
		PrefersColorScheme:         "light",
		PrefersReducedMotion:       "no-preference",
		PrefersContrast:            "no-preference",
		PrefersReducedData:         "no-preference",
		PrefersReducedTransparency: "no-preference",
		DisplayMode:                "browser",
		DynamicRange:               "standard",
		VideoDynamicRange:          "standard",
		Scripting:                  "enabled",
		ColorGamut:                 "srgb",
		VideoColorGamut:            "srgb",
		EnvironmentBlending:        "opaque",
		DevicePosture:              "continuous",
		NavControls:                "none",
		HorizontalViewportSegments: 1,
		VerticalViewportSegments:   1,
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
		for _, orGroup := range splitOr(group) {
			query, err := parseQuery(orGroup)
			if err != nil {
				return nil, err
			}
			list.Queries = append(list.Queries, *query)
		}
	}
	if len(list.Queries) == 0 {
		return nil, fmt.Errorf("css mediaqueries: empty query list")
	}
	return list, nil
}

func splitOr(values []parser.ComponentValue) [][]parser.ComponentValue {
	groups := [][]parser.ComponentValue{}
	current := []parser.ComponentValue{}
	for _, value := range values {
		if token, ok := parser.AsToken(value); ok && token.Type == tokenizer.TokenIdent && strings.EqualFold(token.Value, "or") {
			groups = append(groups, current)
			current = nil
			continue
		}
		current = append(current, value)
	}
	groups = append(groups, current)
	return groups
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
		features, err := parseFeature(block.Value)
		if err != nil {
			return nil, err
		}
		query.Features = append(query.Features, features...)
		index++
	}
	if query.Type == "" {
		query.Type = "all"
	}
	return query, nil
}

func parseFeature(values []parser.ComponentValue) ([]Feature, error) {
	trimmed := trimWhitespace(values)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("css mediaqueries: empty feature")
	}
	if token, ok := parser.AsToken(trimmed[0]); ok && token.Type == tokenizer.TokenIdent {
		feature := Feature{Name: strings.ToLower(token.Value)}
		rest := trimWhitespace(trimmed[1:])
		if len(rest) == 0 {
			return []Feature{feature}, nil
		}
		if token, ok := parser.AsToken(rest[0]); ok && token.Type == tokenizer.TokenColon {
			feature.Value = strings.TrimSpace(parser.SerializeValues(rest[1:]))
			return []Feature{feature}, nil
		}
		if operator, length, ok := matchComparisonOperator(rest); ok {
			feature.Operator = operator
			feature.Value = strings.TrimSpace(parser.SerializeValues(rest[length:]))
			return []Feature{feature}, nil
		}
		feature.Value = strings.TrimSpace(parser.SerializeValues(rest))
		return []Feature{feature}, nil
	}
	operatorIndex, operator, operatorLength, ok := findComparisonOperator(trimmed)
	if !ok {
		return nil, fmt.Errorf("css mediaqueries: expected feature name")
	}
	lower := strings.TrimSpace(parser.SerializeValues(trimmed[:operatorIndex]))
	cursor := skipWhitespaceFrom(trimmed, operatorIndex+operatorLength)
	if cursor >= len(trimmed) {
		return nil, fmt.Errorf("css mediaqueries: expected feature name")
	}
	nameToken, ok := parser.AsToken(trimmed[cursor])
	if !ok || nameToken.Type != tokenizer.TokenIdent {
		return nil, fmt.Errorf("css mediaqueries: expected feature name")
	}
	name := strings.ToLower(nameToken.Value)
	features := []Feature{{Name: name, Operator: reverseOperator(operator), Value: lower}}
	cursor = skipWhitespaceFrom(trimmed, cursor+1)
	if cursor < len(trimmed) {
		secondOperator, length, ok := matchComparisonOperator(trimmed[cursor:])
		if !ok {
			return nil, fmt.Errorf("css mediaqueries: expected range operator")
		}
		cursor = skipWhitespaceFrom(trimmed, cursor+length)
		upper := strings.TrimSpace(parser.SerializeValues(trimmed[cursor:]))
		features = append(features, Feature{Name: name, Operator: secondOperator, Value: upper})
	}
	return features, nil
}

func findComparisonOperator(values []parser.ComponentValue) (int, string, int, bool) {
	for index := 0; index < len(values); index++ {
		if parser.IsWhitespace(values[index]) {
			continue
		}
		operator, length, ok := matchComparisonOperator(values[index:])
		if ok {
			return index, operator, length, true
		}
	}
	return 0, "", 0, false
}

func matchComparisonOperator(values []parser.ComponentValue) (string, int, bool) {
	if len(values) == 0 {
		return "", 0, false
	}
	token, ok := parser.AsToken(values[0])
	if !ok || token.Type != tokenizer.TokenDelim {
		return "", 0, false
	}
	switch token.Value {
	case "=":
		return "=", 1, true
	case "<", ">":
		operator := token.Value
		if len(values) > 1 {
			if next, ok := parser.AsToken(values[1]); ok && next.Type == tokenizer.TokenDelim && next.Value == "=" {
				return operator + "=", 2, true
			}
		}
		return operator, 1, true
	default:
		return "", 0, false
	}
}

func reverseOperator(operator string) string {
	switch operator {
	case "<":
		return ">"
	case "<=":
		return ">="
	case ">":
		return "<"
	case ">=":
		return "<="
	default:
		return operator
	}
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
	operator := feature.Operator
	if strings.HasPrefix(name, "min-") {
		operator = ">="
		name = strings.TrimPrefix(name, "min-")
	} else if strings.HasPrefix(name, "max-") {
		operator = "<="
		name = strings.TrimPrefix(name, "max-")
	}
	switch name {
	case "width":
		return matchLengthValue(environment.Width, feature.Value, operator)
	case "height":
		return matchLengthValue(environment.Height, feature.Value, operator)
	case "device-width":
		return matchLengthValue(environment.Width, feature.Value, operator)
	case "device-height":
		return matchLengthValue(environment.Height, feature.Value, operator)
	case "resolution":
		return matchResolution(environment.Resolution, feature.Value, operator)
	case "color":
		return matchRange(float64(environment.Color), feature.Value, operator)
	case "color-index":
		return matchRange(float64(environment.ColorIndex), feature.Value, operator)
	case "monochrome":
		return matchRange(float64(environment.Monochrome), feature.Value, operator)
	case "grid":
		return matchRange(boolNumber(environment.Grid), feature.Value, operator)
	case "aspect-ratio":
		return matchRatio(environment.Width, environment.Height, feature.Value, operator)
	case "device-aspect-ratio":
		return matchRatio(environment.Width, environment.Height, feature.Value, operator)
	case "orientation":
		if feature.Value == "" {
			return true
		}
		orientation := "landscape"
		if environment.Height > environment.Width {
			orientation = "portrait"
		}
		return matchKeyword(orientation, feature.Value)
	case "scan":
		return matchKeyword(environment.Scan, feature.Value)
	case "update":
		return matchKeyword(environment.Update, feature.Value)
	case "overflow-block":
		return matchKeyword(environment.OverflowBlock, feature.Value)
	case "overflow-inline":
		return matchKeyword(environment.OverflowInline, feature.Value)
	case "pointer":
		return matchKeyword(environment.Pointer, feature.Value)
	case "any-pointer":
		return matchKeyword(environment.AnyPointer, feature.Value)
	case "hover":
		return matchKeyword(environment.Hover, feature.Value)
	case "any-hover":
		return matchKeyword(environment.AnyHover, feature.Value)
	case "prefers-color-scheme":
		return matchKeyword(environment.PrefersColorScheme, feature.Value)
	case "prefers-reduced-motion":
		return matchKeyword(environment.PrefersReducedMotion, feature.Value)
	case "prefers-contrast":
		return matchKeyword(environment.PrefersContrast, feature.Value)
	case "prefers-reduced-data":
		return matchKeyword(environment.PrefersReducedData, feature.Value)
	case "prefers-reduced-transparency":
		return matchKeyword(environment.PrefersReducedTransparency, feature.Value)
	case "forced-colors":
		if feature.Value == "" {
			return environment.ForcedColors
		}
		return matchKeyword(boolKeyword(environment.ForcedColors), feature.Value)
	case "inverted-colors":
		if feature.Value == "" {
			return environment.InvertedColors
		}
		return matchKeyword(invertedKeyword(environment.InvertedColors), feature.Value)
	case "display-mode":
		return matchKeyword(environment.DisplayMode, feature.Value)
	case "dynamic-range":
		return matchKeyword(environment.DynamicRange, feature.Value)
	case "video-dynamic-range":
		return matchKeyword(environment.VideoDynamicRange, feature.Value)
	case "scripting":
		return matchKeyword(environment.Scripting, feature.Value)
	case "color-gamut":
		return matchKeyword(environment.ColorGamut, feature.Value)
	case "video-color-gamut":
		return matchKeyword(environment.VideoColorGamut, feature.Value)
	case "environment-blending":
		return matchKeyword(environment.EnvironmentBlending, feature.Value)
	case "device-posture":
		return matchKeyword(environment.DevicePosture, feature.Value)
	case "nav-controls":
		return matchKeyword(environment.NavControls, feature.Value)
	case "horizontal-viewport-segments":
		return matchRange(float64(environment.HorizontalViewportSegments), feature.Value, operator)
	case "vertical-viewport-segments":
		return matchRange(float64(environment.VerticalViewportSegments), feature.Value, operator)
	default:
		return false
	}
}

func matchLengthValue(value float64, text, operator string) bool {
	if text == "" {
		return value != 0
	}
	target, ok := parseLength(text)
	if !ok {
		return false
	}
	return compare(value, target, operator)
}

func matchRange(value float64, text, operator string) bool {
	if text == "" {
		return value != 0
	}
	target, ok := parseNumber(text)
	if !ok {
		return false
	}
	return compare(value, target, operator)
}

func matchResolution(value float64, text, operator string) bool {
	if text == "" {
		return value != 0
	}
	target, ok := parseResolution(text)
	if !ok {
		return false
	}
	return compare(value, target, operator)
}

func matchRatio(width, height float64, text, operator string) bool {
	if text == "" {
		return width != 0 && height != 0
	}
	target, ok := parseRatio(text)
	if !ok || height == 0 {
		return false
	}
	return compare(width/height, target, operator)
}

func matchKeyword(environment, text string) bool {
	if text == "" {
		lower := strings.ToLower(environment)
		return lower != "" && lower != "none" && lower != "no-preference"
	}
	return strings.EqualFold(environment, text)
}

func parseNumber(text string) (float64, bool) {
	componentValues := parser.ParseComponentValueList(text)
	trimmed := trimWhitespace(componentValues)
	if len(trimmed) != 1 {
		return 0, false
	}
	token, ok := parser.AsToken(trimmed[0])
	if !ok || token.Type != tokenizer.TokenNumber {
		return 0, false
	}
	return token.Number, true
}

func parseRatio(text string) (float64, bool) {
	parts := strings.Split(text, "/")
	if len(parts) == 1 {
		return parseNumber(strings.TrimSpace(parts[0]))
	}
	if len(parts) != 2 {
		return 0, false
	}
	numerator, ok := parseNumber(strings.TrimSpace(parts[0]))
	if !ok {
		return 0, false
	}
	denominator, ok := parseNumber(strings.TrimSpace(parts[1]))
	if !ok || denominator == 0 {
		return 0, false
	}
	return numerator / denominator, true
}

func boolNumber(value bool) float64 {
	if value {
		return 1
	}
	return 0
}

func boolKeyword(value bool) string {
	if value {
		return "active"
	}
	return "none"
}

func invertedKeyword(value bool) string {
	if value {
		return "inverted"
	}
	return "none"
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
