package cssom

import (
	"strings"

	"gonetsurf/parsers/css/parser"
	"gonetsurf/parsers/css/tokenizer"
)

// FontSource is one src() entry in an @font-face rule.
type FontSource struct {
	URL    string
	Local  string
	Format string
}

// UnicodeRange is an inclusive code-point range from unicode-range.
type UnicodeRange struct {
	Start int
	End   int
}

// Contains reports whether character is inside the range.
func (unicodeRange UnicodeRange) Contains(character rune) bool {
	return int(character) >= unicodeRange.Start && int(character) <= unicodeRange.End
}

// FontFaceRule is a CSS @font-face rule.
type FontFaceRule struct {
	Declarations []*Declaration
	Family       string
	Sources      []FontSource
	FontStyle    string
	FontWeight   string
	FontStretch  string
	UnicodeRange []UnicodeRange
	FontDisplay  string

	FeatureSettings   string
	VariationSettings string
	SizeAdjust        string
	AscentOverride    string
	DescentOverride   string
	LineGapOverride   string
}

func (*FontFaceRule) rule() {}

// Style returns a mutable view of the rule's descriptors.
func (rule *FontFaceRule) Style() *CSSStyleDeclaration {
	return &CSSStyleDeclaration{declarations: &rule.Declarations}
}

// Serialize returns the @font-face rule as CSS text.
func (rule *FontFaceRule) Serialize() string {
	var builder strings.Builder
	builder.WriteString("@font-face {")
	for index, declaration := range rule.Declarations {
		if index > 0 {
			builder.WriteByte(' ')
		}
		builder.WriteString(declaration.Name)
		builder.WriteString(": ")
		builder.WriteString(strings.TrimSpace(parser.SerializeValues(declaration.Value)))
		builder.WriteByte(';')
	}
	builder.WriteString(" }")
	return builder.String()
}

func convertFontFaceRule(typed *parser.AtRule) *FontFaceRule {
	rule := &FontFaceRule{}
	if typed.Block != nil {
		declarations, _ := convertBlockContents(typed.Block.Value)
		rule.Declarations = declarations
	}
	for _, declaration := range rule.Declarations {
		name := strings.ToLower(declaration.Name)
		value := strings.TrimSpace(parser.SerializeValues(declaration.Value))
		switch name {
		case "font-family":
			rule.Family = parseFontFamily(declaration.Value)
		case "src":
			rule.Sources = parseFontSources(declaration.Value)
		case "font-style":
			rule.FontStyle = value
		case "font-weight":
			rule.FontWeight = value
		case "font-stretch":
			rule.FontStretch = value
		case "unicode-range":
			rule.UnicodeRange = parseUnicodeRanges(declaration.Value)
		case "font-display":
			rule.FontDisplay = value
		case "font-feature-settings":
			rule.FeatureSettings = value
		case "font-variation-settings":
			rule.VariationSettings = value
		case "size-adjust":
			rule.SizeAdjust = value
		case "ascent-override":
			rule.AscentOverride = value
		case "descent-override":
			rule.DescentOverride = value
		case "line-gap-override":
			rule.LineGapOverride = value
		}
	}
	return rule
}

func parseFontFamily(values []parser.ComponentValue) string {
	for _, value := range values {
		if parser.IsWhitespace(value) {
			continue
		}
		if token, ok := parser.AsToken(value); ok {
			switch token.Type {
			case tokenizer.TokenString, tokenizer.TokenIdent:
				return token.Value
			}
		}
	}
	return ""
}

func parseFontSources(values []parser.ComponentValue) []FontSource {
	groups := splitFontCommaGroups(values)
	var sources []FontSource
	for _, group := range groups {
		source := FontSource{}
		for _, value := range group {
			if parser.IsWhitespace(value) {
				continue
			}
			if token, ok := parser.AsToken(value); ok && token.Type == tokenizer.TokenURL {
				source.URL = token.Value
				continue
			}
			function, ok := value.(*parser.Function)
			if !ok {
				continue
			}
			switch strings.ToLower(function.Name) {
			case "url":
				if text, ok := firstFunctionString(function.Value); ok {
					source.URL = text
				}
			case "local":
				if text, ok := firstFunctionString(function.Value); ok {
					source.Local = text
				}
			case "format":
				if text, ok := firstFunctionString(function.Value); ok {
					source.Format = text
				}
			}
		}
		if source.URL != "" || source.Local != "" {
			sources = append(sources, source)
		}
	}
	return sources
}

func firstFunctionString(values []parser.ComponentValue) (string, bool) {
	for _, value := range values {
		if parser.IsWhitespace(value) {
			continue
		}
		if token, ok := parser.AsToken(value); ok {
			if token.Type == tokenizer.TokenString || token.Type == tokenizer.TokenIdent || token.Type == tokenizer.TokenURL {
				return token.Value, true
			}
		}
	}
	return "", false
}

func parseUnicodeRanges(values []parser.ComponentValue) []UnicodeRange {
	var ranges []UnicodeRange
	for _, value := range values {
		if token, ok := parser.AsToken(value); ok && token.Type == tokenizer.TokenUnicodeRange {
			ranges = append(ranges, UnicodeRange{Start: token.RangeStart, End: token.RangeEnd})
		}
	}
	if len(ranges) > 0 {
		return ranges
	}
	var builder strings.Builder
	for _, value := range values {
		if token, ok := parser.AsToken(value); ok && token.Representation != "" {
			builder.WriteString(token.Representation)
		} else {
			builder.WriteString(parser.SerializeValues([]parser.ComponentValue{value}))
		}
	}
	for _, token := range tokenizer.TokenizeWithOptions(builder.String(), tokenizer.Options{UnicodeRangesAllowed: true}) {
		if token.Type == tokenizer.TokenUnicodeRange {
			ranges = append(ranges, UnicodeRange{Start: token.RangeStart, End: token.RangeEnd})
		}
	}
	return ranges
}

func splitFontCommaGroups(values []parser.ComponentValue) [][]parser.ComponentValue {
	var groups [][]parser.ComponentValue
	var current []parser.ComponentValue
	for _, value := range values {
		if token, ok := parser.AsToken(value); ok && token.Type == tokenizer.TokenComma {
			groups = append(groups, current)
			current = nil
			continue
		}
		current = append(current, value)
	}
	if len(current) > 0 || len(groups) > 0 {
		groups = append(groups, current)
	}
	return groups
}
