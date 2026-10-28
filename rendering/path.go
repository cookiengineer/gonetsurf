package rendering

import "math"

// Point is a position in surface coordinates.
type Point struct {
	X float64
	Y float64
}

type pathVerb uint8

const (
	moveToVerb pathVerb = iota
	lineToVerb
	quadToVerb
	cubicToVerb
	closeVerb
)

// Path is a sequence of move, line, quadratic, cubic, and close commands.
type Path struct {
	verbs  []pathVerb
	points []Point
}

// NewPath returns an empty path.
func NewPath() *Path {
	return &Path{}
}

// MoveTo starts a new subpath at (xPos, yPos).
func (path *Path) MoveTo(xPos, yPos float64) {
	path.verbs = append(path.verbs, moveToVerb)
	path.points = append(path.points, Point{X: xPos, Y: yPos})
}

// LineTo appends a line to the current point.
func (path *Path) LineTo(xPos, yPos float64) {
	path.verbs = append(path.verbs, lineToVerb)
	path.points = append(path.points, Point{X: xPos, Y: yPos})
}

// QuadTo appends a quadratic curve with the given control point.
func (path *Path) QuadTo(controlX, controlY, xPos, yPos float64) {
	path.verbs = append(path.verbs, quadToVerb)
	path.points = append(path.points, Point{X: controlX, Y: controlY}, Point{X: xPos, Y: yPos})
}

// CubicTo appends a cubic curve with the given control points.
func (path *Path) CubicTo(control1X, control1Y, control2X, control2Y, xPos, yPos float64) {
	path.verbs = append(path.verbs, cubicToVerb)
	path.points = append(path.points,
		Point{X: control1X, Y: control1Y},
		Point{X: control2X, Y: control2Y},
		Point{X: xPos, Y: yPos},
	)
}

// Close closes the current subpath.
func (path *Path) Close() {
	path.verbs = append(path.verbs, closeVerb)
}

// AddRect appends an axis-aligned rectangle as a closed subpath.
func (path *Path) AddRect(rect Rect) {
	path.MoveTo(rect.X, rect.Y)
	path.LineTo(rect.X+rect.Width, rect.Y)
	path.LineTo(rect.X+rect.Width, rect.Y+rect.Height)
	path.LineTo(rect.X, rect.Y+rect.Height)
	path.Close()
}

// AddPolygon appends a closed polygon.
func (path *Path) AddPolygon(points []Point) {
	if len(points) == 0 {
		return
	}
	path.MoveTo(points[0].X, points[0].Y)
	for _, point := range points[1:] {
		path.LineTo(point.X, point.Y)
	}
	path.Close()
}

// AddCircle appends a closed approximation of a circle.
func (path *Path) AddCircle(centerX, centerY, radius float64) {
	const circleSegments = 32
	if radius <= 0 {
		return
	}
	for index := 0; index < circleSegments; index++ {
		angle := 2 * math.Pi * float64(index) / circleSegments
		xPos := centerX + radius*math.Cos(angle)
		yPos := centerY + radius*math.Sin(angle)
		if index == 0 {
			path.MoveTo(xPos, yPos)
		} else {
			path.LineTo(xPos, yPos)
		}
	}
	path.Close()
}

// CornerRadii holds the horizontal and vertical radii of each corner.
type CornerRadii struct {
	TopLeft     Point
	TopRight    Point
	BottomRight Point
	BottomLeft  Point
}

// AddRoundedRect appends a closed rounded rectangle. Radii are clamped and
// scaled down when adjacent corners would overlap, following CSS Backgrounds
// and Borders 3 §5.5.
func (path *Path) AddRoundedRect(rect Rect, radii CornerRadii) {
	if rect.Width <= 0 || rect.Height <= 0 {
		return
	}
	radii = normalizeRadii(rect, radii)
	if radii.TopLeft.X == 0 && radii.TopRight.X == 0 && radii.BottomRight.X == 0 && radii.BottomLeft.X == 0 &&
		radii.TopLeft.Y == 0 && radii.TopRight.Y == 0 && radii.BottomRight.Y == 0 && radii.BottomLeft.Y == 0 {
		path.AddRect(rect)
		return
	}
	const arc = 0.5522847498307936
	right := rect.X + rect.Width
	bottom := rect.Y + rect.Height

	path.MoveTo(rect.X+radii.TopLeft.X, rect.Y)
	path.LineTo(right-radii.TopRight.X, rect.Y)
	path.CubicTo(
		right-radii.TopRight.X*(1-arc), rect.Y,
		right, rect.Y+radii.TopRight.Y*(1-arc),
		right, rect.Y+radii.TopRight.Y,
	)
	path.LineTo(right, bottom-radii.BottomRight.Y)
	path.CubicTo(
		right, bottom-radii.BottomRight.Y*(1-arc),
		right-radii.BottomRight.X*(1-arc), bottom,
		right-radii.BottomRight.X, bottom,
	)
	path.LineTo(rect.X+radii.BottomLeft.X, bottom)
	path.CubicTo(
		rect.X+radii.BottomLeft.X*(1-arc), bottom,
		rect.X, bottom-radii.BottomLeft.Y*(1-arc),
		rect.X, bottom-radii.BottomLeft.Y,
	)
	path.LineTo(rect.X, rect.Y+radii.TopLeft.Y)
	path.CubicTo(
		rect.X, rect.Y+radii.TopLeft.Y*(1-arc),
		rect.X+radii.TopLeft.X*(1-arc), rect.Y,
		rect.X+radii.TopLeft.X, rect.Y,
	)
	path.Close()
}

func normalizeRadii(rect Rect, radii CornerRadii) CornerRadii {
	clamp := func(radius Point) Point {
		return Point{X: math.Max(0, radius.X), Y: math.Max(0, radius.Y)}
	}
	radii.TopLeft = clamp(radii.TopLeft)
	radii.TopRight = clamp(radii.TopRight)
	radii.BottomRight = clamp(radii.BottomRight)
	radii.BottomLeft = clamp(radii.BottomLeft)
	scale := 1.0
	scale = math.Min(scale, rect.Width/(radii.TopLeft.X+radii.TopRight.X))
	scale = math.Min(scale, rect.Width/(radii.BottomLeft.X+radii.BottomRight.X))
	scale = math.Min(scale, rect.Height/(radii.TopLeft.Y+radii.BottomLeft.Y))
	scale = math.Min(scale, rect.Height/(radii.TopRight.Y+radii.BottomRight.Y))
	if scale < 1 {
		radii.TopLeft = radii.TopLeft.scaled(scale)
		radii.TopRight = radii.TopRight.scaled(scale)
		radii.BottomRight = radii.BottomRight.scaled(scale)
		radii.BottomLeft = radii.BottomLeft.scaled(scale)
	}
	return radii
}

type flattenedSubpath struct {
	points []Point
	closed bool
}

func (path *Path) flatten(tolerance float64) []flattenedSubpath {
	if tolerance <= 0 {
		tolerance = 0.25
	}
	var subpaths []flattenedSubpath
	var current []Point
	closed := false
	flush := func() {
		if len(current) > 0 {
			subpaths = append(subpaths, flattenedSubpath{points: current, closed: closed})
		}
		current = nil
		closed = false
	}

	pointIndex := 0
	for _, verb := range path.verbs {
		switch verb {
		case moveToVerb:
			flush()
			current = append(current, path.points[pointIndex])
			pointIndex++
		case lineToVerb:
			point := path.points[pointIndex]
			pointIndex++
			if len(current) == 0 {
				current = append(current, point)
			} else {
				current = append(current, point)
			}
		case quadToVerb:
			control := path.points[pointIndex]
			end := path.points[pointIndex+1]
			pointIndex += 2
			if len(current) == 0 {
				current = append(current, control)
			}
			start := current[len(current)-1]
			current = flattenQuad(current, start, control, end, tolerance, 0)
		case cubicToVerb:
			control1 := path.points[pointIndex]
			control2 := path.points[pointIndex+1]
			end := path.points[pointIndex+2]
			pointIndex += 3
			if len(current) == 0 {
				current = append(current, control1)
			}
			start := current[len(current)-1]
			current = flattenCubic(current, start, control1, control2, end, tolerance, 0)
		case closeVerb:
			if len(current) > 0 {
				closed = true
				flush()
			}
		}
	}
	flush()
	return subpaths
}

func flattenQuad(points []Point, start, control, end Point, tolerance float64, depth int) []Point {
	if depth >= 20 || distanceToLine(control, start, end) <= tolerance {
		return append(points, end)
	}
	startControl := midpoint(start, control)
	controlEnd := midpoint(control, end)
	middle := midpoint(startControl, controlEnd)
	points = flattenQuad(points, start, startControl, middle, tolerance, depth+1)
	return flattenQuad(points, middle, controlEnd, end, tolerance, depth+1)
}

func flattenCubic(points []Point, start, control1, control2, end Point, tolerance float64, depth int) []Point {
	if depth >= 20 {
		return append(points, end)
	}
	flatness := math.Max(distanceToLine(control1, start, end), distanceToLine(control2, start, end))
	if flatness <= tolerance {
		return append(points, end)
	}
	startControl1 := midpoint(start, control1)
	control1Control2 := midpoint(control1, control2)
	control2End := midpoint(control2, end)
	startMiddle := midpoint(startControl1, control1Control2)
	middleEnd := midpoint(control1Control2, control2End)
	middle := midpoint(startMiddle, middleEnd)
	points = flattenCubic(points, start, startControl1, startMiddle, middle, tolerance, depth+1)
	return flattenCubic(points, middle, middleEnd, control2End, end, tolerance, depth+1)
}

func midpoint(first, second Point) Point {
	return Point{X: (first.X + second.X) / 2, Y: (first.Y + second.Y) / 2}
}

func distanceToLine(point, start, end Point) float64 {
	deltaX := end.X - start.X
	deltaY := end.Y - start.Y
	lengthSquared := deltaX*deltaX + deltaY*deltaY
	if lengthSquared == 0 {
		return math.Hypot(point.X-start.X, point.Y-start.Y)
	}
	projection := ((point.X-start.X)*deltaX + (point.Y-start.Y)*deltaY) / lengthSquared
	projection = math.Max(0, math.Min(1, projection))
	closestX := start.X + projection*deltaX
	closestY := start.Y + projection*deltaY
	return math.Hypot(point.X-closestX, point.Y-closestY)
}
