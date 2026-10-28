package rendering

import (
	"strings"

	"gonetsurf/dom"
	"gonetsurf/layouts"
	"gonetsurf/urls"
)

func (painter *Painter) paintReplacedContent(box *layouts.Box, surface *Surface, offsetX, offsetY float64) {
	if box.Node == nil || box.Node.Type != dom.ElementNode || !strings.EqualFold(box.Node.Data, "img") {
		return
	}
	source, ok := box.Node.GetAttribute("src")
	if !ok || strings.TrimSpace(source) == "" {
		return
	}
	if painter.BaseURL != nil {
		if resolved, err := urls.ParseReference(painter.BaseURL, source); err == nil {
			source = resolved.String()
		}
	}
	picture := painter.loadImageURL(source)
	if picture == nil {
		return
	}
	contentX := box.ContentX + offsetX
	contentY := box.ContentY + offsetY
	contentWidth := box.ContentWidth
	contentHeight := box.ContentHeight
	if contentWidth <= 0 || contentHeight <= 0 {
		contentX = box.X + offsetX
		contentY = box.Y + offsetY
		contentWidth = box.Width
		contentHeight = box.Height
	}
	if contentWidth <= 0 || contentHeight <= 0 {
		return
	}
	contentRect := Rect{X: contentX, Y: contentY, Width: contentWidth, Height: contentHeight}
	intrinsicWidth := float64(picture.Bounds().Dx())
	intrinsicHeight := float64(picture.Bounds().Dy())
	if intrinsicWidth <= 0 || intrinsicHeight <= 0 {
		return
	}
	fit := "fill"
	if box.Style != nil {
		if value := strings.ToLower(strings.TrimSpace(box.Style.GetString("object-fit"))); value != "" {
			fit = value
		}
	}
	destination := objectFitRect(contentRect, intrinsicWidth, intrinsicHeight, fit)
	surface.DrawImage(picture, destination, contentRect, nil, 1)
}

func objectFitRect(content Rect, intrinsicWidth, intrinsicHeight float64, fit string) Rect {
	switch fit {
	case "contain":
		scale := minFloat(content.Width/intrinsicWidth, content.Height/intrinsicHeight)
		return centeredRect(content, intrinsicWidth*scale, intrinsicHeight*scale)
	case "cover":
		scale := maxFloat(content.Width/intrinsicWidth, content.Height/intrinsicHeight)
		return centeredRect(content, intrinsicWidth*scale, intrinsicHeight*scale)
	case "none":
		return centeredRect(content, intrinsicWidth, intrinsicHeight)
	case "scale-down":
		scale := minFloat(1, minFloat(content.Width/intrinsicWidth, content.Height/intrinsicHeight))
		return centeredRect(content, intrinsicWidth*scale, intrinsicHeight*scale)
	default:
		return content
	}
}

func centeredRect(content Rect, width, height float64) Rect {
	return Rect{
		X:      content.X + (content.Width-width)/2,
		Y:      content.Y + (content.Height-height)/2,
		Width:  width,
		Height: height,
	}
}

func minFloat(first, second float64) float64 {
	if first < second {
		return first
	}
	return second
}
