package layouts

import (
	"math"
	"sort"
	"strings"

	"gonetsurf/parsers/css/parser"
	"gonetsurf/parsers/css/tokenizer"
)

// flexItem holds the resolved flex values for one child.
type flexItem struct {
	box      *Box
	order    int
	grow     float64
	shrink   float64
	basis    Size
	baseSize float64
	mainSize float64
	margin   float64
}

// isFlexContainer reports whether display establishes a flex formatting
// context.
func isFlexContainer(box *Box) bool {
	display := strings.ToLower(strings.TrimSpace(box.Style.GetString("display")))
	return display == "flex" || display == "inline-flex"
}

func (layout *Layout) layoutFlex(box *Box) {
	direction := strings.ToLower(strings.TrimSpace(box.Style.GetString("flex-direction")))
	if direction == "" {
		direction = "row"
	}
	row := direction == "row" || direction == "row-reverse"
	reverse := strings.HasSuffix(direction, "reverse")
	items := []flexItem{}
	for _, child := range box.Children {
		if isOutOfFlow(child) {
			layout.positioned = append(layout.positioned, child)
			continue
		}
		item := flexItem{
			box:    child,
			order:  parseFlexInteger(child.Style.GetString("order")),
			grow:   parseFlexNumber(child.Style.GetString("flex-grow"), 0),
			shrink: parseFlexNumber(child.Style.GetString("flex-shrink"), 1),
			basis:  parseSize(child.Style.GetString("flex-basis")),
		}
		item.margin = childMarginMain(child, row, box.ContentWidth)
		items = append(items, item)
	}
	sort.SliceStable(items, func(left, right int) bool { return items[left].order < items[right].order })
	if reverse {
		for left, right := 0, len(items)-1; left < right; left, right = left+1, right-1 {
			items[left], items[right] = items[right], items[left]
		}
	}
	if len(items) == 0 {
		box.ContentHeight = 0
		return
	}
	mainGap := layout.flexGap(box, row)
	mainSize := box.ContentWidth
	if !row {
		mainSize = box.ContentHeight
	}
	totalBase := 0.0
	for index := range items {
		items[index].baseSize = layout.flexBaseSize(&items[index], row, mainSize)
		totalBase += items[index].baseSize + items[index].margin
	}
	totalBase += mainGap * float64(len(items)-1)
	if !row && mainSize <= 0 {
		mainSize = totalBase
		box.ContentHeight = mainSize
	}
	free := mainSize - totalBase
	totalGrow := 0.0
	totalShrink := 0.0
	for _, item := range items {
		totalGrow += item.grow
		totalShrink += item.shrink * item.baseSize
	}
	for index := range items {
		items[index].mainSize = items[index].baseSize
		if free > 0 && totalGrow > 0 {
			items[index].mainSize += free * items[index].grow / totalGrow
		} else if free < 0 && totalShrink > 0 {
			items[index].mainSize += free * (items[index].shrink * items[index].baseSize) / totalShrink
			if items[index].mainSize < 0 {
				items[index].mainSize = 0
			}
		}
	}
	used := mainGap * float64(len(items)-1)
	for _, item := range items {
		used += item.mainSize + item.margin
	}
	remaining := mainSize - used
	offset, spacing := justifyOffsets(strings.ToLower(strings.TrimSpace(box.Style.GetString("justify-content"))), remaining, len(items))
	if row {
		layout.layoutFlexRow(box, items, offset, spacing, mainGap)
	} else {
		layout.layoutFlexColumn(box, items, offset, spacing, mainGap)
	}
}

func (layout *Layout) layoutFlexRow(box *Box, items []flexItem, offset, spacing, gap float64) {
	cursor := box.ContentX + offset
	maxHeight := 0.0
	for index := range items {
		item := &items[index]
		marginLeft := item.box.Margin.Left.Resolve(box.ContentWidth)
		marginTop := item.box.Margin.Top.Resolve(box.ContentWidth)
		marginBottom := item.box.Margin.Bottom.Resolve(box.ContentWidth)
		paddingLeft := item.box.Padding.Left.Resolve(box.ContentWidth)
		paddingRight := item.box.Padding.Right.Resolve(box.ContentWidth)
		paddingTop := item.box.Padding.Top.Resolve(box.ContentWidth)
		paddingBottom := item.box.Padding.Bottom.Resolve(box.ContentWidth)
		borderLeft := item.box.Border.Left.Resolve(box.ContentWidth)
		borderRight := item.box.Border.Right.Resolve(box.ContentWidth)
		borderTop := item.box.Border.Top.Resolve(box.ContentWidth)
		borderBottom := item.box.Border.Bottom.Resolve(box.ContentWidth)
		horizontalExtra := paddingLeft + paddingRight + borderLeft + borderRight
		verticalExtra := paddingTop + paddingBottom + borderTop + borderBottom
		item.box.ContentWidth = math.Max(0, item.mainSize)
		item.box.Width = item.box.ContentWidth + horizontalExtra
		item.box.X = cursor + marginLeft
		item.box.ContentX = item.box.X + borderLeft + paddingLeft
		item.box.Y = box.ContentY + marginTop
		item.box.ContentY = item.box.Y + borderTop + paddingTop
		layout.layoutFlexItemContent(item.box)
		item.box.ContentHeight = layout.flexContentHeight(item.box)
		item.box.Height = item.box.ContentHeight + verticalExtra
		outerHeight := item.box.Height + marginTop + marginBottom
		if outerHeight > maxHeight {
			maxHeight = outerHeight
		}
		cursor += item.margin + item.mainSize + gap + spacing
	}
	align := strings.ToLower(strings.TrimSpace(box.Style.GetString("align-items")))
	if align == "" {
		align = "stretch"
	}
	for index := range items {
		item := &items[index]
		alignSelf := strings.ToLower(strings.TrimSpace(item.box.Style.GetString("align-self")))
		if alignSelf == "" || alignSelf == "auto" {
			alignSelf = align
		}
		marginTop := item.box.Margin.Top.Resolve(box.ContentWidth)
		marginBottom := item.box.Margin.Bottom.Resolve(box.ContentWidth)
		switch alignSelf {
		case "flex-end", "end":
			item.box.Y = box.ContentY + maxHeight - marginBottom - item.box.Height
		case "center":
			item.box.Y = box.ContentY + (maxHeight-item.box.Height-marginTop-marginBottom)/2 + marginTop
		case "stretch":
			if parseSize(item.box.Style.GetString("height")).Auto {
				item.box.Height = maxHeight - marginTop - marginBottom
			}
		}
		item.box.ContentY = item.box.Y + item.box.Border.Top.Resolve(box.ContentWidth) + item.box.Padding.Top.Resolve(box.ContentWidth)
	}
	box.ContentHeight = maxHeight
}

func (layout *Layout) layoutFlexColumn(box *Box, items []flexItem, offset, spacing, gap float64) {
	cursor := box.ContentY + offset
	maxWidth := 0.0
	align := strings.ToLower(strings.TrimSpace(box.Style.GetString("align-items")))
	if align == "" {
		align = "stretch"
	}
	for index := range items {
		item := &items[index]
		marginLeft := item.box.Margin.Left.Resolve(box.ContentWidth)
		marginRight := item.box.Margin.Right.Resolve(box.ContentWidth)
		marginTop := item.box.Margin.Top.Resolve(box.ContentWidth)
		paddingLeft := item.box.Padding.Left.Resolve(box.ContentWidth)
		paddingRight := item.box.Padding.Right.Resolve(box.ContentWidth)
		paddingTop := item.box.Padding.Top.Resolve(box.ContentWidth)
		paddingBottom := item.box.Padding.Bottom.Resolve(box.ContentWidth)
		borderLeft := item.box.Border.Left.Resolve(box.ContentWidth)
		borderRight := item.box.Border.Right.Resolve(box.ContentWidth)
		borderTop := item.box.Border.Top.Resolve(box.ContentWidth)
		borderBottom := item.box.Border.Bottom.Resolve(box.ContentWidth)
		horizontalExtra := paddingLeft + paddingRight + borderLeft + borderRight
		verticalExtra := paddingTop + paddingBottom + borderTop + borderBottom
		alignSelf := strings.ToLower(strings.TrimSpace(item.box.Style.GetString("align-self")))
		if alignSelf == "" || alignSelf == "auto" {
			alignSelf = align
		}
		width := 0.0
		if alignSelf == "stretch" {
			width = math.Max(0, box.ContentWidth-marginLeft-marginRight-horizontalExtra)
		} else {
			width = layout.resolveWidth(item.box, box.ContentWidth, marginLeft+marginRight, horizontalExtra)
		}
		item.box.ContentWidth = width
		item.box.Width = width + horizontalExtra
		item.box.X = box.ContentX + marginLeft
		item.box.ContentX = item.box.X + borderLeft + paddingLeft
		item.box.Y = cursor + marginTop
		item.box.ContentY = item.box.Y + borderTop + paddingTop
		layout.layoutFlexItemContent(item.box)
		item.box.ContentHeight = math.Max(0, item.mainSize)
		item.box.Height = item.box.ContentHeight + verticalExtra
		outerWidth := item.box.Width + marginLeft + marginRight
		if outerWidth > maxWidth {
			maxWidth = outerWidth
		}
		cursor += item.margin + item.mainSize + gap + spacing
	}
	for index := range items {
		item := &items[index]
		alignSelf := strings.ToLower(strings.TrimSpace(item.box.Style.GetString("align-self")))
		if alignSelf == "" || alignSelf == "auto" {
			alignSelf = align
		}
		marginLeft := item.box.Margin.Left.Resolve(box.ContentWidth)
		marginRight := item.box.Margin.Right.Resolve(box.ContentWidth)
		switch alignSelf {
		case "flex-end", "end":
			item.box.X = box.ContentX + box.ContentWidth - marginRight - item.box.Width
		case "center":
			item.box.X = box.ContentX + (box.ContentWidth-item.box.Width-marginLeft-marginRight)/2 + marginLeft
		}
		item.box.ContentX = item.box.X + item.box.Border.Left.Resolve(box.ContentWidth) + item.box.Padding.Left.Resolve(box.ContentWidth)
	}
	_ = maxWidth
}

func (layout *Layout) layoutFlexItemContent(box *Box) {
	if box.Kind == TextBox {
		width, height := layout.measureText(box.Text, box.Style)
		box.ContentWidth = width
		box.ContentHeight = height
		return
	}
	hasBlock := false
	for _, child := range box.Children {
		if child.IsBlockLevel() {
			hasBlock = true
			break
		}
	}
	if hasBlock {
		layout.layoutBlockChildren(box, false, false)
	} else {
		layout.layoutInlineChildren(box)
	}
}

func (layout *Layout) flexContentHeight(box *Box) float64 {
	if box.Kind == TextBox {
		_, height := layout.measureText(box.Text, box.Style)
		return height
	}
	return box.ContentHeight
}

func (layout *Layout) flexBaseSize(item *flexItem, row bool, containerMain float64) float64 {
	if !item.basis.Auto && item.basis.Keyword != "content" {
		if item.basis.Percent {
			return containerMain * item.basis.Value / 100
		}
		return item.basis.Value
	}
	if row {
		size := parseSize(item.box.Style.GetString("width"))
		if !size.Auto && size.Keyword == "" {
			if size.Percent {
				return containerMain * size.Value / 100
			}
			return size.Value
		}
		layout.computeIntrinsicWidths(item.box)
		return item.box.IntrinsicMaxWidth
	}
	size := parseSize(item.box.Style.GetString("height"))
	if !size.Auto && size.Keyword == "" {
		if size.Percent {
			return containerMain * size.Value / 100
		}
		return size.Value
	}
	return 0
}

func (layout *Layout) flexGap(box *Box, row bool) float64 {
	property := "row-gap"
	if row {
		property = "column-gap"
	}
	length := parseLength(box.Style.GetString(property))
	if length.Auto {
		return 0
	}
	return length.Resolve(box.ContentWidth)
}

func childMarginMain(box *Box, row bool, containingWidth float64) float64 {
	if row {
		return box.Margin.Left.Resolve(containingWidth) + box.Margin.Right.Resolve(containingWidth)
	}
	return box.Margin.Top.Resolve(containingWidth) + box.Margin.Bottom.Resolve(containingWidth)
}

func justifyOffsets(justify string, remaining float64, count int) (float64, float64) {
	if remaining < 0 {
		remaining = 0
	}
	switch justify {
	case "flex-end", "end":
		return remaining, 0
	case "center":
		return remaining / 2, 0
	case "space-between":
		if count > 1 {
			return 0, remaining / float64(count-1)
		}
	case "space-around":
		spacing := remaining / float64(count)
		return spacing / 2, spacing
	case "space-evenly":
		spacing := remaining / float64(count+1)
		return spacing, spacing
	}
	return 0, 0
}

func parseFlexNumber(text string, fallback float64) float64 {
	values := parser.ParseComponentValueList(text)
	if len(values) == 1 {
		if token, ok := parser.AsToken(values[0]); ok && token.Type == tokenizer.TokenNumber {
			return token.Number
		}
	}
	return fallback
}

func parseFlexInteger(text string) int {
	values := parser.ParseComponentValueList(text)
	if len(values) == 1 {
		if token, ok := parser.AsToken(values[0]); ok && token.Type == tokenizer.TokenNumber && token.NumericKind == tokenizer.Integer {
			return int(token.Number)
		}
	}
	return 0
}
