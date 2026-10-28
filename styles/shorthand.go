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
	case "flex":
		return expandFlex(declaration)
	case "flex-flow":
		return expandFlexFlow(declaration)
	case "gap":
		return expandGap(declaration)
	case "margin-inline", "margin-block", "padding-inline", "padding-block",
		"inset-inline", "inset-block":
		return expandLogicalEdges(declaration)
	case "inset":
		return expandPhysicalEdges("inset", declaration, []string{"top", "right", "bottom", "left"})
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

func expandFlex(declaration *cssom.Declaration) []*cssom.Declaration {
	values := splitWhitespaceValues(declaration.Value)
	if len(values) == 0 {
		return []*cssom.Declaration{declaration}
	}
	grow := parser.ParseComponentValueList("0")
	shrink := parser.ParseComponentValueList("1")
	basis := parser.ParseComponentValueList("auto")
	if len(values) == 1 {
		if token, ok := parser.AsToken(values[0][0]); ok && token.Type == tokenizer.TokenIdent {
			switch strings.ToLower(token.Value) {
			case "none", "initial":
				shrink = parser.ParseComponentValueList("0")
			case "auto":
				grow = parser.ParseComponentValueList("1")
			}
		}
	}
	if len(values) >= 1 {
		if token, ok := parser.AsToken(values[0][0]); ok && token.Type == tokenizer.TokenNumber {
			grow = values[0]
			if len(values) == 1 {
				basis = parser.ParseComponentValueList("0%")
			}
		} else {
			grow = parser.ParseComponentValueList("1")
			basis = values[0]
		}
	}
	if len(values) >= 2 {
		if _, ok := parser.AsToken(values[1][0]); ok {
			if token, isNumber := parser.AsToken(values[1][0]); isNumber && token.Type == tokenizer.TokenNumber {
				shrink = values[1]
			} else {
				basis = values[1]
			}
		}
	}
	if len(values) >= 3 {
		basis = values[2]
	}
	return []*cssom.Declaration{
		{Name: "flex-grow", Value: grow, Important: declaration.Important},
		{Name: "flex-shrink", Value: shrink, Important: declaration.Important},
		{Name: "flex-basis", Value: basis, Important: declaration.Important},
	}
}

func expandFlexFlow(declaration *cssom.Declaration) []*cssom.Declaration {
	direction := parser.ParseComponentValueList("row")
	wrap := parser.ParseComponentValueList("nowrap")
	for _, value := range splitWhitespaceValues(declaration.Value) {
		if len(value) != 1 {
			return []*cssom.Declaration{declaration}
		}
		token, ok := parser.AsToken(value[0])
		if !ok || token.Type != tokenizer.TokenIdent {
			return []*cssom.Declaration{declaration}
		}
		switch strings.ToLower(token.Value) {
		case "row", "row-reverse", "column", "column-reverse":
			direction = value
		case "nowrap", "wrap", "wrap-reverse":
			wrap = value
		default:
			return []*cssom.Declaration{declaration}
		}
	}
	return []*cssom.Declaration{
		{Name: "flex-direction", Value: direction, Important: declaration.Important},
		{Name: "flex-wrap", Value: wrap, Important: declaration.Important},
	}
}

func expandGap(declaration *cssom.Declaration) []*cssom.Declaration {
	values := splitWhitespaceValues(declaration.Value)
	if len(values) == 0 || len(values) > 2 {
		return []*cssom.Declaration{declaration}
	}
	row := values[0]
	column := values[0]
	if len(values) == 2 {
		column = values[1]
	}
	return []*cssom.Declaration{
		{Name: "row-gap", Value: row, Important: declaration.Important},
		{Name: "column-gap", Value: column, Important: declaration.Important},
	}
}

func expandLogicalEdges(declaration *cssom.Declaration) []*cssom.Declaration {
	name := strings.ToLower(declaration.Name)
	parts := strings.SplitN(name, "-", 2)
	prefix := parts[0]
	axis := parts[1]
	values := splitWhitespaceValues(declaration.Value)
	if len(values) == 0 || len(values) > 2 {
		return []*cssom.Declaration{declaration}
	}
	start := values[0]
	end := values[0]
	if len(values) == 2 {
		end = values[1]
	}
	return []*cssom.Declaration{
		{Name: prefix + "-" + axis + "-start", Value: start, Important: declaration.Important},
		{Name: prefix + "-" + axis + "-end", Value: end, Important: declaration.Important},
	}
}

func expandPhysicalEdges(prefix string, declaration *cssom.Declaration, sides []string) []*cssom.Declaration {
	values := splitWhitespaceValues(declaration.Value)
	if len(values) == 0 || len(values) > 4 {
		return []*cssom.Declaration{declaration}
	}
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
