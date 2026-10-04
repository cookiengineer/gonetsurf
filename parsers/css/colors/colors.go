// Package colors implements CSS Color Level 4 parsing and serialization.
//
// It supports hexadecimal colors, named colors, rgb()/rgba()/hsl()/hsla(),
// hwb(), the lab/lch/oklab/oklch() families, and color(). Colors are
// serialized using the Color Level 4 rules.
//
// See specifications/css/color-4.html.
package colors

import (
	"math"
	"strconv"
	"strings"

	"gonetsurf/parsers/css/parser"
	"gonetsurf/parsers/css/tokenizer"
)

// Space identifies a color space.
type Space uint8

const (
	// SpaceRGB is sRGB with components in the 0-255 range.
	SpaceRGB Space = iota
	// SpaceHSL stores hue in degrees and saturation/lightness as 0-1.
	SpaceHSL
	// SpaceHWB stores hue in degrees and whiteness/blackness as 0-1.
	SpaceHWB
	// SpaceLab is CIE Lab.
	SpaceLab
	// SpaceLCH is CIE LCH.
	SpaceLCH
	// SpaceOKLab is OKLab.
	SpaceOKLab
	// SpaceOKLCH is OKLCH.
	SpaceOKLCH
	// SpaceCustom is a color() color space such as srgb or display-p3.
	SpaceCustom
)

// Color is a parsed CSS color.
type Color struct {
	Space       Space
	Components  [3]float64
	Alpha       float64
	HasAlpha    bool
	None        [3]bool
	CustomSpace string
	// CurrentColor marks the currentcolor keyword.
	CurrentColor bool
}

// Parse parses a color from a string.
func Parse(input string) (*Color, bool) {
	return ParseValues(parser.ParseComponentValueList(input))
}

// ParseValues parses a color from component values.
func ParseValues(values []parser.ComponentValue) (*Color, bool) {
	trimmed := trimWhitespace(values)
	if len(trimmed) != 1 {
		return nil, false
	}
	value := trimmed[0]
	if token, ok := parser.AsToken(value); ok {
		switch token.Type {
		case tokenizer.TokenHash:
			return parseHex(token.Value)
		case tokenizer.TokenIdent:
			return parseNamed(token.Value)
		default:
			return nil, false
		}
	}
	if function, ok := value.(*parser.Function); ok {
		return parseFunction(function)
	}
	return nil, false
}

func parseHex(value string) (*Color, bool) {
	digits := strings.TrimPrefix(value, "#")
	var expanded string
	switch len(digits) {
	case 3, 4:
		for _, digit := range digits {
			expanded += string(digit) + string(digit)
		}
	case 6, 8:
		expanded = digits
	default:
		return nil, false
	}
	components := [3]float64{}
	for index := 0; index < 3; index++ {
		parsed, err := strconv.ParseUint(expanded[index*2:index*2+2], 16, 8)
		if err != nil {
			return nil, false
		}
		components[index] = float64(parsed)
	}
	color := &Color{Space: SpaceRGB, Components: components, Alpha: 1}
	if len(expanded) == 8 {
		parsed, err := strconv.ParseUint(expanded[6:8], 16, 8)
		if err != nil {
			return nil, false
		}
		color.Alpha = float64(parsed) / 255
		color.HasAlpha = true
	}
	return color, true
}

func parseNamed(value string) (*Color, bool) {
	lower := strings.ToLower(value)
	if lower == "currentcolor" {
		return &Color{CurrentColor: true}, true
	}
	components, ok := namedColors[lower]
	if !ok {
		return nil, false
	}
	color := &Color{Space: SpaceRGB, Components: components, Alpha: 1}
	if lower == "transparent" {
		color.Alpha = 0
		color.HasAlpha = true
	}
	return color, true
}

func parseFunction(function *parser.Function) (*Color, bool) {
	name := strings.ToLower(function.Name)
	switch name {
	case "rgb", "rgba":
		return parseRGB(function.Value)
	case "hsl", "hsla":
		return parseHSL(function.Value)
	case "hwb":
		return parseHWB(function.Value)
	case "lab":
		return parseLabLike(function.Value, SpaceLab)
	case "lch":
		return parseLabLike(function.Value, SpaceLCH)
	case "oklab":
		return parseLabLike(function.Value, SpaceOKLab)
	case "oklch":
		return parseLabLike(function.Value, SpaceOKLCH)
	case "color":
		return parseColorFunction(function.Value)
	default:
		return nil, false
	}
}

// arguments splits a function's arguments into components and an optional
// alpha. It accepts both the comma-separated legacy form and the slash form.
func arguments(values []parser.ComponentValue) (components [][]parser.ComponentValue, alpha []parser.ComponentValue, ok bool) {
	trimmed := trimWhitespace(values)
	commaCount := 0
	for _, value := range trimmed {
		if token, isToken := parser.AsToken(value); isToken && token.Type == tokenizer.TokenComma {
			commaCount++
		}
	}
	if commaCount > 0 {
		groups := splitCommas(trimmed)
		if len(groups) < 3 || len(groups) > 4 {
			return nil, nil, false
		}
		components = groups[:3]
		if len(groups) == 4 {
			alpha = groups[3]
		}
		for index := range components {
			components[index] = trimWhitespace(components[index])
		}
		return components, trimWhitespace(alpha), true
	}
	// Modern syntax: components are individual tokens, optionally followed by
	// "/" and an alpha value. Whitespace between tokens is optional.
	seenSlash := false
	for _, value := range trimmed {
		if parser.IsWhitespace(value) {
			continue
		}
		if token, isToken := parser.AsToken(value); isToken && token.Type == tokenizer.TokenDelim && token.Value == "/" {
			if seenSlash {
				return nil, nil, false
			}
			seenSlash = true
			continue
		}
		if seenSlash {
			alpha = append(alpha, value)
			continue
		}
		components = append(components, []parser.ComponentValue{value})
	}
	if len(components) != 3 {
		return nil, nil, false
	}
	return components, alpha, true
}

// usesCommas reports whether the argument list contains a top-level comma.
func usesCommas(values []parser.ComponentValue) bool {
	for _, value := range values {
		if token, ok := parser.AsToken(value); ok && token.Type == tokenizer.TokenComma {
			return true
		}
	}
	return false
}

// channelKind returns 'n' for numbers and 'p' for percentages.
func channelKind(group []parser.ComponentValue) rune {
	if len(group) != 1 {
		return 0
	}
	token, ok := parser.AsToken(group[0])
	if !ok {
		return 0
	}
	if token.Type == tokenizer.TokenPercentage {
		return 'p'
	}
	return 'n'
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

func splitWhitespace(values []parser.ComponentValue) [][]parser.ComponentValue {
	groups := [][]parser.ComponentValue{}
	current := []parser.ComponentValue{}
	for _, value := range values {
		if parser.IsWhitespace(value) {
			if len(current) > 0 {
				groups = append(groups, current)
				current = nil
			}
			continue
		}
		current = append(current, value)
	}
	if len(current) > 0 {
		groups = append(groups, current)
	}
	return groups
}

func parseRGB(values []parser.ComponentValue) (*Color, bool) {
	components, alpha, ok := arguments(values)
	if !ok {
		return nil, false
	}
	// Legacy comma syntax requires every channel to use the same type.
	if usesCommas(values) {
		first := channelKind(components[0])
		for _, group := range components[1:] {
			if channelKind(group) != first {
				return nil, false
			}
		}
	}
	color := &Color{Space: SpaceRGB, Alpha: 1}
	for index, group := range components {
		number, none, ok := parseChannel(group, 255)
		if !ok {
			return nil, false
		}
		color.Components[index] = clamp(number, 0, 255)
		color.None[index] = none
	}
	if len(alpha) > 0 {
		value, _, ok := parseAlpha(alpha)
		if !ok {
			return nil, false
		}
		color.Alpha = value
		color.HasAlpha = true
	}
	return color, true
}

func parseHSL(values []parser.ComponentValue) (*Color, bool) {
	components, alpha, ok := arguments(values)
	if !ok {
		return nil, false
	}
	hue, _, ok := parseHue(components[0])
	if !ok {
		return nil, false
	}
	saturation, _, ok := parseFraction(components[1])
	if !ok {
		return nil, false
	}
	lightness, _, ok := parseFraction(components[2])
	if !ok {
		return nil, false
	}
	color := &Color{Space: SpaceHSL, Components: [3]float64{hue, clamp(saturation, 0, 1), clamp(lightness, 0, 1)}, Alpha: 1}
	if len(alpha) > 0 {
		value, _, ok := parseAlpha(alpha)
		if !ok {
			return nil, false
		}
		color.Alpha = value
		color.HasAlpha = true
	}
	return color, true
}

func parseHWB(values []parser.ComponentValue) (*Color, bool) {
	components, alpha, ok := arguments(values)
	if !ok {
		return nil, false
	}
	hue, _, ok := parseHue(components[0])
	if !ok {
		return nil, false
	}
	whiteness, _, ok := parseFraction(components[1])
	if !ok {
		return nil, false
	}
	blackness, _, ok := parseFraction(components[2])
	if !ok {
		return nil, false
	}
	color := &Color{Space: SpaceHWB, Components: [3]float64{hue, clamp(whiteness, 0, 1), clamp(blackness, 0, 1)}, Alpha: 1}
	if len(alpha) > 0 {
		value, _, ok := parseAlpha(alpha)
		if !ok {
			return nil, false
		}
		color.Alpha = value
		color.HasAlpha = true
	}
	return color, true
}

func parseLabLike(values []parser.ComponentValue, space Space) (*Color, bool) {
	components, alpha, ok := arguments(values)
	if !ok {
		return nil, false
	}
	lightnessScale := 100.0
	chromaScale := 1.0
	hasHue := false
	switch space {
	case SpaceLab:
		chromaScale = 125
	case SpaceLCH:
		chromaScale = 150
		hasHue = true
	case SpaceOKLab:
		lightnessScale = 1
		chromaScale = 0.4
	case SpaceOKLCH:
		lightnessScale = 1
		chromaScale = 0.4
		hasHue = true
	}
	color := &Color{Space: space, Alpha: 1}
	lightness, none, ok := parseChannel(components[0], lightnessScale)
	if !ok {
		return nil, false
	}
	color.Components[0] = lightness
	color.None[0] = none
	// Chroma (or the a/b axes).
	chroma, none, ok := parseChannel(components[1], chromaScale)
	if !ok {
		return nil, false
	}
	color.Components[1] = chroma
	color.None[1] = none
	// Hue for the LCH spaces, otherwise the third axis.
	if hasHue {
		hue, none, ok := parseHue(components[2])
		if !ok {
			return nil, false
		}
		if !none {
			hue = math.Mod(hue, 360)
			if hue < 0 {
				hue += 360
			}
		}
		color.Components[2] = hue
		color.None[2] = none
	} else {
		third, none, ok := parseChannel(components[2], chromaScale)
		if !ok {
			return nil, false
		}
		color.Components[2] = third
		color.None[2] = none
	}
	if len(alpha) > 0 {
		value, _, ok := parseAlpha(alpha)
		if !ok {
			return nil, false
		}
		color.Alpha = value
		color.HasAlpha = true
	}
	return color, true
}

func parseColorFunction(values []parser.ComponentValue) (*Color, bool) {
	trimmed := trimWhitespace(values)
	if len(trimmed) == 0 {
		return nil, false
	}
	spaceToken, ok := parser.AsToken(trimmed[0])
	if !ok || spaceToken.Type != tokenizer.TokenIdent {
		return nil, false
	}
	rest := trimmed[1:]
	components, alpha, ok := arguments(rest)
	if !ok {
		return nil, false
	}
	spaceName := strings.ToLower(spaceToken.Value)
	if !isValidColorSpace(spaceName) {
		return nil, false
	}
	color := &Color{Space: SpaceCustom, CustomSpace: spaceName, Alpha: 1}
	for index, group := range components {
		number, none, ok := parseChannel(group, 1)
		if !ok {
			return nil, false
		}
		color.Components[index] = number
		color.None[index] = none
	}
	if len(alpha) > 0 {
		value, _, ok := parseAlpha(alpha)
		if !ok {
			return nil, false
		}
		color.Alpha = value
		color.HasAlpha = true
	}
	return color, true
}

// predefinedSpaces is the set of color() spaces defined by CSS Color Level 4,
// plus custom --spaces.
var predefinedSpaces = map[string]bool{
	"srgb": true, "srgb-linear": true, "display-p3": true, "a98-rgb": true,
	"prophoto-rgb": true, "rec2020": true, "xyz": true, "xyz-d50": true,
	"xyz-d65": true,
}

func isValidColorSpace(name string) bool {
	if strings.HasPrefix(name, "--") {
		return true
	}
	return predefinedSpaces[name]
}

func parseChannel(group []parser.ComponentValue, percentageScale float64) (float64, bool, bool) {
	if len(group) != 1 {
		return 0, false, false
	}
	token, ok := parser.AsToken(group[0])
	if !ok {
		return 0, false, false
	}
	switch token.Type {
	case tokenizer.TokenNumber:
		return token.Number, false, true
	case tokenizer.TokenPercentage:
		return token.Number / 100 * percentageScale, false, true
	case tokenizer.TokenIdent:
		if strings.EqualFold(token.Value, "none") {
			return 0, true, true
		}
	}
	return 0, false, false
}

// parseFraction parses a percentage-or-number argument where a bare number is
// interpreted as a percentage (used by hsl() saturation/lightness and hwb()
// whiteness/blackness).
func parseFraction(group []parser.ComponentValue) (float64, bool, bool) {
	if len(group) != 1 {
		return 0, false, false
	}
	token, ok := parser.AsToken(group[0])
	if !ok {
		return 0, false, false
	}
	switch token.Type {
	case tokenizer.TokenNumber:
		return token.Number / 100, false, true
	case tokenizer.TokenPercentage:
		return token.Number / 100, false, true
	case tokenizer.TokenIdent:
		if strings.EqualFold(token.Value, "none") {
			return 0, true, true
		}
	}
	return 0, false, false
}

func parseHue(group []parser.ComponentValue) (float64, bool, bool) {
	if len(group) != 1 {
		return 0, false, false
	}
	token, ok := parser.AsToken(group[0])
	if !ok {
		return 0, false, false
	}
	switch token.Type {
	case tokenizer.TokenNumber:
		return token.Number, false, true
	case tokenizer.TokenDimension:
		factor, valid := angleFactor(token.Value)
		if !valid {
			return 0, false, false
		}
		return token.Number * factor, false, true
	case tokenizer.TokenIdent:
		if strings.EqualFold(token.Value, "none") {
			return 0, true, true
		}
	}
	return 0, false, false
}

func angleFactor(unit string) (float64, bool) {
	switch strings.ToLower(unit) {
	case "deg":
		return 1, true
	case "grad":
		return 360.0 / 400.0, true
	case "rad":
		return 180.0 / math.Pi, true
	case "turn":
		return 360, true
	default:
		return 0, false
	}
}

func parseAlpha(group []parser.ComponentValue) (float64, bool, bool) {
	if len(group) != 1 {
		return 0, false, false
	}
	token, ok := parser.AsToken(group[0])
	if !ok {
		return 0, false, false
	}
	switch token.Type {
	case tokenizer.TokenNumber:
		return clamp(token.Number, 0, 1), false, true
	case tokenizer.TokenPercentage:
		return clamp(token.Number/100, 0, 1), false, true
	case tokenizer.TokenIdent:
		if strings.EqualFold(token.Value, "none") {
			return 0, true, true
		}
	}
	return 0, false, false
}

func clampi(value, minimum, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func clamp(value, minimum, maximum float64) float64 {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
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
