package styles

import (
	"strings"

	"gonetsurf/parsers/css/cssom"
	"gonetsurf/parsers/css/parser"
	"gonetsurf/parsers/css/tokenizer"
)

var lineStyles = map[string]bool{
	"none": true, "hidden": true, "dotted": true, "dashed": true, "solid": true,
	"double": true, "groove": true, "ridge": true, "inset": true, "outset": true,
}

var lineWidths = map[string]bool{"thin": true, "medium": true, "thick": true}

// expandDeclaration expands the common shorthands that the layout engine
// consumes into their longhand declarations.
func expandDeclaration(declaration *cssom.Declaration) []*cssom.Declaration {
	if declaration == nil {
		return nil
	}
	switch strings.ToLower(declaration.Name) {
	case "margin":
		return expandEdges("margin", declaration)
	case "padding":
		return expandEdges("padding", declaration)
	case "border-width":
		return expandBorderComponent("width", declaration)
	case "border-style":
		return expandBorderComponent("style", declaration)
	case "border-color":
		return expandBorderComponent("color", declaration)
	case "border":
		return expandBorder(declaration, "")
	case "border-top":
		return expandBorder(declaration, "top")
	case "border-right":
		return expandBorder(declaration, "right")
	case "border-bottom":
		return expandBorder(declaration, "bottom")
	case "border-left":
		return expandBorder(declaration, "left")
	case "overflow":
		return expandOverflow(declaration)
	default:
		return []*cssom.Declaration{declaration}
	}
}

func expandEdges(prefix string, declaration *cssom.Declaration) []*cssom.Declaration {
	values := splitWhitespaceValues(declaration.Value)
	if len(values) == 0 || len(values) > 4 {
		return []*cssom.Declaration{declaration}
	}
	sides := []string{"top", "right", "bottom", "left"}
	selected := make([][]parser.ComponentValue, 4)
	switch len(values) {
	case 1:
		for index := range selected {
			selected[index] = values[0]
		}
	case 2:
		selected[0], selected[2] = values[0], values[0]
		selected[1], selected[3] = values[1], values[1]
	case 3:
		selected[0] = values[0]
		selected[1], selected[3] = values[1], values[1]
		selected[2] = values[2]
	case 4:
		copy(selected, values)
	}
	declarations := make([]*cssom.Declaration, 0, 4)
	for index, side := range sides {
		declarations = append(declarations, &cssom.Declaration{
			Name:      prefix + "-" + side,
			Value:     selected[index],
			Important: declaration.Important,
		})
	}
	return declarations
}

func expandBorderComponent(component string, declaration *cssom.Declaration) []*cssom.Declaration {
	values := splitWhitespaceValues(declaration.Value)
	if len(values) == 0 || len(values) > 4 {
		return []*cssom.Declaration{declaration}
	}
	sides := []string{"top", "right", "bottom", "left"}
	selected := make([][]parser.ComponentValue, 4)
	switch len(values) {
	case 1:
		for index := range selected {
			selected[index] = values[0]
		}
	case 2:
		selected[0], selected[2] = values[0], values[0]
		selected[1], selected[3] = values[1], values[1]
	case 3:
		selected[0] = values[0]
		selected[1], selected[3] = values[1], values[1]
		selected[2] = values[2]
	case 4:
		copy(selected, values)
	}
	declarations := make([]*cssom.Declaration, 0, 4)
	for index, side := range sides {
		declarations = append(declarations, &cssom.Declaration{
			Name:      "border-" + side + "-" + component,
			Value:     selected[index],
			Important: declaration.Important,
		})
	}
	return declarations
}

func expandBorder(declaration *cssom.Declaration, side string) []*cssom.Declaration {
	width := parser.ParseComponentValueList("medium")
	style := parser.ParseComponentValueList("none")
	color := parser.ParseComponentValueList("currentcolor")
	for _, value := range splitWhitespaceValues(declaration.Value) {
		if len(value) != 1 {
			return []*cssom.Declaration{declaration}
		}
		token, ok := parser.AsToken(value[0])
		if !ok {
			if _, isFunction := value[0].(*parser.Function); isFunction {
				color = value
				continue
			}
			return []*cssom.Declaration{declaration}
		}
		switch token.Type {
		case tokenizer.TokenIdent:
			lower := strings.ToLower(token.Value)
			if lineStyles[lower] {
				style = value
			} else if lineWidths[lower] {
				width = value
			} else {
				color = value
			}
		case tokenizer.TokenDimension, tokenizer.TokenNumber, tokenizer.TokenHash:
			if token.Type == tokenizer.TokenHash {
				color = value
			} else {
				width = value
			}
		default:
			return []*cssom.Declaration{declaration}
		}
	}
	sides := []string{"top", "right", "bottom", "left"}
	if side != "" {
		sides = []string{side}
	}
	declarations := make([]*cssom.Declaration, 0, len(sides)*3)
	for _, current := range sides {
		declarations = append(declarations,
			&cssom.Declaration{Name: "border-" + current + "-width", Value: width, Important: declaration.Important},
			&cssom.Declaration{Name: "border-" + current + "-style", Value: style, Important: declaration.Important},
			&cssom.Declaration{Name: "border-" + current + "-color", Value: color, Important: declaration.Important},
		)
	}
	return declarations
}

func expandOverflow(declaration *cssom.Declaration) []*cssom.Declaration {
	values := splitWhitespaceValues(declaration.Value)
	if len(values) == 1 {
		return []*cssom.Declaration{
			{Name: "overflow-x", Value: values[0], Important: declaration.Important},
			{Name: "overflow-y", Value: values[0], Important: declaration.Important},
		}
	}
	if len(values) == 2 {
		return []*cssom.Declaration{
			{Name: "overflow-x", Value: values[0], Important: declaration.Important},
			{Name: "overflow-y", Value: values[1], Important: declaration.Important},
		}
	}
	return []*cssom.Declaration{declaration}
}

func splitWhitespaceValues(values []parser.ComponentValue) [][]parser.ComponentValue {
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
