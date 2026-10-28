package rendering

import (
	"sort"
	"strings"

	"gonetsurf/layouts"
)

type stackingEntry struct {
	box     *layouts.Box
	zIndex  int
	order   int
	offsetX float64
	offsetY float64
	clip    Rect
	hasClip bool
}

func (painter *Painter) paintStackingContext(box *layouts.Box, surface *Surface, offsetX, offsetY float64) {
	painter.paintStackingContextWithEffects(box, surface, offsetX, offsetY, true)
}

func (painter *Painter) paintStackingContextWithEffects(box *layouts.Box, surface *Surface, offsetX, offsetY float64, applyEffects bool) {
	if box == nil {
		return
	}
	if applyEffects {
		matrix, hasTransform := parseBoxTransform(box)
		opacity, hasOpacity := boxOpacity(box)
		if hasTransform || hasOpacity {
			layer := NewSurface(surface.Width, surface.Height)
			painter.paintStackingContextWithEffects(box, layer, offsetX, offsetY, false)
			compositeTransformed(surface, layer, matrix, opacity)
			return
		}
	}
	if box.Kind == layouts.TextBox {
		painter.paintText(box, surface, offsetX, offsetY)
		return
	}

	xPos, yPos, deltaX, deltaY := paintOrigin(box, offsetX, offsetY)
	painter.paintDecorations(box, surface, xPos, yPos)

	childOffsetX := offsetX - box.ScrollLeft + deltaX
	childOffsetY := offsetY - box.ScrollTop + deltaY

	clip := surface.Clip()
	hasClip := true
	if box.ClipX || box.ClipY {
		clip = boxClipRect(box, offsetX, offsetY, surface.Width, surface.Height).intersect(clip)
	}

	var entries []stackingEntry
	counter := 0
	collectStackingEntries(box, childOffsetX, childOffsetY, clip, hasClip, &entries, &counter)
	sort.SliceStable(entries, func(first, second int) bool {
		if entries[first].zIndex == entries[second].zIndex {
			return entries[first].order < entries[second].order
		}
		return entries[first].zIndex < entries[second].zIndex
	})

	for _, entry := range entries {
		if entry.zIndex < 0 {
			painter.paintStackingEntry(surface, entry)
		}
	}
	painter.paintContent(box, surface, childOffsetX, childOffsetY)
	for _, entry := range entries {
		if entry.zIndex >= 0 {
			painter.paintStackingEntry(surface, entry)
		}
	}
	painter.paintScrollbars(box, surface, xPos, yPos)
}

func (painter *Painter) paintStackingEntry(surface *Surface, entry stackingEntry) {
	previousClip := surface.Clip()
	if entry.hasClip {
		surface.SetClip(entry.clip)
	}
	painter.paintStackingContextWithEffects(entry.box, surface, entry.offsetX, entry.offsetY, true)
	surface.SetClip(previousClip)
}

func (painter *Painter) paintContent(box *layouts.Box, surface *Surface, offsetX, offsetY float64) {
	previousClip := surface.Clip()
	clipChanged := false
	if box.ClipX || box.ClipY {
		surface.SetClip(boxClipRect(box, offsetX, offsetY, surface.Width, surface.Height))
		clipChanged = true
	}
	childOffsetX := offsetX - box.ScrollLeft
	childOffsetY := offsetY - box.ScrollTop
	for _, child := range box.Children {
		if child == nil {
			continue
		}
		if child.Kind == layouts.TextBox {
			painter.paintText(child, surface, childOffsetX, childOffsetY)
			continue
		}
		if isStackingContext(child) || isPositioned(child) {
			continue
		}
		xPos, yPos, deltaX, deltaY := paintOrigin(child, childOffsetX, childOffsetY)
		painter.paintDecorations(child, surface, xPos, yPos)
		painter.paintContent(child, surface, childOffsetX+deltaX, childOffsetY+deltaY)
		painter.paintScrollbars(child, surface, xPos, yPos)
	}
	if clipChanged {
		surface.SetClip(previousClip)
	}
}

func paintOrigin(box *layouts.Box, offsetX, offsetY float64) (float64, float64, float64, float64) {
	xPos := box.X + offsetX
	yPos := box.Y + offsetY
	if box.Style == nil || !strings.EqualFold(strings.TrimSpace(box.Style.GetString("position")), "sticky") {
		return xPos, yPos, 0, 0
	}
	container := box.ScrollContainer()
	if container == nil {
		return xPos, yPos, 0, 0
	}
	stickyX, stickyY := box.StickyOffset(container.ScrollLeft, container.ScrollTop)
	return stickyX + offsetX + container.ScrollLeft,
		stickyY + offsetY + container.ScrollTop,
		stickyX - box.X,
		stickyY - box.Y
}

func (painter *Painter) paintDecorations(box *layouts.Box, surface *Surface, xPos, yPos float64) {
	if box.Style != nil {
		painter.paintOuterShadows(box, surface, xPos, yPos)
	}
	painter.paintBackground(box, surface, xPos-box.X, yPos-box.Y)
	painter.paintBorders(box, surface, xPos, yPos)
	if box.Style != nil {
		painter.paintInnerShadows(box, surface, xPos, yPos)
	}
	painter.paintReplacedContent(box, surface, xPos-box.X, yPos-box.Y)
}

func collectStackingEntries(box *layouts.Box, offsetX, offsetY float64, clip Rect, hasClip bool, entries *[]stackingEntry, counter *int) {
	for _, child := range box.Children {
		if child == nil || child.Kind == layouts.TextBox {
			continue
		}
		if isStackingContext(child) || isPositioned(child) {
			entryClip := clip
			*entries = append(*entries, stackingEntry{
				box:     child,
				zIndex:  effectiveZIndex(child),
				order:   *counter,
				offsetX: offsetX,
				offsetY: offsetY,
				clip:    entryClip,
				hasClip: hasClip,
			})
			*counter++
			continue
		}
		childClip := clip
		childHasClip := hasClip
		if child.ClipX || child.ClipY {
			rect := boxClipRect(child, offsetX, offsetY, 1<<30, 1<<30)
			childClip = rect.intersect(clip)
			childHasClip = true
		}
		nextOffsetX := offsetX - child.ScrollLeft
		nextOffsetY := offsetY - child.ScrollTop
		collectStackingEntries(child, nextOffsetX, nextOffsetY, childClip, childHasClip, entries, counter)
	}
}

func boxClipRect(box *layouts.Box, offsetX, offsetY float64, surfaceWidth, surfaceHeight int) Rect {
	xPos := box.X + offsetX
	yPos := box.Y + offsetY
	rect := Rect{X: xPos, Y: yPos, Width: box.Width, Height: box.Height}
	if box.ClipX && !box.ClipY {
		rect.Y = -float64(surfaceHeight)
		rect.Height = float64(surfaceHeight * 2)
	}
	if box.ClipY && !box.ClipX {
		rect.X = -float64(surfaceWidth)
		rect.Width = float64(surfaceWidth * 2)
	}
	return rect
}

func isPositioned(box *layouts.Box) bool {
	if box == nil || box.Style == nil {
		return false
	}
	position := strings.ToLower(strings.TrimSpace(box.Style.GetString("position")))
	return position != "" && position != "static"
}

func isStackingContext(box *layouts.Box) bool {
	if box == nil || box.Style == nil {
		return false
	}
	if _, hasTransform := parseBoxTransform(box); hasTransform {
		return true
	}
	if _, hasOpacity := boxOpacity(box); hasOpacity {
		return true
	}
	position := strings.ToLower(strings.TrimSpace(box.Style.GetString("position")))
	if position == "fixed" || position == "sticky" {
		return true
	}
	if position != "" && position != "static" && !zIndexAuto(box.Style.GetString("z-index")) {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(box.Style.GetString("isolation")), "isolate") {
		return true
	}
	return false
}

func effectiveZIndex(box *layouts.Box) int {
	if box == nil || box.Style == nil {
		return 0
	}
	if zIndexAuto(box.Style.GetString("z-index")) {
		return 0
	}
	return box.ZIndex
}

func zIndexAuto(value string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	return trimmed == "" || trimmed == "auto"
}

func boxOpacity(box *layouts.Box) (float64, bool) {
	if box == nil || box.Style == nil {
		return 1, false
	}
	value, ok := parseOpacityValue(box.Style.GetString("opacity"))
	if !ok || value >= 1 {
		return 1, false
	}
	return value, true
}
