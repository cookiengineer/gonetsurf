package layouts

import (
	"math"
	"strconv"
	"strings"

	"gonetsurf/dom"
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
	// FitArgument is the optional argument of fit-content(<length-percentage>).
	FitArgument    float64
	FitPercent     bool
	FitHasArgument bool
}

// Layout performs layout for a box tree.
type Layout struct {
	ViewportWidth  float64
	ViewportHeight float64
	// MeasureText overrides the default simple advance-width measurement.
	MeasureText func(text string, style *styles.Style) (width, height float64)
	// Shaper provides glyph advances when MeasureText is not set.
	Shaper TextShaper
	// Metrics overrides the default font metrics when set.
	Metrics func(style *styles.Style) FontMetrics
	// Dirty restricts incremental layout to nodes needing reflow. A nil map
	// means full layout.
	Dirty map[*dom.Node]bool

	positioned []*Box
}

// New creates a layout context for a viewport.
func New(viewportWidth, viewportHeight float64) *Layout {
	return &Layout{ViewportWidth: viewportWidth, ViewportHeight: viewportHeight, Shaper: SimpleShaper{}}
}

func (layout *Layout) metricsForStyle(style *styles.Style) FontMetrics {
	if layout.Metrics != nil {
		return layout.Metrics(style)
	}
	return MetricsForStyle(style)
}

// Layout lays out root in the viewport.
func (layout *Layout) Layout(root *Box) {
	if root == nil {
		return
	}
	layout.positioned = nil
	layout.layoutBlock(root, 0, 0, layout.ViewportWidth, layout.ViewportHeight)
	layout.layoutPositioned()
	layout.applyRelativeOffsets(root)
	layout.computeOverflow(root)
}

func (layout *Layout) layoutBlock(box *Box, containingX, containingY, containingWidth, containingHeight float64) {
	if box == nil || box.Style == nil {
		return
	}
	box.LaidOut = false
	box.ContainingX = containingX
	box.ContainingY = containingY
	box.ContainingWidth = containingWidth
	box.ContainingHeight = containingHeight
	if isOutOfFlow(box) {
		box.StaticX = containingX
		box.StaticY = containingY
		box.HasStaticPosition = true
		layout.positioned = append(layout.positioned, box)
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
	box.ContentWidth = layout.resolveWidth(box, containingWidth, marginLeft+marginRight, horizontalExtra)
	box.Width = box.ContentWidth + horizontalExtra
	box.X = containingX + marginLeft
	box.ContentX = box.X + borderLeft + paddingLeft
	verticalExtra := paddingTop + paddingBottom + borderTop + borderBottom
	specifiedHeight, hasSpecifiedHeight := layout.specifiedContentHeight(box, containingHeight, verticalExtra)
	if hasSpecifiedHeight {
		box.ContentHeight = specifiedHeight
	} else if ratio, ok := parseAspectRatio(box.Style.GetString("aspect-ratio")); ok && box.ContentWidth > 0 {
		box.ContentHeight = box.ContentWidth / ratio
	}
	firstChild := firstBlockLevelChild(box)
	lastChild := lastBlockLevelChild(box)
	blockFormattingContext := establishesBlockFormattingContext(box)
	collapseTop := box.Kind == BlockBox && firstChild != nil && firstChild.Kind == BlockBox &&
		borderTop == 0 && paddingTop == 0 && !blockFormattingContext
	collapseBottom := box.Kind == BlockBox && lastChild != nil && lastChild.Kind == BlockBox &&
		borderBottom == 0 && paddingBottom == 0 && !blockFormattingContext &&
		parseSize(box.Style.GetString("height")).Auto
	effectiveMarginTop := marginTop
	if collapseTop {
		childTop := firstChild.Margin.Top.Resolve(containingWidth)
		effectiveMarginTop = collapseMargins(marginTop, childTop)
	}
	box.Y = containingY + effectiveMarginTop
	box.ContentY = box.Y + borderTop + paddingTop

	if isGridContainer(box) {
		layout.layoutGrid(box)
	} else if isTableContainer(box) {
		layout.layoutTable(box)
	} else if isFlexContainer(box) {
		if hasSpecifiedHeight {
			box.ContentHeight = specifiedHeight
		}
		layout.layoutFlex(box)
	} else if isVerticalWritingMode(box) {
		if firstChild != nil {
			layout.layoutVerticalBlockChildren(box)
		} else {
			layout.layoutInlineChildren(box)
		}
	} else if firstChild != nil {
		childContainingHeight := 0.0
		if hasSpecifiedHeight {
			childContainingHeight = box.ContentHeight
		}
		layout.layoutBlockChildren(box, collapseTop, collapseBottom, childContainingHeight)
	} else {
		layout.layoutInlineChildren(box)
	}
	if collapseBottom {
		childBottom := lastChild.Margin.Bottom.Resolve(box.ContentWidth)
		parentBottom := box.Margin.Bottom.Resolve(box.ContentWidth)
		box.Margin.Bottom = Length{Value: collapseMargins(parentBottom, childBottom)}
	}

	contentHeight := layout.resolveContentHeight(box, box.ContentHeight, containingHeight, verticalExtra)
	box.ContentHeight = contentHeight
	box.Height = contentHeight + verticalExtra
	box.LaidOut = true
}

func (layout *Layout) layoutBlockChildren(box *Box, collapseFirstTop, collapseLastBottom bool, containingHeight float64) {
	cursorY := box.ContentY
	pendingMargin := 0.0
	first := true
	floatBottom := 0.0
	floats := []*Box{}
	lastChild := lastBlockLevelChild(box)
	for _, child := range box.Children {
		if isOutOfFlow(child) {
			child.StaticX = box.ContentX
			child.StaticY = cursorY
			child.HasStaticPosition = true
			layout.positioned = append(layout.positioned, child)
			continue
		}
		if isFloat(child) {
			layout.layoutFloat(child, box.ContentX, cursorY, box.ContentWidth)
			positionFloat(child, floats, box.ContentX, box.ContentWidth)
			floats = append(floats, child)
			if bottom := floatOuterBottom(child); bottom > floatBottom {
				floatBottom = bottom
			}
			continue
		}
		if !child.IsBlockLevel() {
			continue
		}
		if clear := clearOf(child); clear != "" {
			if required := clearY(floats, clear); required > cursorY {
				cursorY = required
				floatBottom = required
			}
		}
		if floatBottom > cursorY {
			cursorY = floatBottom
		}
		marginTop := child.Margin.Top.Resolve(box.ContentWidth)
		collapsed := marginTop
		if first {
			if collapseFirstTop {
				collapsed = 0
			}
		} else {
			collapsed = collapseMargins(pendingMargin, marginTop)
		}
		containingY := cursorY + collapsed - marginTop
		if layout.canReuseChild(child, box, containingY, containingHeight) {
			deltaX := box.ContentX - child.ContainingX
			deltaY := containingY - child.ContainingY
			if deltaX != 0 || deltaY != 0 {
				shiftBox(child, deltaX, deltaY)
			}
			child.ContainingX = box.ContentX
			child.ContainingY = containingY
			first = false
			cursorY = child.Y + child.Height
			pendingMargin = child.Margin.Bottom.Resolve(box.ContentWidth)
			if collapseLastBottom && child == lastChild {
				pendingMargin = 0
			}
			continue
		}
		layout.layoutBlock(child, box.ContentX, containingY, box.ContentWidth, containingHeight)
		first = false
		marginBottom := child.Margin.Bottom.Resolve(box.ContentWidth)
		if isSelfCollapsingBlock(child) {
			pendingMargin = collapseMargins(collapsed, marginBottom)
			continue
		}
		cursorY = child.Y + child.Height
		pendingMargin = marginBottom
		if collapseLastBottom && child == lastChild {
			pendingMargin = 0
		}
	}
	if !first {
		cursorY += pendingMargin
	}
	if floatBottom > cursorY {
		cursorY = floatBottom
	}
	box.ContentHeight = math.Max(0, cursorY-box.ContentY)
}

// isSelfCollapsingBlock reports whether a block's own top and bottom margins
// collapse through it. See CSS 2.2 §8.3.1.
func isSelfCollapsingBlock(box *Box) bool {
	if box == nil || box.Kind != BlockBox {
		return false
	}
	height := parseSize(box.Style.GetString("height"))
	if !height.Auto && height.Keyword == "" {
		return false
	}
	if box.Border.Top.Resolve(box.ContentWidth) != 0 || box.Border.Bottom.Resolve(box.ContentWidth) != 0 {
		return false
	}
	if box.Padding.Top.Resolve(box.ContentWidth) != 0 || box.Padding.Bottom.Resolve(box.ContentWidth) != 0 {
		return false
	}
	for _, child := range box.Children {
		if isOutOfFlow(child) || isFloat(child) {
			continue
		}
		return false
	}
	return true
}

func firstBlockLevelChild(box *Box) *Box {
	for _, child := range box.Children {
		if child.IsBlockLevel() {
			return child
		}
	}
	return nil
}

func lastBlockLevelChild(box *Box) *Box {
	for index := len(box.Children) - 1; index >= 0; index-- {
		if box.Children[index].IsBlockLevel() {
			return box.Children[index]
		}
	}
	return nil
}

func (layout *Layout) isDirtyBox(box *Box) bool {
	if layout.Dirty == nil {
		return true
	}
	if box == nil || box.Node == nil {
		return true
	}
	return layout.Dirty[box.Node]
}

func (layout *Layout) canReuseChild(child *Box, parent *Box, containingY, containingHeight float64) bool {
	if layout.Dirty == nil || child == nil || !child.LaidOut {
		return false
	}
	if layout.isDirtyBox(child) || isOutOfFlow(child) || isFloat(child) {
		return false
	}
	if child.ContainingWidth != parent.ContentWidth || child.ContainingHeight != containingHeight {
		return false
	}
	if child.ContainingX != parent.ContentX {
		return false
	}
	return true
}

func establishesBlockFormattingContext(box *Box) bool {
	display := strings.ToLower(strings.TrimSpace(box.Style.GetString("display")))
	switch display {
	case "flow-root", "inline-block", "inline-flex", "inline-grid", "inline-table", "table", "flex", "grid":
		return true
	}
	switch strings.ToLower(strings.TrimSpace(box.Style.GetString("position"))) {
	case "absolute", "fixed":
		return true
	}
	switch strings.ToLower(strings.TrimSpace(box.Style.GetString("float"))) {
	case "left", "right", "inline-start", "inline-end":
		return true
	}
	for _, property := range []string{"overflow-x", "overflow-y"} {
		value := strings.ToLower(strings.TrimSpace(box.Style.GetString(property)))
		if value != "" && value != "visible" {
			return true
		}
	}
	return false
}

func (layout *Layout) layoutInlineChildren(box *Box) {
	if isVerticalWritingMode(box) {
		layout.layoutVerticalInlineChildren(box)
		return
	}
	layout.layoutInlineFormatting(box)
}

func collectInlineBoxes(box *Box, result *[]*Box) {
	if box == nil {
		return
	}
	if box.Kind == TextBox {
		direction := ""
		if box.Style != nil {
			direction = box.Style.GetString("direction")
		}
		words := visualWords(box.Text, direction)
		for _, word := range words {
			*result = append(*result, &Box{
				Node:       box.Node,
				Style:      box.Style,
				Kind:       TextBox,
				Text:       word.Text,
				VisualText: word.Visual,
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

func parseOffset(text string) Length {
	if strings.TrimSpace(text) == "" {
		return Length{Auto: true}
	}
	return parseLength(text)
}

func isOutOfFlow(box *Box) bool {
	if box == nil || box.Style == nil {
		return false
	}
	position := strings.ToLower(strings.TrimSpace(box.Style.GetString("position")))
	return position == "absolute" || position == "fixed"
}

func (layout *Layout) measureText(text string, style *styles.Style) (float64, float64) {
	if layout.MeasureText != nil {
		return layout.MeasureText(text, style)
	}
	glyphs := layout.shapeText(text, style)
	width := 0.0
	for _, glyph := range glyphs {
		width += glyph.Advance
	}
	_, lineHeight := fontMetrics(style)
	if lineHeight <= 0 {
		lineHeight = 19.2
	}
	return width, lineHeight
}

func (layout *Layout) resolveWidth(box *Box, containingWidth, marginHorizontal, paddingBorderHorizontal float64) float64 {
	style := box.Style
	size := parseSize(style.GetString("width"))
	borderBox := strings.EqualFold(strings.TrimSpace(style.GetString("box-sizing")), "border-box")
	contentWidth := 0.0
	switch {
	case size.Keyword != "":
		layout.computeIntrinsicWidths(box)
		available := containingWidth - marginHorizontal - paddingBorderHorizontal
		switch size.Keyword {
		case "min-content":
			contentWidth = box.IntrinsicMinWidth
		case "max-content":
			contentWidth = box.IntrinsicMaxWidth
		case "fit-content":
			if size.FitHasArgument {
				argument := size.Value
				if size.FitPercent {
					argument = containingWidth * size.Value / 100
				}
				contentWidth = math.Max(box.IntrinsicMinWidth, math.Min(argument, box.IntrinsicMaxWidth))
			} else {
				contentWidth = math.Max(box.IntrinsicMinWidth, math.Min(available, box.IntrinsicMaxWidth))
			}
		case "stretch":
			contentWidth = available
		default:
			contentWidth = math.Max(box.IntrinsicMinWidth, math.Min(available, box.IntrinsicMaxWidth))
		}
	case size.Auto:
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
	contentWidth = layout.clampSize(box, style.GetString("min-width"), contentWidth, containingWidth, paddingBorderHorizontal, borderBox, true)
	contentWidth = layout.clampSize(box, style.GetString("max-width"), contentWidth, containingWidth, paddingBorderHorizontal, borderBox, false)
	return math.Max(0, contentWidth)
}

func (layout *Layout) computeIntrinsicWidths(box *Box) {
	if box == nil || box.intrinsicsComputed {
		return
	}
	box.intrinsicsComputed = true
	if box.Kind == TextBox {
		words := strings.Fields(box.Text)
		minimum := 0.0
		total := 0.0
		for index, word := range words {
			width, _ := layout.measureText(word, box.Style)
			if width > minimum {
				minimum = width
			}
			total += width
			if index > 0 {
				total += box.FontSize * 0.25
			}
		}
		box.IntrinsicMinWidth = minimum
		box.IntrinsicMaxWidth = total
		return
	}
	size := parseSize(box.Style.GetString("width"))
	if !size.Auto && !size.Percent && size.Keyword == "" {
		box.IntrinsicMinWidth = size.Value
		box.IntrinsicMaxWidth = size.Value
		return
	}
	hasBlock := false
	for _, child := range box.Children {
		layout.computeIntrinsicWidths(child)
		if child.IsBlockLevel() {
			hasBlock = true
		}
	}
	if hasBlock {
		minimum := 0.0
		maximum := 0.0
		for _, child := range box.Children {
			if child.IntrinsicMinWidth > minimum {
				minimum = child.IntrinsicMinWidth
			}
			if child.IntrinsicMaxWidth > maximum {
				maximum = child.IntrinsicMaxWidth
			}
		}
		box.IntrinsicMinWidth = minimum
		box.IntrinsicMaxWidth = maximum
		return
	}
	minimum := 0.0
	maximum := 0.0
	for index, child := range box.Children {
		if child.IntrinsicMinWidth > minimum {
			minimum = child.IntrinsicMinWidth
		}
		maximum += child.IntrinsicMaxWidth
		if index > 0 {
			maximum += child.FontSize * 0.25
		}
	}
	box.IntrinsicMinWidth = minimum
	box.IntrinsicMaxWidth = maximum
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

// specifiedContentHeight resolves a definite height. Auto heights and keyword
// sizes are not definite for percentage resolution in descendants.
func (layout *Layout) specifiedContentHeight(box *Box, containingHeight, verticalExtra float64) (float64, bool) {
	size := parseSize(box.Style.GetString("height"))
	if size.Auto || size.Keyword != "" {
		return 0, false
	}
	if size.Percent && containingHeight <= 0 {
		return 0, false
	}
	value := resolveHeight(box.Style, 0, containingHeight, verticalExtra)
	value = layout.clampHeight(box, value, containingHeight, verticalExtra)
	return value, true
}

// resolveContentHeight resolves the used content height, including the
// preferred height from aspect-ratio and min/max height clamping.
func (layout *Layout) resolveContentHeight(box *Box, contentHeight, containingHeight, verticalExtra float64) float64 {
	heightSize := parseSize(box.Style.GetString("height"))
	if heightSize.Auto && box.ContentWidth > 0 {
		if ratio, ok := parseAspectRatio(box.Style.GetString("aspect-ratio")); ok {
			contentHeight = box.ContentWidth / ratio
		}
	}
	contentHeight = resolveHeight(box.Style, contentHeight, containingHeight, verticalExtra)
	return layout.clampHeight(box, contentHeight, containingHeight, verticalExtra)
}

func (layout *Layout) clampHeight(box *Box, current, containingHeight, paddingBorderVertical float64) float64 {
	borderBox := strings.EqualFold(strings.TrimSpace(box.Style.GetString("box-sizing")), "border-box")
	resolve := func(text string) (float64, bool) {
		size := parseSize(text)
		if size.Auto || size.Keyword != "" {
			return 0, false
		}
		value := size.Value
		if size.Percent {
			if containingHeight <= 0 {
				return 0, false
			}
			value = containingHeight * size.Value / 100
		}
		if borderBox {
			value -= paddingBorderVertical
		}
		return math.Max(0, value), true
	}
	if value, ok := resolve(box.Style.GetString("min-height")); ok {
		current = math.Max(current, value)
	}
	if value, ok := resolve(box.Style.GetString("max-height")); ok {
		current = math.Min(current, value)
	}
	return math.Max(0, current)
}

func parseAspectRatio(text string) (float64, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, false
	}
	for _, prefix := range []string{"auto ", "auto/"} {
		text = strings.TrimSpace(strings.TrimPrefix(text, prefix))
	}
	parts := strings.Split(text, "/")
	if len(parts) != 2 {
		return 0, false
	}
	width, ok := parseRatioNumber(parts[0])
	if !ok {
		return 0, false
	}
	height, ok := parseRatioNumber(parts[1])
	if !ok || width <= 0 || height <= 0 {
		return 0, false
	}
	return width / height, true
}

func parseRatioNumber(text string) (float64, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, false
	}
	value := 0.0
	seenDigit := false
	for _, character := range text {
		if character == '.' {
			if !seenDigit {
				return 0, false
			}
			continue
		}
		if character < '0' || character > '9' {
			return 0, false
		}
		seenDigit = true
		value = value*10 + float64(character-'0')
	}
	if !seenDigit {
		return 0, false
	}
	// Re-parse with strconv semantics when a decimal point is present.
	if strings.Contains(text, ".") {
		parsed, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	}
	return value, true
}

func (layout *Layout) clampSize(box *Box, text string, current, containingWidth, paddingBorder float64, borderBox, minimum bool) float64 {
	size := parseSize(text)
	if size.Auto {
		return current
	}
	value := 0.0
	if size.Keyword != "" {
		layout.computeIntrinsicWidths(box)
		switch size.Keyword {
		case "min-content":
			value = box.IntrinsicMinWidth
		case "max-content":
			value = box.IntrinsicMaxWidth
		default:
			value = math.Max(box.IntrinsicMinWidth, math.Min(current, box.IntrinsicMaxWidth))
		}
	} else {
		value = size.Value
		if size.Percent {
			value = containingWidth * size.Value / 100
		}
		if borderBox {
			value -= paddingBorder
		}
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
	if function, ok := values[0].(*parser.Function); ok && strings.EqualFold(function.Name, "fit-content") {
		argument, ok := parseFitContentArgument(function.Value)
		if !ok {
			return Size{Auto: true}
		}
		argument.Keyword = "fit-content"
		argument.FitHasArgument = true
		return argument
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

func parseFitContentArgument(values []parser.ComponentValue) (Size, bool) {
	for _, value := range values {
		if parser.IsWhitespace(value) {
			continue
		}
		numeric, ok := cssvalues.ParseNumeric(value)
		if !ok {
			return Size{}, false
		}
		if numeric.Unit == "%" {
			return Size{Value: numeric.Number * 100, Percent: true}, true
		}
		if numeric.Kind == cssvalues.Length || (numeric.Kind == cssvalues.Other && numeric.Unit == "") {
			return Size{Value: numeric.Number}, true
		}
		return Size{}, false
	}
	return Size{}, false
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

func (layout *Layout) layoutPositioned() {
	for index := 0; index < len(layout.positioned); index++ {
		box := layout.positioned[index]
		position := strings.ToLower(strings.TrimSpace(box.Style.GetString("position")))
		cbX, cbY, cbWidth, cbHeight := 0.0, 0.0, layout.ViewportWidth, layout.ViewportHeight
		if position != "fixed" {
			if containing := positionedAncestor(box); containing != nil {
				cbX = containing.X + containing.Border.Left.Resolve(containing.Width)
				cbY = containing.Y + containing.Border.Top.Resolve(containing.Width)
				cbWidth = containing.ContentWidth + containing.Padding.Horizontal(containing.Width)
				cbHeight = containing.ContentHeight + containing.Padding.Vertical(containing.Width)
			}
		}
		layout.layoutOutOfFlow(box, cbX, cbY, cbWidth, cbHeight)
	}
}

func (layout *Layout) layoutOutOfFlow(box *Box, containingX, containingY, containingWidth, containingHeight float64) {
	if box == nil || box.Style == nil {
		return
	}
	marginLeft := box.Margin.Left.Resolve(containingWidth)
	marginRight := box.Margin.Right.Resolve(containingWidth)
	marginTop := box.Margin.Top.Resolve(containingWidth)
	marginBottom := box.Margin.Bottom.Resolve(containingWidth)
	paddingLeft := box.Padding.Left.Resolve(containingWidth)
	paddingRight := box.Padding.Right.Resolve(containingWidth)
	paddingTop := box.Padding.Top.Resolve(containingWidth)
	paddingBottom := box.Padding.Bottom.Resolve(containingWidth)
	borderLeft := box.Border.Left.Resolve(containingWidth)
	borderRight := box.Border.Right.Resolve(containingWidth)
	borderTop := box.Border.Top.Resolve(containingWidth)
	borderBottom := box.Border.Bottom.Resolve(containingWidth)
	horizontalExtra := paddingLeft + paddingRight + borderLeft + borderRight
	verticalExtra := paddingTop + paddingBottom + borderTop + borderBottom
	left := parseOffset(box.Style.GetString("left"))
	right := parseOffset(box.Style.GetString("right"))
	top := parseOffset(box.Style.GetString("top"))
	bottom := parseOffset(box.Style.GetString("bottom"))
	heightSize := parseSize(box.Style.GetString("height"))
	widthSize := parseSize(box.Style.GetString("width"))
	if widthSize.Auto && !left.Auto && !right.Auto {
		box.ContentWidth = math.Max(0, containingWidth-left.Resolve(containingWidth)-right.Resolve(containingWidth)-marginLeft-marginRight-horizontalExtra)
	} else if widthSize.Auto {
		layout.computeIntrinsicWidths(box)
		available := containingWidth - marginLeft - marginRight - horizontalExtra
		if !left.Auto {
			available -= left.Resolve(containingWidth)
		}
		if !right.Auto {
			available -= right.Resolve(containingWidth)
		}
		box.ContentWidth = math.Max(box.IntrinsicMinWidth, math.Min(available, box.IntrinsicMaxWidth))
	} else {
		box.ContentWidth = layout.resolveWidth(box, containingWidth, marginLeft+marginRight, horizontalExtra)
	}
	box.Width = box.ContentWidth + horizontalExtra
	if !left.Auto {
		box.X = containingX + left.Resolve(containingWidth) + marginLeft
	} else if !right.Auto {
		box.X = containingX + containingWidth - right.Resolve(containingWidth) - box.Width - marginRight
	} else if box.HasStaticPosition {
		box.X = box.StaticX + marginLeft
	} else {
		box.X = containingX + marginLeft
	}
	if !top.Auto {
		box.Y = containingY + top.Resolve(containingWidth) + marginTop
	} else if box.HasStaticPosition && bottom.Auto {
		box.Y = box.StaticY + marginTop
	} else {
		box.Y = containingY + marginTop
	}
	box.ContentX = box.X + borderLeft + paddingLeft
	box.ContentY = box.Y + borderTop + paddingTop

	hasBlockChildren := false
	for _, child := range box.Children {
		if child.IsBlockLevel() {
			hasBlockChildren = true
			break
		}
	}
	if hasBlockChildren {
		layout.layoutBlockChildren(box, false, false, 0)
	} else {
		layout.layoutInlineChildren(box)
	}
	contentHeight := box.ContentHeight
	if heightSize.Auto && !top.Auto && !bottom.Auto {
		contentHeight = math.Max(0, containingHeight-top.Resolve(containingWidth)-bottom.Resolve(containingWidth)-marginTop-marginBottom-verticalExtra)
	} else {
		contentHeight = layout.resolveContentHeight(box, contentHeight, containingHeight, verticalExtra)
	}
	box.ContentHeight = contentHeight
	box.Height = contentHeight + verticalExtra
	if top.Auto && !bottom.Auto {
		desiredY := containingY + containingHeight - bottom.Resolve(containingWidth) - box.Height - marginBottom
		deltaY := desiredY - box.Y
		if deltaY != 0 {
			shiftBox(box, 0, deltaY)
		}
	}
}

func positionedAncestor(box *Box) *Box {
	for parent := box.Parent; parent != nil; parent = parent.Parent {
		position := strings.ToLower(strings.TrimSpace(parent.Style.GetString("position")))
		if position != "" && position != "static" {
			return parent
		}
	}
	return nil
}

func (layout *Layout) applyRelativeOffsets(box *Box) {
	if box == nil {
		return
	}
	position := strings.ToLower(strings.TrimSpace(box.Style.GetString("position")))
	if position == "relative" {
		left := parseOffset(box.Style.GetString("left"))
		right := parseOffset(box.Style.GetString("right"))
		top := parseOffset(box.Style.GetString("top"))
		bottom := parseOffset(box.Style.GetString("bottom"))
		width, height := layout.ViewportWidth, layout.ViewportHeight
		if box.Parent != nil {
			width, height = box.Parent.ContentWidth, box.Parent.ContentHeight
		}
		deltaX := 0.0
		if !left.Auto {
			deltaX = left.Resolve(width)
		} else if !right.Auto {
			deltaX = -right.Resolve(width)
		}
		deltaY := 0.0
		if !top.Auto {
			deltaY = top.Resolve(height)
		} else if !bottom.Auto {
			deltaY = -bottom.Resolve(height)
		}
		if deltaX != 0 || deltaY != 0 {
			shiftBox(box, deltaX, deltaY)
		}
	}
	for _, child := range box.Children {
		layout.applyRelativeOffsets(child)
	}
}

func shiftBox(box *Box, deltaX, deltaY float64) {
	if box == nil {
		return
	}
	box.X += deltaX
	box.Y += deltaY
	box.ContentX += deltaX
	box.ContentY += deltaY
	for _, child := range box.Children {
		shiftBox(child, deltaX, deltaY)
	}
}

func (layout *Layout) computeOverflow(box *Box) (float64, float64) {
	if box == nil {
		return 0, 0
	}
	maxRight := box.ContentX + box.ContentWidth
	maxBottom := box.ContentY + box.ContentHeight
	for _, child := range box.Children {
		childRight, childBottom := layout.computeOverflow(child)
		if right := child.X + child.Width; right > maxRight {
			maxRight = right
		}
		if bottom := child.Y + child.Height; bottom > maxBottom {
			maxBottom = bottom
		}
		if childRight > maxRight {
			maxRight = childRight
		}
		if childBottom > maxBottom {
			maxBottom = childBottom
		}
	}
	box.ScrollWidth = math.Max(0, maxRight-box.ContentX)
	box.ScrollHeight = math.Max(0, maxBottom-box.ContentY)
	box.HasHorizontalScrollbar = box.ScrollX && (box.ScrollWidth > box.ContentWidth || strings.EqualFold(strings.TrimSpace(box.Style.GetString("overflow-x")), "scroll"))
	box.HasVerticalScrollbar = box.ScrollY && (box.ScrollHeight > box.ContentHeight || strings.EqualFold(strings.TrimSpace(box.Style.GetString("overflow-y")), "scroll"))
	return maxRight, maxBottom
}

func isVerticalWritingMode(box *Box) bool {
	if box == nil || box.Style == nil {
		return false
	}
	mode := strings.ToLower(strings.TrimSpace(box.Style.GetString("writing-mode")))
	return strings.HasPrefix(mode, "vertical") || strings.HasPrefix(mode, "sideways")
}

func (layout *Layout) layoutVerticalBlockChildren(box *Box) {
	mode := strings.ToLower(strings.TrimSpace(box.Style.GetString("writing-mode")))
	rightToLeft := strings.HasSuffix(mode, "rl")
	cursorX := box.ContentX
	if rightToLeft {
		cursorX += box.ContentWidth
	}
	maxHeight := 0.0
	for _, child := range box.Children {
		if isOutOfFlow(child) {
			layout.positioned = append(layout.positioned, child)
			continue
		}
		if isFloat(child) {
			continue
		}
		if !child.IsBlockLevel() {
			continue
		}
		layout.layoutBlock(child, box.ContentX, box.ContentY, box.ContentWidth, 0)
		marginStart := child.Margin.Left.Resolve(box.ContentWidth)
		marginEnd := child.Margin.Right.Resolve(box.ContentWidth)
		if rightToLeft {
			marginStart = child.Margin.Right.Resolve(box.ContentWidth)
			marginEnd = child.Margin.Left.Resolve(box.ContentWidth)
		}
		marginTop := child.Margin.Top.Resolve(box.ContentWidth)
		desiredX := cursorX
		if rightToLeft {
			desiredX -= marginStart + child.Width
			cursorX = desiredX - marginEnd
		} else {
			desiredX += marginStart
			cursorX = desiredX + child.Width + marginEnd
		}
		shiftBox(child, desiredX-child.X, box.ContentY+marginTop-child.Y)
		outerHeight := child.Height + marginTop + child.Margin.Bottom.Resolve(box.ContentWidth)
		if outerHeight > maxHeight {
			maxHeight = outerHeight
		}
	}
	box.ContentHeight = maxHeight
}

func (layout *Layout) layoutVerticalInlineChildren(box *Box) {
	items := []*Box{}
	for _, child := range box.Children {
		if isOutOfFlow(child) {
			layout.positioned = append(layout.positioned, child)
			continue
		}
		if isFloat(child) {
			continue
		}
		collectInlineBoxes(child, &items)
	}
	lineCross := box.LineHeight
	if lineCross <= 0 {
		lineCross = box.FontSize * 1.2
	}
	mode := strings.ToLower(strings.TrimSpace(box.Style.GetString("writing-mode")))
	rightToLeft := strings.HasSuffix(mode, "rl")
	lineX := box.ContentX
	if rightToLeft {
		lineX = box.ContentX + box.ContentWidth - lineCross
	}
	available := math.Inf(1)
	if size := parseSize(box.Style.GetString("height")); !size.Auto && !size.Percent && size.Keyword == "" {
		available = size.Value
	}
	cursorY := box.ContentY
	maxHeight := 0.0
	wordBoxes := make([]*Box, 0, len(items))
	for _, item := range items {
		advance, _ := layout.measureText(item.Text, item.Style)
		if cursorY > box.ContentY && cursorY+advance > box.ContentY+available {
			cursorY = box.ContentY
			if rightToLeft {
				lineX -= lineCross
			} else {
				lineX += lineCross
			}
		}
		item.X = lineX
		item.Y = cursorY
		item.Width = lineCross
		item.Height = advance
		item.ContentX = lineX
		item.ContentY = cursorY
		item.ContentWidth = lineCross
		item.ContentHeight = advance
		cursorY += advance
		if cursorY-box.ContentY > maxHeight {
			maxHeight = cursorY - box.ContentY
		}
		wordBoxes = append(wordBoxes, item)
	}
	box.Children = wordBoxes
	for _, word := range wordBoxes {
		word.Parent = box
	}
	box.ContentHeight = maxHeight
}
