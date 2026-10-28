package styles

import (
	"math"
	"strconv"
	"strings"

	"gonetsurf/parsers/css/cascade"
	"gonetsurf/parsers/css/colors"
	"gonetsurf/parsers/css/parser"
	"gonetsurf/parsers/css/tokenizer"
	cssvalues "gonetsurf/parsers/css/values"
)

type computedContext struct {
	FontSize       float64
	ParentFontSize float64
	RootFontSize   float64
	ViewportWidth  float64
	ViewportHeight float64
}

var colorProperties = map[string]bool{
	"background-color": true, "border-top-color": true, "border-right-color": true,
	"border-bottom-color": true, "border-left-color": true, "outline-color": true,
	"text-decoration-color": true, "caret-color": true, "column-rule-color": true,
}

var borderWidthProperties = map[string]bool{
	"border-top-width": true, "border-right-width": true,
	"border-bottom-width": true, "border-left-width": true,
	"outline-width": true, "column-rule-width": true,
}

func computeStyleValues(values map[string][]parser.ComponentValue, parent *Style, context computedContext) map[string][]parser.ComponentValue {
	parentFontSize := 16.0
	parentWeight := 400.0
	parentColor := "black"
	if parent != nil {
		if size, ok := parseComputedPixels(parent.GetString("font-size")); ok && size > 0 {
			parentFontSize = size
		}
		if weight, ok := parseComputedNumber(parent.GetString("font-weight")); ok && weight > 0 {
			parentWeight = weight
		}
		if color := parent.GetString("color"); color != "" {
			parentColor = color
		}
	}
	context.ParentFontSize = parentFontSize
	fontSize, fontSizeValue := computeFontSize(values["font-size"], context, parentFontSize)
	context.FontSize = fontSize
	computed := map[string][]parser.ComponentValue{}
	computed["font-size"] = fontSizeValue
	colorValue := computeColor(values["color"], parentColor)
	computed["color"] = colorValue
	currentColor := strings.TrimSpace(parser.SerializeValues(colorValue))
	computed["font-weight"] = computeFontWeight(values["font-weight"], parentWeight)
	if raw, ok := values["line-height"]; ok {
		computed["line-height"] = computeLineHeight(raw, context)
	}
	for name, raw := range values {
		if cascade.IsCustom(name) {
			computed[name] = raw
			continue
		}
		if raw == nil {
			continue
		}
		switch name {
		case "font-size", "color", "font-weight", "line-height":
			continue
		}
		switch {
		case colorProperties[name]:
			computed[name] = computeColor(raw, currentColor)
		case borderWidthProperties[name]:
			computed[name] = computeBorderWidth(raw, context)
		case name == "opacity":
			computed[name] = computeOpacity(raw)
		default:
			computed[name] = computeGenericValues(raw, context)
		}
	}
	return computed
}

func computeFontSize(raw []parser.ComponentValue, context computedContext, parentFontSize float64) (float64, []parser.ComponentValue) {
	trimmed := trimComponentWhitespace(raw)
	if len(trimmed) == 0 {
		return parentFontSize, pxValue(parentFontSize)
	}
	if len(trimmed) == 1 {
		if token, ok := parser.AsToken(trimmed[0]); ok && token.Type == tokenizer.TokenIdent {
			if size, ok := absoluteFontSizes[strings.ToLower(token.Value)]; ok {
				return size, pxValue(size)
			}
			switch strings.ToLower(token.Value) {
			case "larger":
				return parentFontSize * 1.2, pxValue(parentFontSize * 1.2)
			case "smaller":
				return parentFontSize * 0.8, pxValue(parentFontSize * 0.8)
			case "math":
				return 16, pxValue(16)
			}
		}
	}
	fontContext := context
	fontContext.FontSize = parentFontSize
	if pixels, percent, ok := resolveToPixels(trimmed, fontContext); ok {
		if percent {
			pixels = parentFontSize * pixels
		}
		return pixels, pxValue(pixels)
	}
	return parentFontSize, pxValue(parentFontSize)
}

var absoluteFontSizes = map[string]float64{
	"xx-small": 9, "x-small": 10, "small": 13, "medium": 16,
	"large": 18, "x-large": 24, "xx-large": 32, "xxx-large": 48,
}

func computeFontWeight(raw []parser.ComponentValue, parentWeight float64) []parser.ComponentValue {
	trimmed := trimComponentWhitespace(raw)
	if len(trimmed) == 0 {
		return numberValue(400)
	}
	if len(trimmed) == 1 {
		if token, ok := parser.AsToken(trimmed[0]); ok && token.Type == tokenizer.TokenIdent {
			switch strings.ToLower(token.Value) {
			case "normal":
				return numberValue(400)
			case "bold":
				return numberValue(700)
			case "bolder":
				return numberValue(bolderWeight(parentWeight))
			case "lighter":
				return numberValue(lighterWeight(parentWeight))
			}
		}
		if token, ok := parser.AsToken(trimmed[0]); ok && token.Type == tokenizer.TokenNumber {
			return numberValue(token.Number)
		}
	}
	return numberValue(parentWeight)
}

func bolderWeight(parent float64) float64 {
	switch {
	case parent < 350:
		return 400
	case parent < 550:
		return 700
	default:
		return 900
	}
}

func lighterWeight(parent float64) float64 {
	switch {
	case parent < 550:
		return 100
	case parent < 750:
		return 400
	default:
		return 700
	}
}

func computeLineHeight(raw []parser.ComponentValue, context computedContext) []parser.ComponentValue {
	trimmed := trimComponentWhitespace(raw)
	if len(trimmed) == 1 {
		if token, ok := parser.AsToken(trimmed[0]); ok && token.Type == tokenizer.TokenIdent {
			if strings.EqualFold(token.Value, "normal") {
				return raw
			}
		}
		if token, ok := parser.AsToken(trimmed[0]); ok && token.Type == tokenizer.TokenNumber {
			return numberValue(token.Number)
		}
	}
	if pixels, percent, ok := resolveToPixels(trimmed, context); ok {
		if percent {
			pixels = context.FontSize * pixels
		}
		return pxValue(pixels)
	}
	return raw
}

func computeColor(raw []parser.ComponentValue, currentColor string) []parser.ComponentValue {
	if len(trimComponentWhitespace(raw)) == 0 {
		return parser.ParseComponentValueList(currentColor)
	}
	color, ok := colors.ParseValues(raw)
	if !ok {
		return parser.ParseComponentValueList("black")
	}
	if color.CurrentColor {
		if currentColor == "" {
			currentColor = "black"
		}
		return parser.ParseComponentValueList(currentColor)
	}
	return parser.ParseComponentValueList(color.Serialize())
}

func computeBorderWidth(raw []parser.ComponentValue, context computedContext) []parser.ComponentValue {
	trimmed := trimComponentWhitespace(raw)
	if len(trimmed) == 1 {
		if token, ok := parser.AsToken(trimmed[0]); ok && token.Type == tokenizer.TokenIdent {
			switch strings.ToLower(token.Value) {
			case "thin":
				return pxValue(1)
			case "medium":
				return pxValue(3)
			case "thick":
				return pxValue(5)
			}
		}
	}
	if pixels, _, ok := resolveToPixels(trimmed, context); ok {
		return pxValue(pixels)
	}
	return raw
}

func computeOpacity(raw []parser.ComponentValue) []parser.ComponentValue {
	trimmed := trimComponentWhitespace(raw)
	if len(trimmed) != 1 {
		return raw
	}
	numeric, ok := cssvalues.ParseNumeric(trimmed[0])
	if !ok {
		return raw
	}
	value := numeric.Number
	if numeric.Unit == "%" {
		value = numeric.Number
	}
	value = math.Max(0, math.Min(1, value))
	return numberValue(value)
}

func computeGenericValues(values []parser.ComponentValue, context computedContext) []parser.ComponentValue {
	result := make([]parser.ComponentValue, 0, len(values))
	for _, value := range values {
		if token, ok := parser.AsToken(value); ok {
			if token.Type == tokenizer.TokenDimension {
				numeric, ok := cssvalues.ParseNumeric(value)
				if ok {
					if numeric.Kind == cssvalues.Length {
						if pixels, ok := lengthToPixels(numeric, context); ok {
							result = append(result, pxValue(pixels)...)
							continue
						}
					}
					if numeric.Kind == cssvalues.Angle {
						if degrees, ok := cssvalues.ToDegrees(numeric); ok {
							result = append(result, degValue(degrees)...)
							continue
						}
					}
					if numeric.Kind == cssvalues.Time {
						if seconds, ok := cssvalues.ToSeconds(numeric); ok {
							result = append(result, secondsValue(seconds)...)
							continue
						}
					}
				}
			}
			result = append(result, value)
			continue
		}
		if function, ok := value.(*parser.Function); ok {
			inner := computeGenericValues(function.Value, context)
			if isMathFunction(function.Name) {
				candidate := &parser.Function{Name: function.Name, Value: inner}
				if expression, ok := cssvalues.ParseMath(candidate); ok {
					if simplified, ok := expression.Simplify(); ok {
						if converted := computedValueComponents(simplified); converted != nil {
							result = append(result, converted...)
							continue
						}
					}
				}
			}
			result = append(result, &parser.Function{Name: function.Name, Value: inner})
			continue
		}
		if block, ok := value.(*parser.Block); ok {
			result = append(result, &parser.Block{Associated: block.Associated, Value: computeGenericValues(block.Value, context)})
			continue
		}
		result = append(result, value)
	}
	return result
}

func isMathFunction(name string) bool {
	switch strings.ToLower(name) {
	case "calc", "min", "max", "clamp", "round", "mod", "rem", "abs", "sign",
		"hypot", "pow", "sqrt", "log", "exp", "sin", "cos", "tan", "asin",
		"acos", "atan", "atan2":
		return true
	default:
		return false
	}
}

func resolveToPixels(values []parser.ComponentValue, context computedContext) (float64, bool, bool) {
	generic := computeGenericValues(values, context)
	if len(generic) != 1 {
		return 0, false, false
	}
	numeric, ok := cssvalues.ParseNumeric(generic[0])
	if !ok {
		return 0, false, false
	}
	if numeric.Unit == "%" {
		return numeric.Number, true, true
	}
	if numeric.Kind == cssvalues.Length {
		pixels, ok := lengthToPixels(numeric, context)
		return pixels, false, ok
	}
	if numeric.Kind == cssvalues.Other && numeric.Unit == "" {
		return numeric.Number, false, true
	}
	return 0, false, false
}

func lengthToPixels(value cssvalues.Value, context computedContext) (float64, bool) {
	unit := strings.ToLower(value.Unit)
	number := value.Number
	switch unit {
	case "em":
		return number * context.FontSize, true
	case "rem":
		return number * context.RootFontSize, true
	case "ex", "ch", "cap":
		return number * context.FontSize * 0.5, true
	case "ic":
		return number * context.FontSize, true
	case "lh", "rlh":
		return number * context.FontSize * 1.2, true
	case "vw", "svw", "lvw", "dvw":
		return number * context.ViewportWidth / 100, true
	case "vh", "svh", "lvh", "dvh":
		return number * context.ViewportHeight / 100, true
	case "vmin", "svmin", "lvmin", "dvmin":
		smaller := math.Min(context.ViewportWidth, context.ViewportHeight)
		return number * smaller / 100, true
	case "vmax", "svmax", "lvmax", "dvmax":
		larger := math.Max(context.ViewportWidth, context.ViewportHeight)
		return number * larger / 100, true
	case "cqw", "cqi":
		return number * context.ViewportWidth / 100, true
	case "cqh", "cqb":
		return number * context.ViewportHeight / 100, true
	case "cqmin":
		return number * math.Min(context.ViewportWidth, context.ViewportHeight) / 100, true
	case "cqmax":
		return number * math.Max(context.ViewportWidth, context.ViewportHeight) / 100, true
	default:
		return cssvalues.ToPixels(value, context.ViewportWidth, context.ViewportHeight)
	}
}

func computedValueComponents(value cssvalues.Value) []parser.ComponentValue {
	switch {
	case value.Unit == "%":
		return percentageValue(value.Number)
	case value.Kind == cssvalues.Length:
		return pxValue(value.Number)
	case value.Kind == cssvalues.Angle:
		return degValue(value.Number)
	case value.Kind == cssvalues.Time:
		return secondsValue(value.Number)
	case value.Kind == cssvalues.Other && value.Unit == "":
		return numberValue(value.Number)
	default:
		return nil
	}
}

func pxValue(number float64) []parser.ComponentValue {
	return parser.ParseComponentValueList(formatComputedNumber(number) + "px")
}

func numberValue(number float64) []parser.ComponentValue {
	return parser.ParseComponentValueList(formatComputedNumber(number))
}

func percentageValue(number float64) []parser.ComponentValue {
	return parser.ParseComponentValueList(formatComputedNumber(number*100) + "%")
}

func degValue(number float64) []parser.ComponentValue {
	return parser.ParseComponentValueList(formatComputedNumber(number) + "deg")
}

func secondsValue(number float64) []parser.ComponentValue {
	return parser.ParseComponentValueList(formatComputedNumber(number) + "s")
}

func formatComputedNumber(number float64) string {
	return strconv.FormatFloat(number, 'f', -1, 64)
}

func trimComponentWhitespace(values []parser.ComponentValue) []parser.ComponentValue {
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

func parseComputedPixels(text string) (float64, bool) {
	values := parser.ParseComponentValueList(text)
	trimmed := trimComponentWhitespace(values)
	if len(trimmed) != 1 {
		return 0, false
	}
	numeric, ok := cssvalues.ParseNumeric(trimmed[0])
	if !ok || numeric.Kind != cssvalues.Length {
		return 0, false
	}
	return numeric.Number, strings.EqualFold(numeric.Unit, "px")
}

func parseComputedNumber(text string) (float64, bool) {
	values := parser.ParseComponentValueList(text)
	trimmed := trimComponentWhitespace(values)
	if len(trimmed) != 1 {
		return 0, false
	}
	token, ok := parser.AsToken(trimmed[0])
	if !ok || token.Type != tokenizer.TokenNumber {
		return 0, false
	}
	return token.Number, true
}
