package layouts

import (
	"math"
	"strings"
)

func floatOf(box *Box) string {
	if box == nil || box.Style == nil {
		return ""
	}
	value := strings.ToLower(strings.TrimSpace(box.Style.GetString("float")))
	switch value {
	case "left", "right", "inline-start", "inline-end":
		return value
	default:
		return ""
	}
}

func isFloat(box *Box) bool {
	return floatOf(box) != ""
}

func (layout *Layout) layoutFloat(box *Box, containingX, containingY, containingWidth float64) float64 {
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
	widthSize := parseSize(box.Style.GetString("width"))
	if widthSize.Auto || widthSize.Keyword != "" {
		layout.computeIntrinsicWidths(box)
		available := containingWidth - marginLeft - marginRight - horizontalExtra
		box.ContentWidth = math.Max(box.IntrinsicMinWidth, math.Min(available, box.IntrinsicMaxWidth))
	} else {
		box.ContentWidth = layout.resolveWidth(box, containingWidth, marginLeft+marginRight, horizontalExtra)
	}
	box.Width = box.ContentWidth + horizontalExtra
	side := floatOf(box)
	if side == "right" || side == "inline-end" {
		box.X = containingX + containingWidth - marginRight - box.Width
	} else {
		box.X = containingX + marginLeft
	}
	box.Y = containingY + marginTop
	box.ContentX = box.X + borderLeft + paddingLeft
	box.ContentY = box.Y + borderTop + paddingTop
	layout.layoutFlexItemContent(box)
	box.ContentHeight = resolveHeight(box.Style, box.ContentHeight, 0, verticalExtra)
	box.Height = box.ContentHeight + verticalExtra
	return box.Y + box.Height + marginBottom
}
