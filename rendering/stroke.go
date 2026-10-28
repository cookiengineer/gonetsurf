package rendering

import "math"

// LineCap selects how open subpath ends are drawn.
type LineCap uint8

const (
	// ButtCap ends the stroke exactly at the endpoint.
	ButtCap LineCap = iota
	// RoundCap draws a semicircle at the endpoint.
	RoundCap
	// SquareCap extends the stroke by half its width.
	SquareCap
)

// LineJoin selects how segments meet at a vertex.
type LineJoin uint8

const (
	// MiterJoin extends the outer edges to a point.
	MiterJoin LineJoin = iota
	// BevelJoin connects the outer edges with a straight line.
	BevelJoin
	// RoundJoin draws a circular arc at the vertex.
	RoundJoin
)

// StrokeStyle configures path stroking.
type StrokeStyle struct {
	Width      float64
	Cap        LineCap
	Join       LineJoin
	MiterLimit float64
}

// DefaultStrokeStyle returns a butt/round stroke of the given width.
func DefaultStrokeStyle(width float64) StrokeStyle {
	return StrokeStyle{Width: width, Cap: ButtCap, Join: RoundJoin, MiterLimit: 10}
}

// RasterizeStroke rasterizes a stroked path into a coverage mask.
func RasterizeStroke(path *Path, width, height int, style StrokeStyle) *Mask {
	mask := NewMask(width, height)
	if path == nil || style.Width <= 0 {
		return mask
	}
	if style.MiterLimit <= 0 {
		style.MiterLimit = 10
	}
	halfWidth := style.Width / 2
	for _, subpath := range path.flatten(0.25) {
		points := deduplicatePoints(subpath.points)
		if len(points) == 0 {
			continue
		}
		if len(points) == 1 {
			addCap(mask, points[0], Point{X: 1}, halfWidth, style.Cap)
			continue
		}
		segmentCount := len(points) - 1
		if subpath.closed {
			segmentCount = len(points)
		}
		for index := 0; index < segmentCount; index++ {
			start := points[index]
			end := points[(index+1)%len(points)]
			addSegment(mask, start, end, halfWidth)
		}
		if subpath.closed {
			for index := range points {
				previous := points[(index-1+len(points))%len(points)]
				next := points[(index+1)%len(points)]
				addJoin(mask, points[index], previous, next, halfWidth, style)
			}
			continue
		}
		for index := 1; index+1 < len(points); index++ {
			addJoin(mask, points[index], points[index-1], points[index+1], halfWidth, style)
		}
		addCap(mask, points[0], points[0].subtract(points[1]), halfWidth, style.Cap)
		last := len(points) - 1
		addCap(mask, points[last], points[last].subtract(points[last-1]), halfWidth, style.Cap)
	}
	return mask
}

func addSegment(mask *Mask, start, end Point, halfWidth float64) {
	direction := end.subtract(start)
	if direction.length() == 0 {
		return
	}
	normal := direction.normalized().leftNormal().scaled(halfWidth)
	polygon := []Point{
		start.add(normal),
		end.add(normal),
		end.subtract(normal),
		start.subtract(normal),
	}
	addPolygon(mask, polygon)
}

func addCap(mask *Mask, endpoint, outward Point, halfWidth float64, cap LineCap) {
	switch cap {
	case RoundCap:
		circle := NewPath()
		circle.AddCircle(endpoint.X, endpoint.Y, halfWidth)
		for _, subpath := range circle.flatten(0.25) {
			addPolygon(mask, subpath.points)
		}
	case SquareCap:
		direction := outward.normalized()
		if direction.length() == 0 {
			return
		}
		normal := direction.leftNormal().scaled(halfWidth)
		extension := direction.scaled(halfWidth)
		polygon := []Point{
			endpoint.add(normal),
			endpoint.add(normal).add(extension),
			endpoint.subtract(normal).add(extension),
			endpoint.subtract(normal),
		}
		addPolygon(mask, polygon)
	}
}

func addJoin(mask *Mask, vertex, previous, next Point, halfWidth float64, style StrokeStyle) {
	previousDirection := vertex.subtract(previous)
	nextDirection := next.subtract(vertex)
	if previousDirection.length() == 0 || nextDirection.length() == 0 {
		return
	}
	previousDirection = previousDirection.normalized()
	nextDirection = nextDirection.normalized()
	cross := crossProduct(previousDirection, nextDirection)
	if math.Abs(cross) < 1e-9 {
		return
	}
	side := 1.0
	if cross > 0 {
		side = -1
	}
	previousOuter := vertex.add(previousDirection.leftNormal().scaled(halfWidth * side))
	nextOuter := vertex.add(nextDirection.leftNormal().scaled(halfWidth * side))

	switch style.Join {
	case RoundJoin:
		circle := NewPath()
		circle.AddCircle(vertex.X, vertex.Y, halfWidth)
		for _, subpath := range circle.flatten(0.25) {
			addPolygon(mask, subpath.points)
		}
	case BevelJoin:
		addPolygon(mask, []Point{vertex, previousOuter, nextOuter})
	default:
		miterPoint, ok := lineIntersection(previousOuter, previousDirection, nextOuter, nextDirection)
		if !ok {
			addPolygon(mask, []Point{vertex, previousOuter, nextOuter})
			return
		}
		miterLength := miterPoint.subtract(vertex).length() / halfWidth
		if miterLength > style.MiterLimit {
			addPolygon(mask, []Point{vertex, previousOuter, nextOuter})
			return
		}
		addPolygon(mask, []Point{vertex, previousOuter, miterPoint, nextOuter})
	}
}

func addPolygon(mask *Mask, points []Point) {
	if len(points) < 3 {
		return
	}
	edges := make([]rasterEdge, 0, len(points))
	for index := range points {
		edges = append(edges, newEdge(points[index], points[(index+1)%len(points)]))
	}
	primitive := NewMask(mask.Width, mask.Height)
	accumulateEdges(edges, primitive, NonZeroFill, false)
	mask.Union(primitive)
}

func deduplicatePoints(points []Point) []Point {
	result := make([]Point, 0, len(points))
	for _, point := range points {
		if len(result) > 0 && result[len(result)-1].subtract(point).length() < 1e-9 {
			continue
		}
		result = append(result, point)
	}
	return result
}

func lineIntersection(first Point, firstDirection Point, second Point, secondDirection Point) (Point, bool) {
	cross := crossProduct(firstDirection, secondDirection)
	if math.Abs(cross) < 1e-9 {
		return Point{}, false
	}
	delta := second.subtract(first)
	progress := crossProduct(delta, secondDirection) / cross
	return first.add(firstDirection.scaled(progress)), true
}

func (point Point) add(other Point) Point {
	return Point{X: point.X + other.X, Y: point.Y + other.Y}
}

func (point Point) subtract(other Point) Point {
	return Point{X: point.X - other.X, Y: point.Y - other.Y}
}

func (point Point) scaled(factor float64) Point {
	return Point{X: point.X * factor, Y: point.Y * factor}
}

func (point Point) length() float64 {
	return math.Hypot(point.X, point.Y)
}

func (point Point) normalized() Point {
	length := point.length()
	if length == 0 {
		return Point{}
	}
	return Point{X: point.X / length, Y: point.Y / length}
}

func (point Point) leftNormal() Point {
	return Point{X: -point.Y, Y: point.X}
}

func crossProduct(first, second Point) float64 {
	return first.X*second.Y - first.Y*second.X
}
