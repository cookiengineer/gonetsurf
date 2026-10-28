package layouts

import (
	"math"
	"strings"

	"gonetsurf/parsers/css/parser"
	cssvalues "gonetsurf/parsers/css/values"
	"gonetsurf/styles"
)

// Size is a computed width or height before containing-block resolution.
type Size struct {
	Value   float64
	Percent bool
	Auto    bool
	Keyword string
}

// Layout performs layout for a box tree.
type Layout struct {
	ViewportWidth  float64
	ViewportHeight float64
	// MeasureText overrides the default simple advance-width measurement.
	MeasureText func(text string, style *styles.Style) (width, height float64)
}

// New creates a layout context for a viewport.
func New(viewportWidth, viewportHeight float64) *Layout {
	return &Layout{ViewportWidth: viewportWidth, ViewportHeight: viewportHeight}
}

// Layout lays out root in the viewport.
func (layout *Layout) Layout(root *Box) {
	if root == nil {
		return
	}
	layout.layoutBlock(root, 0, 0, layout.ViewportWidth, layout.ViewportHeight)
}

func (layout *Layout) layoutBlock(box *Box, containingX, containingY, containingWidth, containingHeight float64) {
	if box == nil || box.Style == nil {
		return
	}
	marginLeft := box.Margin.Left.Resolve(containingWidth)
	marginRight := box.Margin.Right.Resolve(containingWidth)
	marginTop := box.Margin.Top.Resolve(containingWidth)
	paddingLeft := box.Padding.Left.Resolve(containingWidth)
	paddingRight := box.Padding.Right.Resolve(containingWidth)
	paddingTop := box.Padding.Top.Resolve(containingWidth)
	paddingBottom := box.Padding.Bottom.Resolve(containingWidth)
	borderLeft := box.Border.Left.Resolve(containingWidth)
	borderRight := box.Border.Right.Resolve(containingWidth)
	borderTop := box.Border.Top.Resolve(containingWidth)
	borderBottom := box.Border.Bottom.Resolve(containingWidth)
	horizontalExtra := paddingLeft + paddingRight + borderLeft + borderRight
	box.ContentWidth = resolveWidth(box.Style, containingWidth, marginLeft+marginRight, horizontalExtra)
	box.Width = box.ContentWidth + horizontalExtra
	box.X = containingX + marginLeft
	box.ContentX = box.X + borderLeft + paddingLeft
	box.Y = containingY + marginTop
	box.ContentY = box.Y + borderTop + paddingTop

	hasBlockChildren := false
	for _, child := range box.Children {
		if child.IsBlockLevel() {
			hasBlockChildren = true
			break
		}
	}
	if hasBlockChildren {
		layout.layoutBlockChildren(box)
	} else {
		layout.layoutInlineChildren(box)
	}

	verticalExtra := paddingTop + paddingBottom + borderTop + borderBottom
	contentHeight := resolveHeight(box.Style, box.ContentHeight, containingHeight, verticalExtra)
	box.ContentHeight = contentHeight
	box.Height = contentHeight + verticalExtra
}

func (layout *Layout) layoutBlockChildren(box *Box) {
	cursorY := box.ContentY
	previousBottom := 0.0
	first := true
	for _, child := range box.Children {
		if !child.IsBlockLevel() {
			continue
		}
		marginTop := child.Margin.Top.Resolve(box.ContentWidth)
		collapsed := marginTop
		if !first {
			collapsed = collapseMargins(previousBottom, marginTop)
		}
		containingY := cursorY + collapsed - marginTop
		layout.layoutBlock(child, box.ContentX, containingY, box.ContentWidth, 0)
		cursorY = child.Y + child.Height
		previousBottom = child.Margin.Bottom.Resolve(box.ContentWidth)
		first = false
	}
	if !first {
		cursorY += previousBottom
	}
	box.ContentHeight = math.Max(0, cursorY-box.ContentY)
}

func (layout *Layout) layoutInlineChildren(box *Box) {
	items := []*Box{}
	for _, child := range box.Children {
		collectInlineBoxes(child, &items)
	}
	lineX := box.ContentX
	lineY := box.ContentY
	maxLineHeight := 0.0
	wordBoxes := make([]*Box, 0, len(items))
	for _, item := range items {
		width, height := layout.measureText(item.Text, item.Style)
		space := 0.0
		if len(wordBoxes) > 0 {
			space = item.FontSize * 0.25
		}
		if lineX > box.ContentX && lineX+space+width > box.ContentX+box.ContentWidth {
			lineY += maxLineHeight
			lineX = box.ContentX
			maxLineHeight = 0
			space = 0
		}
		lineX += space
		item.X = lineX
		item.Y = lineY
		item.Width = width
		item.Height = height
		lineX += width
		if height > maxLineHeight {
			maxLineHeight = height
		}
		wordBoxes = append(wordBoxes, item)
	}
	if len(wordBoxes) > 0 {
		lineY += maxLineHeight
		box.Children = wordBoxes
		for _, word := range wordBoxes {
			word.Parent = box
		}
	}
	box.ContentHeight = math.Max(0, lineY-box.ContentY)
}

func collectInlineBoxes(box *Box, result *[]*Box) {
	if box == nil {
		return
	}
	if box.Kind == TextBox {
		words := strings.Fields(box.Text)
		for _, word := range words {
			*result = append(*result, &Box{
				Node:       box.Node,
				Style:      box.Style,
				Kind:       TextBox,
				Text:       word,
				FontSize:   box.FontSize,
				LineHeight: box.LineHeight,
				Parent:     box.Parent,
			})
		}
		return
	}
	for _, child := range box.Children {
		collectInlineBoxes(child, result)
	}
}

func (layout *Layout) measureText(text string, style *styles.Style) (float64, float64) {
	if layout.MeasureText != nil {
		return layout.MeasureText(text, style)
	}
	fontSize := 16.0
	lineHeight := 19.2
	if style != nil {
		if value, ok := parseComputedPixels(style.GetString("font-size")); ok && value > 0 {
			fontSize = value
		}
		lineHeight = fontSize * 1.2
		if value, ok := parseComputedPixels(style.GetString("line-height")); ok && value > 0 {
			lineHeight = value
		} else if value, ok := parseComputedNumber(style.GetString("line-height")); ok && value > 0 {
			lineHeight = fontSize * value
		}
	}
	return float64(len([]rune(text))) * fontSize * 0.5, lineHeight
}

func resolveWidth(style *styles.Style, containingWidth, marginHorizontal, paddingBorderHorizontal float64) float64 {
	size := parseSize(style.GetString("width"))
	borderBox := strings.EqualFold(strings.TrimSpace(style.GetString("box-sizing")), "border-box")
	contentWidth := 0.0
	switch {
	case size.Auto || size.Keyword != "":
		contentWidth = containingWidth - marginHorizontal - paddingBorderHorizontal
	case size.Percent:
		contentWidth = containingWidth * size.Value / 100
		if borderBox {
			contentWidth -= paddingBorderHorizontal
		}
	default:
		contentWidth = size.Value
		if borderBox {
			contentWidth -= paddingBorderHorizontal
		}
	}
	contentWidth = clampSize(style.GetString("min-width"), contentWidth, containingWidth, paddingBorderHorizontal, borderBox, true)
	contentWidth = clampSize(style.GetString("max-width"), contentWidth, containingWidth, paddingBorderHorizontal, borderBox, false)
	return math.Max(0, contentWidth)
}

func resolveHeight(style *styles.Style, contentHeight, containingHeight, paddingBorderVertical float64) float64 {
	size := parseSize(style.GetString("height"))
	borderBox := strings.EqualFold(strings.TrimSpace(style.GetString("box-sizing")), "border-box")
	switch {
	case size.Auto || size.Keyword != "":
		return contentHeight
	case size.Percent:
		if containingHeight <= 0 {
			return contentHeight
		}
		value := containingHeight * size.Value / 100
		if borderBox {
			value -= paddingBorderVertical
		}
		return math.Max(0, value)
	default:
		value := size.Value
		if borderBox {
			value -= paddingBorderVertical
		}
		return math.Max(0, value)
	}
}

func clampSize(text string, current, containingWidth, paddingBorder float64, borderBox, minimum bool) float64 {
	size := parseSize(text)
	if size.Auto || size.Keyword != "" {
		return current
	}
	value := size.Value
	if size.Percent {
		value = containingWidth * size.Value / 100
	}
	if borderBox {
		value -= paddingBorder
	}
	if minimum {
		return math.Max(current, value)
	}
	return math.Min(current, value)
}

func parseSize(text string) Size {
	text = strings.TrimSpace(text)
	if text == "" || strings.EqualFold(text, "auto") || strings.EqualFold(text, "none") {
		return Size{Auto: true}
	}
	switch strings.ToLower(text) {
	case "min-content", "max-content", "fit-content", "stretch":
		return Size{Keyword: strings.ToLower(text)}
	}
	values := parser.ParseComponentValueList(text)
	if len(values) != 1 {
		return Size{Auto: true}
	}
	numeric, ok := cssvalues.ParseNumeric(values[0])
	if !ok {
		return Size{Auto: true}
	}
	if numeric.Unit == "%" {
		return Size{Value: numeric.Number * 100, Percent: true}
	}
	if numeric.Kind == cssvalues.Length || (numeric.Kind == cssvalues.Other && numeric.Unit == "") {
		return Size{Value: numeric.Number}
	}
	return Size{Auto: true}
}

func parseComputedPixels(text string) (float64, bool) {
	values := parser.ParseComponentValueList(text)
	if len(values) != 1 {
		return 0, false
	}
	numeric, ok := cssvalues.ParseNumeric(values[0])
	if !ok || numeric.Kind != cssvalues.Length || !strings.EqualFold(numeric.Unit, "px") {
		return 0, false
	}
	return numeric.Number, true
}

func parseComputedNumber(text string) (float64, bool) {
	values := parser.ParseComponentValueList(text)
	if len(values) != 1 {
		return 0, false
	}
	numeric, ok := cssvalues.ParseNumeric(values[0])
	if !ok || numeric.Kind != cssvalues.Other || numeric.Unit != "" {
		return 0, false
	}
	return numeric.Number, true
}

func collapseMargins(first, second float64) float64 {
	if first >= 0 && second >= 0 {
		return math.Max(first, second)
	}
	if first <= 0 && second <= 0 {
		return math.Min(first, second)
	}
	return first + second
}
