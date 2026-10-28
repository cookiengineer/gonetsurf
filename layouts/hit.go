package layouts

import "gonetsurf/dom"

// FindBox returns the box generated for node, or nil when the node has no box.
func FindBox(root *Box, node *dom.Node) *Box {
	return findBoxByNode(root, node)
}

// HitTest returns the deepest box containing the viewport-space point. Scroll
// offsets recorded on the box tree are applied while descending, so callers can
// pass coordinates in the same space used for painting.
func HitTest(root *Box, x, y float64) *Box {
	if root == nil {
		return nil
	}
	return hitTestBox(root, 0, 0, x, y)
}

func hitTestBox(box *Box, offsetX, offsetY, x, y float64) *Box {
	if box == nil {
		return nil
	}
	boxX := box.X + offsetX
	boxY := box.Y + offsetY
	if x < boxX || y < boxY || x >= boxX+box.Width || y >= boxY+box.Height {
		return nil
	}
	if box.ClipX || box.ClipY {
		if x < box.ContentX+offsetX || y < box.ContentY+offsetY ||
			x >= box.ContentX+offsetX+box.ContentWidth ||
			y >= box.ContentY+offsetY+box.ContentHeight {
			return nil
		}
	}
	childOffsetX := offsetX - box.ScrollLeft
	childOffsetY := offsetY - box.ScrollTop
	for index := len(box.Children) - 1; index >= 0; index-- {
		if found := hitTestBox(box.Children[index], childOffsetX, childOffsetY, x, y); found != nil {
			return found
		}
	}
	return box
}

// BoxRect returns the viewport-space border rectangle of target within root.
// Nested scroll offsets recorded on the box tree are applied.
func BoxRect(root, target *Box) (Rect, bool) {
	if root == nil || target == nil {
		return Rect{}, false
	}
	return boxRect(root, 0, 0, target)
}

func boxRect(box *Box, offsetX, offsetY float64, target *Box) (Rect, bool) {
	if box == nil {
		return Rect{}, false
	}
	if box == target {
		return Rect{
			X:      box.X + offsetX,
			Y:      box.Y + offsetY,
			Width:  box.Width,
			Height: box.Height,
		}, true
	}
	childOffsetX := offsetX - box.ScrollLeft
	childOffsetY := offsetY - box.ScrollTop
	for _, child := range box.Children {
		if rect, ok := boxRect(child, childOffsetX, childOffsetY, target); ok {
			return rect, true
		}
	}
	return Rect{}, false
}

// Rect is an axis-aligned rectangle in layout coordinates.
type Rect struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}
