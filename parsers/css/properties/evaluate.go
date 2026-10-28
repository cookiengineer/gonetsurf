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
	context.Types["number-nonnegative"] = matchNonNegativeNumber
	context.Types["integer"] = matchInteger
	context.Types["percentage"] = matchPercentage
	context.Types["percentage-nonnegative"] = matchNonNegativePercentage
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
	context.Types["identifier"] = matchIdent
	context.Types["string"] = matchString
	context.Types["uri"] = matchURI
	context.Types["image"] = matchImage
	context.Types["bg-image"] = matchImage
	context.Types["attachment"] = keywordMatcher("scroll", "fixed", "local")
	context.Types["box-size"] = matchBoxSize
	context.Types["repeat-style"] = matchRepeatStyle
	context.Types["visual-box"] = keywordMatcher("content-box", "padding-box", "border-box", "margin-box", "fill-box", "stroke-box", "view-box")
	context.Types["easing-function"] = matchEasingFunction
	context.Types["timing-function"] = matchEasingFunction
	context.Types["single-animation-direction"] = keywordMatcher("normal", "reverse", "alternate", "alternate-reverse")
	context.Types["single-animation-fill-mode"] = keywordMatcher("none", "forwards", "backwards", "both")
	context.Types["single-animation-play-state"] = keywordMatcher("running", "paused")
	context.Types["single-animation-iteration-count"] = matchAnimationIterationCount
	context.Types["single-transition-property"] = matchTransitionProperty
	context.Types["single-animation"] = matchGroupUntilComma
	context.Types["single-transition"] = matchGroupUntilComma
	context.Types["bg-layer"] = matchGroupUntilComma
	context.Types["final-bg-layer"] = matchGroupUntilComma
	context.Types["bg-position"] = matchBackgroundPosition
	context.Types["bg-size"] = matchBackgroundSize
	context.Types["shadow"] = matchShadow
	context.Types["transform-list"] = matchGroupUntilComma
	context.Types["track-list"] = matchGroupUntilComma
	context.Types["auto-track-list"] = matchGroupUntilComma
	context.Types["explicit-track-list"] = matchGroupUntilComma
	context.Types["feature-tag-value"] = matchFeatureTagValue
	context.Types["font-feature-value-name"] = matchStringOrIdent
	context.Types["opentype-tag"] = matchOpenTypeTag
	context.Types["palette-identifier"] = matchPaletteIdentifier
	context.Types["palette-mix"] = matchAnySingle
	context.Types["font-family-name"] = matchFamilyName
	context.Types["generic-font-family"] = matchGenericFamily
	context.Types["system-font-family-name"] = keywordMatcher("caption", "icon", "menu", "message-box", "small-caption", "status-bar")
	context.Types["font-variant-css2"] = keywordMatcher("normal", "small-caps")
	context.Types["font-width-css3"] = keywordMatcher("normal", "condensed", "expanded", "semi-condensed", "semi-expanded", "extra-condensed", "extra-expanded", "ultra-condensed", "ultra-expanded")
	context.Types["common-lig-values"] = keywordMatcher("common-ligatures", "no-common-ligatures")
	context.Types["discretionary-lig-values"] = keywordMatcher("discretionary-ligatures", "no-discretionary-ligatures")
	context.Types["historical-lig-values"] = keywordMatcher("historical-ligatures", "no-historical-ligatures")
	context.Types["contextual-alt-values"] = keywordMatcher("contextual", "no-contextual")
	context.Types["numeric-figure-values"] = keywordMatcher("lining-nums", "oldstyle-nums")
	context.Types["numeric-spacing-values"] = keywordMatcher("proportional-nums", "tabular-nums")
	context.Types["numeric-fraction-values"] = keywordMatcher("diagonal-fractions", "stacked-fractions")
	context.Types["east-asian-variant-values"] = keywordMatcher("jis78", "jis83", "jis90", "jis04", "simplified", "traditional")
	context.Types["east-asian-width-values"] = keywordMatcher("full-width", "proportional-width")
	context.Types["opacity-value"] = matchOpacityValue
	context.Types["margin-width"] = matchMarginWidth
	context.Types["ratio"] = matchRatio
	context.Types["line-names"] = matchLineNames
	context.Types["line-name-list"] = matchGroupUntilComma
	context.Types["grid-line"] = matchGroupUntilComma
	context.Types["text-wrap-mode"] = keywordMatcher("wrap", "nowrap")
	context.Types["text-wrap-style"] = keywordMatcher("auto", "balance", "stable", "pretty")
	context.Types["integer-nonnegative"] = matchNonNegativeInteger
	context.Types["line-width-nonnegative"] = matchLineWidthNonNegative
	context.Types["language-override"] = matchLanguageOverride
	context.Types["font-variant-value"] = matchFontVariantValue
	context.Types["font-variant-alternates-value"] = matchFontVariantAlternatesValue
	context.Types["border-image-value"] = matchGroupUntilComma
	context.Types["display-outside"] = keywordMatcher("block", "inline", "run-in")
	context.Types["display-inside"] = keywordMatcher("flow", "flow-root", "table", "flex", "grid", "ruby")
	context.Types["display-listitem"] = matchDisplayListitem
	context.Types["display-internal"] = keywordMatcher("table-row-group", "table-header-group", "table-footer-group", "table-row", "table-column-group", "table-column", "table-cell", "table-caption", "ruby-base", "ruby-text", "ruby-base-container", "ruby-text-container")
	context.Types["display-box"] = keywordMatcher("contents", "none")
	context.Types["display-legacy"] = keywordMatcher("inline-block", "inline-table", "inline-flex", "inline-grid")
	context.Types["font-synthesis-value"] = matchFontSynthesis
	context.Types["font-variant-list"] = matchFontVariantList
	context.Types["font-variant-alternates-list"] = matchFontVariantAlternatesList
	context.Types["background-clip-value"] = matchBackgroundClip
	context.Types["variation-tag-value"] = matchVariationTagValue
	context.Types["text-shadow-value"] = matchTextShadow
	context.Types["text-underline-value"] = matchTextUnderlinePosition
	context.Types["tab-size-value"] = matchTabSizeValue
	context.Types["number-nonnegative"] = matchNonNegativeNumber
	context.Types["percentage-nonnegative"] = matchNonNegativePercentage
}

func keywordMatcher(keywords ...string) grammar.TypeMatcher {
	set := map[string]bool{}
	for _, keyword := range keywords {
		set[keyword] = true
	}
	return func(values []parser.ComponentValue) int {
		if consumed := matchDynamic(values); consumed > 0 {
			return consumed
		}
		token, ok := singleToken(values)
		if ok && token.Type == tokenizer.TokenIdent && set[strings.ToLower(token.Value)] {
			return 1
		}
		return 0
	}
}

func matchString(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	token, ok := singleToken(values)
	if ok && token.Type == tokenizer.TokenString {
		return 1
	}
	return 0
}

func matchURI(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	token, ok := singleToken(values)
	if ok && token.Type == tokenizer.TokenURL {
		return 1
	}
	if function, ok := firstFunction(values); ok && strings.EqualFold(function.Name, "url") {
		return 1
	}
	return 0
}

func matchImage(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	token, ok := singleToken(values)
	if ok {
		if token.Type == tokenizer.TokenURL {
			return 1
		}
		if token.Type == tokenizer.TokenIdent && strings.EqualFold(token.Value, "none") {
			return 1
		}
	}
	function, ok := firstFunction(values)
	if !ok {
		return 0
	}
	switch strings.ToLower(function.Name) {
	case "url", "image", "image-set", "cross-fade", "element", "paint", "light-dark",
		"linear-gradient", "radial-gradient", "conic-gradient",
		"repeating-linear-gradient", "repeating-radial-gradient", "repeating-conic-gradient":
		return 1
	default:
		return 0
	}
}

func matchAnySingle(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	if len(values) == 0 || parser.IsWhitespace(values[0]) {
		return 0
	}
	return 1
}

func matchGroupUntilComma(values []parser.ComponentValue) int {
	if len(values) == 0 {
		return 0
	}
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	consumed := 0
	hasValue := false
	for _, value := range values {
		if token, ok := parser.AsToken(value); ok && token.Type == tokenizer.TokenComma {
			break
		}
		consumed++
		if !parser.IsWhitespace(value) {
			hasValue = true
		}
	}
	if !hasValue {
		return 0
	}
	return consumed
}

func matchStringOrIdent(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	token, ok := singleToken(values)
	if !ok {
		return 0
	}
	if token.Type == tokenizer.TokenString {
		return 1
	}
	if token.Type == tokenizer.TokenIdent {
		return 1
	}
	return 0
}

func matchAnimationIterationCount(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	token, ok := singleToken(values)
	if !ok {
		return 0
	}
	if token.Type == tokenizer.TokenIdent && strings.EqualFold(token.Value, "infinite") {
		return 1
	}
	if token.Type == tokenizer.TokenNumber && token.Number >= 0 {
		return 1
	}
	return 0
}

func matchTransitionProperty(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	token, ok := singleToken(values)
	if !ok {
		return 0
	}
	if token.Type != tokenizer.TokenIdent {
		return 0
	}
	lower := strings.ToLower(token.Value)
	if lower == "all" || lower == "none" {
		return 1
	}
	if globalKeywords[lower] {
		return 0
	}
	return 1
}

func matchOpacityValue(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	token, ok := singleToken(values)
	if !ok {
		return 0
	}
	if token.Type == tokenizer.TokenNumber {
		return 1
	}
	if token.Type == tokenizer.TokenPercentage {
		return 1
	}
	return 0
}

func matchMarginWidth(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	token, ok := singleToken(values)
	if ok && token.Type == tokenizer.TokenIdent && strings.EqualFold(token.Value, "auto") {
		return 1
	}
	return matchLengthPercentage(values)
}

func matchRatio(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	token, ok := singleToken(values)
	if !ok || token.Type != tokenizer.TokenNumber || token.Number < 0 {
		return 0
	}
	index := 1
	for index < len(values) && parser.IsWhitespace(values[index]) {
		index++
	}
	if index >= len(values) {
		return 1
	}
	slash, ok := parser.AsToken(values[index])
	if !ok || slash.Type != tokenizer.TokenDelim || slash.Value != "/" {
		return 1
	}
	index++
	for index < len(values) && parser.IsWhitespace(values[index]) {
		index++
	}
	if index >= len(values) {
		return 0
	}
	second, ok := parser.AsToken(values[index])
	if !ok || second.Type != tokenizer.TokenNumber || second.Number < 0 {
		return 0
	}
	return index + 1
}

func matchLineNames(values []parser.ComponentValue) int {
	if len(values) == 0 {
		return 0
	}
	block, ok := parser.IsBlock(values[0], tokenizer.TokenOpenSquare)
	if !ok || block == nil {
		return 0
	}
	return 1
}

func matchNonNegativeInteger(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	token, ok := singleToken(values)
	if ok && token.Type == tokenizer.TokenNumber && token.NumericKind == tokenizer.Integer && token.Number >= 0 {
		return 1
	}
	return 0
}

func firstFunction(values []parser.ComponentValue) (*parser.Function, bool) {
	if len(values) == 0 {
		return nil, false
	}
	function, ok := values[0].(*parser.Function)
	return function, ok
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
	"var": true, "env": true, "attr": true,
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

func matchNonNegativeNumber(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	if token, ok := singleToken(values); ok && token.Type == tokenizer.TokenNumber && token.Number >= 0 {
		return 1
	}
	return 0
}

func matchNonNegativePercentage(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	if token, ok := singleToken(values); ok && token.Type == tokenizer.TokenPercentage && token.Number >= 0 {
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
	"revert-layer": true,
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
		if globalKeywords[strings.ToLower(token.Value)] || strings.EqualFold(token.Value, "default") {
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
	if token.Type == tokenizer.TokenIdent && !globalKeywords[strings.ToLower(token.Value)] && !strings.EqualFold(token.Value, "default") {
		return 1
	}
	return 0
}

func matchBoxSize(values []parser.ComponentValue) int {
	if consumed := matchNonNegativeLengthPercentage(values); consumed > 0 {
		return consumed
	}
	token, ok := singleToken(values)
	if ok && token.Type == tokenizer.TokenIdent {
		switch strings.ToLower(token.Value) {
		case "min-content", "max-content", "fit-content":
			return 1
		}
	}
	if function, ok := firstFunction(values); ok && strings.EqualFold(function.Name, "fit-content") {
		return 1
	}
	return 0
}

var repeatKeywords = map[string]bool{
	"repeat-x": true, "repeat-y": true, "repeat": true,
	"space": true, "round": true, "no-repeat": true,
}

func matchRepeatStyle(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	index := 0
	count := 0
	for count < 2 {
		for index < len(values) && parser.IsWhitespace(values[index]) {
			index++
		}
		if index >= len(values) {
			break
		}
		token, ok := parser.AsToken(values[index])
		if !ok || token.Type != tokenizer.TokenIdent || !repeatKeywords[strings.ToLower(token.Value)] {
			break
		}
		lower := strings.ToLower(token.Value)
		if lower == "repeat-x" || lower == "repeat-y" {
			if count != 0 {
				return 0
			}
			return index + 1
		}
		index++
		count++
	}
	if count == 0 {
		return 0
	}
	return index
}

func matchBackgroundSize(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	index := 0
	count := 0
	for count < 2 {
		for index < len(values) && parser.IsWhitespace(values[index]) {
			index++
		}
		if index >= len(values) {
			break
		}
		token, ok := parser.AsToken(values[index])
		if ok && token.Type == tokenizer.TokenIdent {
			lower := strings.ToLower(token.Value)
			if lower == "cover" || lower == "contain" {
				if count > 0 {
					break
				}
				return index + 1
			}
			if lower == "auto" {
				index++
				count++
				continue
			}
		}
		consumed := matchNonNegativeLengthPercentage(values[index:])
		if consumed <= 0 {
			break
		}
		index += consumed
		count++
	}
	if count == 0 {
		return 0
	}
	return index
}

type positionItem struct {
	isKeyword    bool
	isHorizontal bool
	isVertical   bool
}

func matchBackgroundPosition(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	items := []positionItem{}
	consumed := 0
	for _, value := range values {
		if token, ok := parser.AsToken(value); ok && token.Type == tokenizer.TokenComma {
			break
		}
		consumed++
		if parser.IsWhitespace(value) {
			continue
		}
		token, ok := parser.AsToken(value)
		if !ok {
			if function, ok := value.(*parser.Function); ok && isDynamicMathName(function.Name) {
				items = append(items, positionItem{})
				continue
			}
			return 0
		}
		switch token.Type {
		case tokenizer.TokenIdent:
			switch strings.ToLower(token.Value) {
			case "left", "right":
				items = append(items, positionItem{isKeyword: true, isHorizontal: true})
			case "top", "bottom":
				items = append(items, positionItem{isKeyword: true, isVertical: true})
			case "center":
				items = append(items, positionItem{isKeyword: true})
			default:
				return 0
			}
		case tokenizer.TokenNumber:
			if token.Number != 0 {
				return 0
			}
			items = append(items, positionItem{})
		case tokenizer.TokenPercentage:
			items = append(items, positionItem{})
		case tokenizer.TokenDimension:
			if kind, _, ok := cssvalues.ClassifyUnit(token.Value); !ok || kind != cssvalues.Length {
				return 0
			}
			items = append(items, positionItem{})
		default:
			return 0
		}
	}
	if !validPositionItems(items) {
		return 0
	}
	return consumed
}

func validPositionItems(items []positionItem) bool {
	if len(items) == 0 || len(items) > 4 {
		return false
	}
	keywords := 0
	horizontal := 0
	vertical := 0
	for index, item := range items {
		if item.isKeyword {
			keywords++
			if item.isHorizontal {
				horizontal++
			}
			if item.isVertical {
				vertical++
			}
			continue
		}
		if len(items) >= 3 && (index == 0 || !items[index-1].isKeyword) {
			return false
		}
	}
	if len(items) == 2 {
		if keywords == 2 && (horizontal > 1 || vertical > 1) {
			return false
		}
		return true
	}
	if len(items) >= 3 {
		if keywords == 0 || keywords > 2 {
			return false
		}
		if horizontal > 1 || vertical > 1 {
			return false
		}
	}
	return true
}

func matchOpenTypeTag(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	token, ok := singleToken(values)
	if ok && token.Type == tokenizer.TokenString && len(token.Value) == 4 {
		return 1
	}
	return 0
}

func matchLanguageOverride(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	token, ok := singleToken(values)
	if !ok || token.Type != tokenizer.TokenString {
		return 0
	}
	if len(token.Value) < 1 || len(token.Value) > 4 {
		return 0
	}
	for _, character := range token.Value {
		if character > 0x7f {
			return 0
		}
	}
	return 1
}

func matchFeatureTagValue(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	if len(values) == 0 {
		return 0
	}
	token, ok := parser.AsToken(values[0])
	if !ok || token.Type != tokenizer.TokenString || len(token.Value) != 4 {
		return 0
	}
	index := 1
	for index < len(values) && parser.IsWhitespace(values[index]) {
		index++
	}
	if index >= len(values) {
		return 1
	}
	next, ok := parser.AsToken(values[index])
	if !ok {
		return 1
	}
	if next.Type == tokenizer.TokenNumber && next.NumericKind == tokenizer.Integer {
		return index + 1
	}
	if next.Type == tokenizer.TokenIdent {
		lower := strings.ToLower(next.Value)
		if lower == "on" || lower == "off" {
			return index + 1
		}
	}
	return 1
}

func isDynamicMathName(name string) bool {
	switch strings.ToLower(name) {
	case "var", "env", "attr", "calc", "min", "max", "clamp", "round", "mod", "rem",
		"abs", "sign", "hypot", "pow", "sqrt", "log", "exp", "sin", "cos", "tan",
		"asin", "acos", "atan", "atan2":
		return true
	default:
		return false
	}
}

func matchFontVariantValue(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	if len(values) == 0 || parser.IsWhitespace(values[0]) {
		return 0
	}
	if function, ok := values[0].(*parser.Function); ok {
		switch strings.ToLower(function.Name) {
		case "stylistic", "styleset", "character-variant", "swash", "ornaments", "annotation":
			return 1
		default:
			return 0
		}
	}
	token, ok := parser.AsToken(values[0])
	if !ok || token.Type != tokenizer.TokenIdent {
		return 0
	}
	return 1
}

func matchFontVariantAlternatesValue(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	if len(values) == 0 || parser.IsWhitespace(values[0]) {
		return 0
	}
	if function, ok := values[0].(*parser.Function); ok {
		switch strings.ToLower(function.Name) {
		case "stylistic", "styleset", "character-variant", "swash", "ornaments", "annotation":
			return 1
		default:
			return 0
		}
	}
	token, ok := parser.AsToken(values[0])
	if ok && token.Type == tokenizer.TokenIdent && strings.EqualFold(token.Value, "historical-forms") {
		return 1
	}
	return 0
}

func matchLineWidthNonNegative(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	token, ok := singleToken(values)
	if ok && token.Type == tokenizer.TokenIdent {
		switch strings.ToLower(token.Value) {
		case "thin", "medium", "thick":
			return 1
		}
	}
	return matchNonNegativeLength(values)
}

func matchEasingFunction(values []parser.ComponentValue) int {
	if consumed := keywordMatcher("linear", "ease", "ease-in", "ease-out", "ease-in-out", "step-start", "step-end")(values); consumed > 0 {
		return consumed
	}
	function, ok := firstFunction(values)
	if !ok {
		return 0
	}
	switch strings.ToLower(function.Name) {
	case "cubic-bezier":
		arguments := splitCommaValues(function.Value)
		if len(arguments) != 4 {
			return 0
		}
		for index, argument := range arguments {
			token, ok := singleNonWhitespaceToken(argument)
			if !ok || token.Type != tokenizer.TokenNumber {
				return 0
			}
			if (index == 0 || index == 2) && (token.Number < 0 || token.Number > 1) {
				return 0
			}
		}
		return 1
	case "steps":
		arguments := splitCommaValues(function.Value)
		if len(arguments) != 1 && len(arguments) != 2 {
			return 0
		}
		first, ok := singleNonWhitespaceToken(arguments[0])
		if !ok {
			if _, isFunction := firstFunction(arguments[0]); !isFunction {
				return 0
			}
		} else if first.Type != tokenizer.TokenNumber || first.NumericKind != tokenizer.Integer || first.Number < 1 {
			return 0
		}
		if len(arguments) == 2 {
			position, ok := singleNonWhitespaceToken(arguments[1])
			if !ok || position.Type != tokenizer.TokenIdent {
				return 0
			}
			switch strings.ToLower(position.Value) {
			case "start", "end", "jump-start", "jump-end", "jump-none", "jump-both":
				if strings.EqualFold(position.Value, "jump-none") && ok && first.Type == tokenizer.TokenNumber && first.Number < 2 {
					return 0
				}
			default:
				return 0
			}
		}
		return 1
	case "linear":
		return 1
	default:
		return 0
	}
}

func splitCommaValues(values []parser.ComponentValue) [][]parser.ComponentValue {
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

func matchDisplayListitem(values []parser.ComponentValue) int {
	index := 0
	seenOutside := false
	seenInside := false
	seenList := false
	for index < len(values) {
		for index < len(values) && parser.IsWhitespace(values[index]) {
			index++
		}
		if index >= len(values) {
			break
		}
		token, ok := parser.AsToken(values[index])
		if !ok || token.Type != tokenizer.TokenIdent {
			break
		}
		switch strings.ToLower(token.Value) {
		case "block", "inline", "run-in":
			if seenOutside {
				return 0
			}
			seenOutside = true
		case "flow", "flow-root":
			if seenInside {
				return 0
			}
			seenInside = true
		case "list-item":
			if seenList {
				return 0
			}
			seenList = true
		default:
			index--
			goto done
		}
		index++
	}
done:
	if !seenList {
		return 0
	}
	for index > 0 && parser.IsWhitespace(values[index-1]) {
		index--
	}
	return index
}

func singleNonWhitespaceToken(values []parser.ComponentValue) (tokenizer.Token, bool) {
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
	return parser.AsToken(trimmed[0])
}

func matchShadow(values []parser.ComponentValue) int {
	return matchShadowValues(values, 4, true)
}

func matchTextShadow(values []parser.ComponentValue) int {
	return matchShadowValues(values, 3, false)
}

func matchShadowValues(values []parser.ComponentValue, maxLengths int, allowInset bool) int {
	if len(values) > 0 {
		if function, ok := values[0].(*parser.Function); ok {
			switch strings.ToLower(function.Name) {
			case "var", "env", "attr":
				return 1
			}
		}
	}
	if token, ok := singleToken(values); ok && token.Type == tokenizer.TokenIdent && strings.EqualFold(token.Value, "none") {
		return 1
	}
	consumed := 0
	lengths := 0
	hasInset := false
	hasColor := false
	phase := 0
	for _, value := range values {
		if token, ok := parser.AsToken(value); ok && token.Type == tokenizer.TokenComma {
			break
		}
		consumed++
		if parser.IsWhitespace(value) {
			continue
		}
		if matchColor([]parser.ComponentValue{value}) > 0 {
			if hasColor {
				return 0
			}
			if phase == 1 {
				phase = 2
			}
			hasColor = true
			continue
		}
		token, ok := parser.AsToken(value)
		if ok {
			switch token.Type {
			case tokenizer.TokenIdent:
				if allowInset && strings.EqualFold(token.Value, "inset") && !hasInset {
					if phase == 1 {
						phase = 2
					}
					hasInset = true
					continue
				}
			case tokenizer.TokenNumber:
				if token.Number == 0 {
					if phase == 2 {
						return 0
					}
					phase = 1
					lengths++
					continue
				}
			case tokenizer.TokenDimension:
				if kind, _, valid := cssvalues.ClassifyUnit(token.Value); valid && kind == cssvalues.Length {
					if phase == 2 {
						return 0
					}
					if lengths == 2 && token.Number < 0 {
						return 0
					}
					phase = 1
					lengths++
					continue
				}
			}
			return 0
		}
		function, ok := value.(*parser.Function)
		if !ok {
			return 0
		}
		if expression, valid := cssvalues.ParseMath(function); valid && !mathHasPercentage(expression) {
			if phase == 2 {
				return 0
			}
			lengths++
			phase = 1
			continue
		}
		return 0
	}
	if lengths < 2 || lengths > maxLengths {
		return 0
	}
	return consumed
}

func matchFontSynthesis(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	seen := map[string]bool{}
	consumed := 0
	for _, value := range values {
		if token, ok := parser.AsToken(value); ok && token.Type == tokenizer.TokenComma {
			break
		}
		consumed++
		if parser.IsWhitespace(value) {
			continue
		}
		token, ok := parser.AsToken(value)
		if !ok || token.Type != tokenizer.TokenIdent {
			return 0
		}
		lower := strings.ToLower(token.Value)
		if lower == "none" {
			if countNonWhitespace(values) != 1 {
				return 0
			}
			return consumed
		}
		switch lower {
		case "weight", "style", "small-caps", "position", "oblique-only":
			if seen[lower] {
				return 0
			}
			seen[lower] = true
		default:
			return 0
		}
	}
	if seen["style"] && seen["oblique-only"] {
		return 0
	}
	if len(seen) == 0 {
		return 0
	}
	return consumed
}

func matchFontVariantList(values []parser.ComponentValue) int {
	return matchFontVariantValues(values, false)
}

func matchFontVariantAlternatesList(values []parser.ComponentValue) int {
	return matchFontVariantValues(values, true)
}

func matchFontVariantValues(values []parser.ComponentValue, alternatesOnly bool) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	seen := map[string]bool{}
	alternates := 0
	alternateNames := map[string]bool{}
	consumed := 0
	for _, value := range values {
		if token, ok := parser.AsToken(value); ok && token.Type == tokenizer.TokenComma {
			break
		}
		consumed++
		if parser.IsWhitespace(value) {
			continue
		}
		if function, ok := value.(*parser.Function); ok {
			if !validFontVariantAlternatesFunction(function) {
				return 0
			}
			name := strings.ToLower(function.Name)
			if alternateNames[name] {
				return 0
			}
			alternateNames[name] = true
			alternates++
			continue
		}
		token, ok := parser.AsToken(value)
		if !ok || token.Type != tokenizer.TokenIdent {
			return 0
		}
		lower := strings.ToLower(token.Value)
		if lower == "normal" || lower == "none" || globalKeywords[lower] {
			return 0
		}
		if lower == "historical-forms" {
			alternates++
			continue
		}
		if alternatesOnly {
			return 0
		}
		category := fontVariantCategory(lower)
		if category == "" {
			return 0
		}
		if seen[category] {
			return 0
		}
		seen[category] = true
	}
	if consumed == 0 || (!alternatesOnly && len(seen) == 0 && alternates == 0) || (alternatesOnly && alternates == 0) {
		return 0
	}
	return consumed
}

func isFontVariantAlternatesFunction(name string) bool {
	switch strings.ToLower(name) {
	case "stylistic", "styleset", "character-variant", "swash", "ornaments", "annotation":
		return true
	default:
		return false
	}
}

func fontVariantCategory(value string) string {
	switch value {
	case "common-ligatures", "no-common-ligatures":
		return "common-lig"
	case "discretionary-ligatures", "no-discretionary-ligatures":
		return "discretionary-lig"
	case "historical-ligatures", "no-historical-ligatures":
		return "historical-lig"
	case "contextual", "no-contextual":
		return "contextual"
	case "small-caps", "all-small-caps", "petite-caps", "all-petite-caps", "unicase", "titling-caps":
		return "caps"
	case "lining-nums", "oldstyle-nums":
		return "numeric-figure"
	case "proportional-nums", "tabular-nums":
		return "numeric-spacing"
	case "diagonal-fractions", "stacked-fractions":
		return "numeric-fraction"
	case "ordinal":
		return "numeric-ordinal"
	case "slashed-zero":
		return "numeric-zero"
	case "jis78", "jis83", "jis90", "jis04", "simplified", "traditional":
		return "east-asian-variant"
	case "full-width", "proportional-width":
		return "east-asian-width"
	case "ruby":
		return "ruby"
	case "sub", "super":
		return "position"
	case "text", "emoji", "unicode":
		return "emoji"
	default:
		return ""
	}
}

func matchBackgroundClip(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	keywords := map[string]bool{
		"border-box": true, "padding-box": true, "content-box": true,
		"text": true, "border-area": true,
	}
	consumed := 0
	count := 0
	for _, value := range values {
		if token, ok := parser.AsToken(value); ok && token.Type == tokenizer.TokenComma {
			break
		}
		consumed++
		if parser.IsWhitespace(value) {
			continue
		}
		token, ok := parser.AsToken(value)
		if !ok || token.Type != tokenizer.TokenIdent || !keywords[strings.ToLower(token.Value)] {
			return 0
		}
		count++
		if count > 2 {
			return 0
		}
	}
	if count == 0 {
		return 0
	}
	return consumed
}

func matchPaletteIdentifier(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	token, ok := singleToken(values)
	if ok && token.Type == tokenizer.TokenIdent && strings.HasPrefix(token.Value, "--") {
		return 1
	}
	return 0
}

func matchVariationTagValue(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	if len(values) == 0 {
		return 0
	}
	token, ok := parser.AsToken(values[0])
	if !ok || token.Type != tokenizer.TokenString || len(token.Value) != 4 {
		return 0
	}
	index := 1
	for index < len(values) && parser.IsWhitespace(values[index]) {
		index++
	}
	if index >= len(values) {
		return 0
	}
	value := values[index]
	if next, ok := parser.AsToken(value); ok && next.Type == tokenizer.TokenNumber {
		return index + 1
	}
	if function, ok := value.(*parser.Function); ok {
		if expression, valid := cssvalues.ParseMath(function); valid {
			if simplified, ok := expression.Simplify(); ok && simplified.Kind == cssvalues.Other && simplified.Unit == "" {
				return index + 1
			}
		}
	}
	return 0
}

func matchTabSizeValue(values []parser.ComponentValue) int {
	if len(values) > 0 {
		if function, ok := values[0].(*parser.Function); ok {
			switch strings.ToLower(function.Name) {
			case "var", "env", "attr":
				return 1
			}
		}
	}
	token, ok := singleToken(values)
	if ok {
		if token.Type == tokenizer.TokenNumber && token.Number >= 0 {
			return 1
		}
		if token.Type == tokenizer.TokenDimension && token.Number >= 0 {
			if kind, _, valid := cssvalues.ClassifyUnit(token.Value); valid && kind == cssvalues.Length {
				return 1
			}
		}
	}
	function, ok := firstFunction(values)
	if !ok {
		return 0
	}
	expression, valid := cssvalues.ParseMath(function)
	if !valid || mathHasPercentage(expression) {
		return 0
	}
	return 1
}

func mathHasPercentage(expression *cssvalues.MathExpression) bool {
	if expression == nil {
		return false
	}
	if expression.Kind == cssvalues.MathValue && expression.Value.Unit == "%" {
		return true
	}
	if mathHasPercentage(expression.Left) || mathHasPercentage(expression.Right) {
		return true
	}
	for _, argument := range expression.Args {
		if mathHasPercentage(argument) {
			return true
		}
	}
	return false
}

func countNonWhitespace(values []parser.ComponentValue) int {
	count := 0
	for _, value := range values {
		if !parser.IsWhitespace(value) {
			count++
		}
	}
	return count
}

func validFontVariantAlternatesFunction(function *parser.Function) bool {
	name := strings.ToLower(function.Name)
	switch name {
	case "styleset", "character-variant":
		groups := splitCommaValues(function.Value)
		if len(groups) == 0 {
			return false
		}
		for _, group := range groups {
			if _, ok := singleNonWhitespaceIdent(group); !ok {
				return false
			}
		}
		return true
	case "stylistic", "swash", "ornaments", "annotation":
		groups := splitCommaValues(function.Value)
		if len(groups) != 1 {
			return false
		}
		_, ok := singleNonWhitespaceIdent(groups[0])
		return ok
	default:
		return false
	}
}

func singleNonWhitespaceIdent(values []parser.ComponentValue) (string, bool) {
	token, ok := singleNonWhitespaceToken(values)
	if !ok || token.Type != tokenizer.TokenIdent {
		return "", false
	}
	return token.Value, true
}

func matchTextUnderlinePosition(values []parser.ComponentValue) int {
	if consumed := matchDynamic(values); consumed > 0 {
		return consumed
	}
	words := []string{}
	consumed := 0
	for _, value := range values {
		if token, ok := parser.AsToken(value); ok && token.Type == tokenizer.TokenComma {
			break
		}
		consumed++
		if parser.IsWhitespace(value) {
			continue
		}
		token, ok := parser.AsToken(value)
		if !ok || token.Type != tokenizer.TokenIdent {
			return 0
		}
		words = append(words, strings.ToLower(token.Value))
	}
	switch len(words) {
	case 1:
		switch words[0] {
		case "auto", "under", "from-font", "left", "right":
			return consumed
		}
	case 2:
		first, second := words[0], words[1]
		if first == "left" || first == "right" {
			first, second = second, first
		}
		if (first == "under" || first == "from-font") && (second == "left" || second == "right") {
			return consumed
		}
	}
	return 0
}
