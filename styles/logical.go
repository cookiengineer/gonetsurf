package styles

import (
	"strings"

	"gonetsurf/parsers/css/cascade"
)

func logicalSides(writingMode, direction string) (inlineStart, inlineEnd, blockStart, blockEnd string) {
	mode := strings.ToLower(strings.TrimSpace(writingMode))
	dir := strings.ToLower(strings.TrimSpace(direction))
	switch mode {
	case "vertical-rl", "sideways-rl":
		return "top", "bottom", "right", "left"
	case "vertical-lr", "sideways-lr":
		return "top", "bottom", "left", "right"
	default:
		if dir == "rtl" {
			return "right", "left", "top", "bottom"
		}
		return "left", "right", "top", "bottom"
	}
}

func logicalProperty(name, writingMode, direction string) string {
	inlineStart, inlineEnd, blockStart, blockEnd := logicalSides(writingMode, direction)
	lower := strings.ToLower(name)
	horizontal := strings.TrimSpace(writingMode) == "" || strings.HasPrefix(strings.ToLower(writingMode), "horizontal")
	switch lower {
	case "inline-size":
		if horizontal {
			return "width"
		}
		return "height"
	case "block-size":
		if horizontal {
			return "height"
		}
		return "width"
	}
	axes := []struct {
		axis  string
		start string
		end   string
	}{
		{"inline", inlineStart, inlineEnd},
		{"block", blockStart, blockEnd},
	}
	for _, axis := range axes {
		portions := []struct {
			suffix string
			side   string
		}{
			{"-start", axis.start},
			{"-end", axis.end},
		}
		for _, portion := range portions {
			suffix := "-" + axis.axis + portion.suffix
			if lower == "margin"+suffix {
				return "margin-" + portion.side
			}
			if lower == "padding"+suffix {
				return "padding-" + portion.side
			}
			if lower == "inset"+suffix {
				return portion.side
			}
			if strings.HasPrefix(lower, "border"+suffix+"-") {
				component := strings.TrimPrefix(lower, "border"+suffix+"-")
				return "border-" + portion.side + "-" + component
			}
		}
	}
	return ""
}

func expandLogicalDeclarations(declarations []cascade.Declaration, writingMode, direction string) []cascade.Declaration {
	expanded := make([]cascade.Declaration, 0, len(declarations))
	for _, declaration := range declarations {
		if mapped := logicalProperty(declaration.Property, writingMode, direction); mapped != "" {
			declaration.Property = mapped
		}
		expanded = append(expanded, declaration)
	}
	return expanded
}
