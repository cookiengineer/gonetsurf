package layouts

import (
	"math"
	"strings"
)

func isTableContainer(box *Box) bool {
	display := displayOf(box)
	return display == "table" || display == "inline-table"
}

func displayOf(box *Box) string {
	if box == nil || box.Style == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(box.Style.GetString("display")))
}

func (layout *Layout) layoutTable(box *Box) {
	rows := []*Box{}
	for _, child := range box.Children {
		if isOutOfFlow(child) {
			layout.positioned = append(layout.positioned, child)
			continue
		}
		switch displayOf(child) {
		case "table-row":
			rows = append(rows, child)
		case "table-row-group", "table-header-group", "table-footer-group":
			for _, row := range child.Children {
				if displayOf(row) == "table-row" {
					rows = append(rows, row)
				}
			}
		}
	}
	if len(rows) == 0 {
		box.ContentHeight = 0
		return
	}
	columnCount := 0
	for _, row := range rows {
		if count := len(tableCells(row)); count > columnCount {
			columnCount = count
		}
	}
	minWidths := make([]float64, columnCount)
	maxWidths := make([]float64, columnCount)
	for _, row := range rows {
		for column, cell := range tableCells(row) {
			layout.computeIntrinsicWidths(cell)
			horizontalExtra := cellHorizontalExtra(cell, box.ContentWidth)
			minimum := cell.IntrinsicMinWidth + horizontalExtra
			maximum := cell.IntrinsicMaxWidth + horizontalExtra
			if minimum > minWidths[column] {
				minWidths[column] = minimum
			}
			if maximum > maxWidths[column] {
				maxWidths[column] = maximum
			}
		}
	}
	tableExtra := box.Width - box.ContentWidth
	width := box.ContentWidth
	if parseSize(box.Style.GetString("width")).Auto {
		total := 0.0
		for _, maximum := range maxWidths {
			total += maximum
		}
		if total > 0 {
			width = total
			box.ContentWidth = width
			box.Width = width + tableExtra
		}
	}
	widths := distributeTableWidths(minWidths, maxWidths, width)
	cursorY := box.ContentY
	for _, row := range rows {
		rowHeight := 0.0
		cursorX := box.ContentX
		for column, cell := range tableCells(row) {
			cellWidth := widths[column]
			layout.layoutTableCell(cell, cursorX, cursorY, cellWidth)
			if cell.Height > rowHeight {
				rowHeight = cell.Height
			}
			cursorX += cellWidth
		}
		row.X = box.ContentX
		row.ContentX = box.ContentX
		row.Y = cursorY
		row.ContentY = cursorY
		row.Width = box.ContentWidth
		row.ContentWidth = box.ContentWidth
		row.Height = rowHeight
		row.ContentHeight = rowHeight
		cursorY += rowHeight
	}
	box.ContentHeight = math.Max(0, cursorY-box.ContentY)
}

func (layout *Layout) layoutTableCell(cell *Box, x, y, width float64) {
	marginLeft := cell.Margin.Left.Resolve(width)
	marginTop := cell.Margin.Top.Resolve(width)
	paddingLeft := cell.Padding.Left.Resolve(width)
	paddingRight := cell.Padding.Right.Resolve(width)
	paddingTop := cell.Padding.Top.Resolve(width)
	paddingBottom := cell.Padding.Bottom.Resolve(width)
	borderLeft := cell.Border.Left.Resolve(width)
	borderRight := cell.Border.Right.Resolve(width)
	borderTop := cell.Border.Top.Resolve(width)
	borderBottom := cell.Border.Bottom.Resolve(width)
	horizontalExtra := paddingLeft + paddingRight + borderLeft + borderRight
	verticalExtra := paddingTop + paddingBottom + borderTop + borderBottom
	cell.ContentWidth = math.Max(0, width-horizontalExtra)
	cell.Width = width
	cell.X = x + marginLeft
	cell.ContentX = cell.X + borderLeft + paddingLeft
	cell.Y = y + marginTop
	cell.ContentY = cell.Y + borderTop + paddingTop
	layout.layoutFlexItemContent(cell)
	cell.Height = cell.ContentHeight + verticalExtra
}

func tableCells(row *Box) []*Box {
	cells := []*Box{}
	for _, child := range row.Children {
		if displayOf(child) == "table-cell" {
			cells = append(cells, child)
		}
	}
	return cells
}

func cellHorizontalExtra(cell *Box, base float64) float64 {
	return cell.Padding.Horizontal(base) + cell.Border.Horizontal(base) + cell.Margin.Horizontal(base)
}

func distributeTableWidths(minWidths, maxWidths []float64, width float64) []float64 {
	widths := make([]float64, len(minWidths))
	totalMin := 0.0
	totalMax := 0.0
	for index := range minWidths {
		totalMin += minWidths[index]
		totalMax += maxWidths[index]
	}
	if totalMax <= 0 {
		if len(widths) == 0 {
			return widths
		}
		for index := range widths {
			widths[index] = width / float64(len(widths))
		}
		return widths
	}
	switch {
	case width >= totalMax:
		extra := width - totalMax
		for index := range widths {
			widths[index] = maxWidths[index] + extra*maxWidths[index]/totalMax
		}
	case width <= totalMin:
		for index := range widths {
			widths[index] = minWidths[index] * width / totalMin
		}
	default:
		delta := totalMax - totalMin
		for index := range widths {
			widths[index] = minWidths[index] + (maxWidths[index]-minWidths[index])*(width-totalMin)/delta
		}
	}
	return widths
}
