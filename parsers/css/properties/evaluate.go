package properties

import (
	"strings"

	"gonetsurf/parsers/css/colors"
	"gonetsurf/parsers/css/grammar"
	"gonetsurf/parsers/css/parser"
	"gonetsurf/parsers/css/tokenizer"
	cssvalues "gonetsurf/parsers/css/values"
)

// Context returns a grammar matching context populated with the type matchers
// and property grammars in this registry.
func Context() *grammar.Context {
	context := grammar.NewContext()
	registerTypes(context)
	for name, property := range registry {
		context.Properties[name] = property.Grammar
	}
	return context
}

// Validate reports whether a property value matches the property's grammar.
// The second result is false when the property is not registered.
func Validate(property, value string) (valid bool, known bool) {
	if _, ok := registry[property]; !ok {
		return false, false
	}
	componentValues := parser.ParseComponentValueList(value)
	if isSingleGlobalKeyword(componentValues) {
		return true, true
	}
	match, err := grammar.MatchDefinition(registry[property].Grammar, componentValues, Context())
	if err != nil {
		return false, true
	}
	return match != nil, true
}

func isSingleGlobalKeyword(values []parser.ComponentValue) bool {
	trimmed := []parser.ComponentValue{}
	for _, value := range values {
		if parser.IsWhitespace(value) {
			continue
		}
		trimmed = append(trimmed, value)
	}
	if len(trimmed) != 1 {
		return false
	}
	token, ok := parser.AsToken(trimmed[0])
	if !ok || token.Type != tokenizer.TokenIdent {
		return false
	}
	return globalKeywords[strings.ToLower(token.Value)]
}

func registerTypes(context *grammar.Context) {
	context.Types["number"] = matchNumber
	context.Types["integer"] = matchInteger
	context.Types["percentage"] = matchPercentage
	context.Types["length"] = matchLength
	context.Types["length-percentage"] = matchLengthPercentage
	context.Types["length-nonnegative"] = matchNonNegativeLength
	context.Types["length-percentage-nonnegative"] = matchNonNegativeLengthPercentage
	context.Types["angle"] = matchAngle
	context.Types["time"] = matchTime
	context.Types["time-nonnegative"] = matchNonNegativeTime
	context.Types["fit-content"] = matchFitContent
	context.Types["frequency"] = matchFrequency
	context.Types["resolution"] = matchResolution
	context.Types["color"] = matchColor
	context.Types["custom-ident"] = matchCustomIdent
	context.Types["ident"] = matchIdent
	context.Types["family-name"] = matchFamilyName
	context.Types["generic-family"] = matchGenericFamily
	context.Types["absolute-size"] = matchAbsoluteSize
	context.Types["relative-size"] = matchRelativeSize
	context.Types["font-weight-absolute"] = matchFontWeightAbsolute
	context.Types["line-width"] = matchLineWidth
	context.Types["line-style"] = matchLineStyle
	context.Types["keyframes-name"] = matchKeyframesName
}

func singleToken(values []parser.ComponentValue) (tokenizer.Token, bool) {
	if len(values) == 0 {
		return tokenizer.Token{}, false
	}
	return parser.AsToken(values[0])
}

// dynamicFunctions are valid wherever their category is expected: var() and
// env() substitute any value, and math functions produce a numeric value.
var dynamicFunctions = map[string]bool{
	"var": true, "env": true,
	"calc": true, "min": true, "max": true, "clamp": true, "round": true,
	"mod": true, "rem": true, "abs": true, "sign": true, "hypot": true,
	"pow": true, "sqrt": true, "log": true, "exp": true, "sin": true,
	"cos": true, "tan": true, "asin": true, "acos": true, "atan": true,
	"atan2": true, "calc-size": true, "anchor": true, "anchor-size": true,
}

func isDynamicFunction(value parser.ComponentValue) bool {
	function, ok := value.(*parser.Function)
	if !ok {
		return false
	}
	return dynamicFunctions[strings.ToLower(function.Name)]
}

func matchDynamic(values []parser.ComponentValue) int {
	if len(values) > 0 && isDynamicFunction(values[0]) {
		return 1
	}
	return 0
}

func matchNumber(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	if token, ok := singleToken(values); ok && token.Type == tokenizer.TokenNumber {
		return 1
	}
	return 0
}

func matchInteger(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	if token, ok := singleToken(values); ok && token.Type == tokenizer.TokenNumber && token.NumericKind == tokenizer.Integer {
		return 1
	}
	return 0
}

func matchPercentage(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	if token, ok := singleToken(values); ok && token.Type == tokenizer.TokenPercentage {
		return 1
	}
	return 0
}

func matchLength(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	token, ok := singleToken(values)
	if !ok {
		return 0
	}
	if token.Type == tokenizer.TokenNumber && token.Number == 0 {
		return 1
	}
	if token.Type == tokenizer.TokenDimension {
		if kind, _, valid := cssvalues.ClassifyUnit(token.Value); valid && kind == cssvalues.Length {
			return 1
		}
	}
	return 0
}

func matchLengthPercentage(values []parser.ComponentValue) int {
	if consumed := matchLength(values); consumed > 0 {
		return consumed
	}
	return matchPercentage(values)
}

func matchNonNegativeLength(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	if token, ok := singleToken(values); ok && token.Type == tokenizer.TokenDimension && token.Number >= 0 {
		if kind, _, valid := cssvalues.ClassifyUnit(token.Value); valid && kind == cssvalues.Length {
			return 1
		}
	}
	if token, ok := singleToken(values); ok && token.Type == tokenizer.TokenNumber && token.Number == 0 {
		return 1
	}
	return 0
}

func matchNonNegativeLengthPercentage(values []parser.ComponentValue) int {
	if consumed := matchNonNegativeLength(values); consumed > 0 {
		return consumed
	}
	if token, ok := singleToken(values); ok && token.Type == tokenizer.TokenPercentage && token.Number >= 0 {
		return 1
	}
	return 0
}

func matchAngle(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	if token, ok := singleToken(values); ok && token.Type == tokenizer.TokenDimension {
		if kind, _, valid := cssvalues.ClassifyUnit(token.Value); valid && kind == cssvalues.Angle {
			return 1
		}
	}
	return 0
}

func matchTime(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	if token, ok := singleToken(values); ok && token.Type == tokenizer.TokenDimension {
		if kind, _, valid := cssvalues.ClassifyUnit(token.Value); valid && kind == cssvalues.Time {
			return 1
		}
	}
	return 0
}

func matchNonNegativeTime(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	if token, ok := singleToken(values); ok && token.Type == tokenizer.TokenDimension && token.Number >= 0 {
		if kind, _, valid := cssvalues.ClassifyUnit(token.Value); valid && kind == cssvalues.Time {
			return 1
		}
	}
	return 0
}

// matchFitContent matches the fit-content keyword or the fit-content()
// function.
func matchFitContent(values []parser.ComponentValue) int {
	if token, ok := singleToken(values); ok && token.Type == tokenizer.TokenIdent && strings.EqualFold(token.Value, "fit-content") {
		return 1
	}
	if len(values) > 0 {
		if function, ok := values[0].(*parser.Function); ok && strings.EqualFold(function.Name, "fit-content") {
			return 1
		}
	}
	return 0
}

func matchFrequency(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	if token, ok := singleToken(values); ok && token.Type == tokenizer.TokenDimension {
		if kind, _, valid := cssvalues.ClassifyUnit(token.Value); valid && kind == cssvalues.Frequency {
			return 1
		}
	}
	return 0
}

func matchResolution(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	if token, ok := singleToken(values); ok && token.Type == tokenizer.TokenDimension {
		if kind, _, valid := cssvalues.ClassifyUnit(token.Value); valid && kind == cssvalues.Resolution {
			return 1
		}
	}
	return 0
}

// colorFunctions includes the Color 4 and Color 5 color functions.
var colorFunctions = map[string]bool{
	"rgb": true, "rgba": true, "hsl": true, "hsla": true, "hwb": true,
	"lab": true, "lch": true, "oklab": true, "oklch": true, "color": true,
	"color-mix": true, "light-dark": true, "alpha": true, "device-cmyk": true,
	"contrast-color": true,
}

func matchColor(values []parser.ComponentValue) int {
	if len(values) == 0 {
		return 0
	}
	if _, ok := colors.ParseValues(values[:1]); ok {
		return 1
	}
	if function, ok := values[0].(*parser.Function); ok && colorFunctions[strings.ToLower(function.Name)] {
		return 1
	}
	return 0
}

var globalKeywords = map[string]bool{
	"inherit": true, "initial": true, "unset": true, "revert": true,
	"revert-layer": true, "default": true,
}

func matchIdent(values []parser.ComponentValue) int {
	if token, ok := singleToken(values); ok && token.Type == tokenizer.TokenIdent {
		return 1
	}
	return 0
}

func matchCustomIdent(values []parser.ComponentValue) int {
	token, ok := singleToken(values)
	if !ok || token.Type != tokenizer.TokenIdent {
		return 0
	}
	if globalKeywords[strings.ToLower(token.Value)] {
		return 0
	}
	return 1
}

func matchFamilyName(values []parser.ComponentValue) int {
	if len(values) == 0 {
		return 0
	}
	if token, ok := parser.AsToken(values[0]); ok && token.Type == tokenizer.TokenString {
		return 1
	}
	// An unquoted family name is one or more identifier tokens. CSS-wide
	// keywords may not appear inside a family name.
	consumed := 0
	for consumed < len(values) {
		token, ok := parser.AsToken(values[consumed])
		if !ok || token.Type != tokenizer.TokenIdent {
			break
		}
		if globalKeywords[strings.ToLower(token.Value)] {
			break
		}
		consumed++
	}
	return consumed
}

var genericFamilies = map[string]bool{
	"serif": true, "sans-serif": true, "monospace": true, "cursive": true,
	"fantasy": true, "system-ui": true, "ui-serif": true, "ui-sans-serif": true,
	"ui-monospace": true, "ui-rounded": true, "emoji": true, "math": true,
	"fangsong": true,
}

func matchGenericFamily(values []parser.ComponentValue) int {
	token, ok := singleToken(values)
	if !ok || token.Type != tokenizer.TokenIdent {
		return 0
	}
	if genericFamilies[strings.ToLower(token.Value)] {
		return 1
	}
	return 0
}

var absoluteSizes = map[string]bool{
	"xx-small": true, "x-small": true, "small": true, "medium": true,
	"large": true, "x-large": true, "xx-large": true, "xxx-large": true,
}

func matchAbsoluteSize(values []parser.ComponentValue) int {
	token, ok := singleToken(values)
	if ok && token.Type == tokenizer.TokenIdent && absoluteSizes[strings.ToLower(token.Value)] {
		return 1
	}
	return 0
}

func matchRelativeSize(values []parser.ComponentValue) int {
	token, ok := singleToken(values)
	if ok && token.Type == tokenizer.TokenIdent {
		switch strings.ToLower(token.Value) {
		case "larger", "smaller":
			return 1
		}
	}
	return 0
}

func matchFontWeightAbsolute(values []parser.ComponentValue) int {
	token, ok := singleToken(values)
	if !ok {
		return 0
	}
	if token.Type == tokenizer.TokenIdent {
		switch strings.ToLower(token.Value) {
		case "normal", "bold":
			return 1
		}
		return 0
	}
	if token.Type == tokenizer.TokenNumber && token.NumericKind == tokenizer.Integer && token.Number >= 1 && token.Number <= 1000 {
		return 1
	}
	return 0
}

var lineStyles = map[string]bool{
	"none": true, "hidden": true, "dotted": true, "dashed": true, "solid": true,
	"double": true, "groove": true, "ridge": true, "inset": true, "outset": true,
}

func matchLineStyle(values []parser.ComponentValue) int {
	token, ok := singleToken(values)
	if ok && token.Type == tokenizer.TokenIdent && lineStyles[strings.ToLower(token.Value)] {
		return 1
	}
	return 0
}

func matchLineWidth(values []parser.ComponentValue) int {
	token, ok := singleToken(values)
	if ok && token.Type == tokenizer.TokenIdent {
		switch strings.ToLower(token.Value) {
		case "thin", "medium", "thick":
			return 1
		}
	}
	return matchLength(values)
}

func matchKeyframesName(values []parser.ComponentValue) int {
	token, ok := singleToken(values)
	if !ok {
		return 0
	}
	if token.Type == tokenizer.TokenString {
		if token.Value == "" {
			return 0
		}
		return 1
	}
	if token.Type == tokenizer.TokenIdent && !globalKeywords[strings.ToLower(token.Value)] {
		return 1
	}
	return 0
}
