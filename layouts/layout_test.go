package layouts

import (
	"math"
	"testing"

	"gonetsurf/dom"
	"gonetsurf/parsers/css/cssom"
	"gonetsurf/styles"

	"golang.org/x/text/unicode/bidi"
)

func testDocument() (*dom.Document, *dom.Node) {
	document := dom.NewDocument()
	html := document.CreateElement("html")
	body := document.CreateElement("body")
	html.AppendChild(body)
	document.AppendChild(html)
	return document, body
}

func testEngine(document *dom.Document, css string) *styles.Engine {
	engine := styles.NewEngine(document)
	engine.AddStylesheet(cssom.Parse(css))
	return engine
}

func findBox(root *Box, node *dom.Node) *Box {
	if root == nil {
		return nil
	}
	if root.Node == node {
		return root
	}
	for _, child := range root.Children {
		if found := findBox(child, node); found != nil {
			return found
		}
	}
	return nil
}

func TestBoxTreeConstruction(t *testing.T) {
	document, body := testDocument()
	container := document.CreateElement("div")
	container.SetAttribute("id", "container")
	container.AppendChild(document.CreateTextNode("hello"))
	span := document.CreateElement("span")
	span.AppendChild(document.CreateTextNode("world"))
	container.AppendChild(span)
	paragraph := document.CreateElement("p")
	paragraph.AppendChild(document.CreateTextNode("child"))
	container.AppendChild(paragraph)
	hidden := document.CreateElement("div")
	hidden.SetAttribute("class", "hidden")
	hidden.AppendChild(document.CreateTextNode("hidden"))
	container.AppendChild(hidden)
	contents := document.CreateElement("div")
	contents.SetAttribute("class", "contents")
	promoted := document.CreateElement("p")
	promoted.AppendChild(document.CreateTextNode("promoted"))
	contents.AppendChild(promoted)
	container.AppendChild(contents)
	body.AppendChild(container)

	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		div { display:block }
		span { display:inline }
		.hidden { display:none }
		.contents { display:contents }
	`)
	root := BuildDocument(document, engine)
	containerBox := findBox(root, container)
	if containerBox == nil {
		t.Fatal("container box missing")
	}
	if len(containerBox.Children) != 3 {
		t.Fatalf("container children = %d, want 3", len(containerBox.Children))
	}
	anonymous := containerBox.Children[0]
	if anonymous.Kind != AnonymousBlockBox || len(anonymous.Children) != 2 {
		t.Errorf("anonymous box = %#v", anonymous)
	}
	if anonymous.Children[0].Kind != TextBox || anonymous.Children[0].Text != "hello" {
		t.Errorf("text box = %#v", anonymous.Children[0])
	}
	if anonymous.Children[1].Kind != InlineBox || anonymous.Children[1].Node != span {
		t.Errorf("inline box = %#v", anonymous.Children[1])
	}
	if containerBox.Children[1].Node != paragraph || containerBox.Children[1].Kind != BlockBox {
		t.Errorf("paragraph box = %#v", containerBox.Children[1])
	}
	if containerBox.Children[2].Node != promoted {
		t.Errorf("promoted contents box = %#v", containerBox.Children[2])
	}
	if findBox(root, hidden) != nil {
		t.Error("display:none box must not be generated")
	}
}

func TestBoxModel(t *testing.T) {
	document, body := testDocument()
	boxElement := document.CreateElement("div")
	boxElement.SetAttribute("class", "edges box")
	autoElement := document.CreateElement("div")
	autoElement.SetAttribute("class", "edges auto")
	borderElement := document.CreateElement("div")
	borderElement.SetAttribute("class", "edges border")
	body.AppendChild(boxElement)
	body.AppendChild(autoElement)
	body.AppendChild(borderElement)

	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		.edges { display:block; margin:5px; padding:10px; border:2px solid }
		.box { width:100px; height:50px }
		.border { width:100px; box-sizing:border-box }
	`)
	root := BuildDocument(document, engine)
	layout := New(400, 600)
	layout.Layout(root)

	box := findBox(root, boxElement)
	if box.ContentWidth != 100 || box.Width != 124 {
		t.Errorf("content-box width = %v / %v", box.ContentWidth, box.Width)
	}
	if box.X != 5 || box.ContentX != 17 {
		t.Errorf("content-box x = %v / %v", box.X, box.ContentX)
	}
	if box.ContentHeight != 50 || box.Height != 74 {
		t.Errorf("content-box height = %v / %v", box.ContentHeight, box.Height)
	}

	auto := findBox(root, autoElement)
	if auto.ContentWidth != 366 || auto.Width != 390 {
		t.Errorf("auto width = %v / %v", auto.ContentWidth, auto.Width)
	}
	if auto.ContentX != 17 {
		t.Errorf("auto content x = %v", auto.ContentX)
	}

	border := findBox(root, borderElement)
	if border.ContentWidth != 76 || border.Width != 100 {
		t.Errorf("border-box width = %v / %v", border.ContentWidth, border.Width)
	}
}

func TestBlockLayoutAndMarginCollapse(t *testing.T) {
	document, body := testDocument()
	first := document.CreateElement("div")
	first.SetAttribute("class", "a")
	second := document.CreateElement("div")
	second.SetAttribute("class", "b")
	body.AppendChild(first)
	body.AppendChild(second)

	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		.a { display:block; margin:0; margin-bottom:10px; height:50px }
		.b { display:block; margin:0; margin-top:20px; height:30px }
	`)
	root := BuildDocument(document, engine)
	layout := New(400, 600)
	layout.Layout(root)

	firstBox := findBox(root, first)
	secondBox := findBox(root, second)
	if firstBox.Y != 0 || firstBox.Height != 50 {
		t.Errorf("first = y:%v h:%v", firstBox.Y, firstBox.Height)
	}
	if secondBox.Y != 70 {
		t.Errorf("second y = %v, want 70", secondBox.Y)
	}
	bodyBox := findBox(root, body)
	if bodyBox.ContentHeight != 100 {
		t.Errorf("body content height = %v, want 100", bodyBox.ContentHeight)
	}
}

func TestInlineLineLayout(t *testing.T) {
	document, body := testDocument()
	paragraph := document.CreateElement("p")
	paragraph.AppendChild(document.CreateTextNode("hello world hello world"))
	body.AppendChild(paragraph)

	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		p { display:block; width:100px; font-size:16px; line-height:19.2px; margin:0; padding:0; border:0 }
	`)
	root := BuildDocument(document, engine)
	layout := New(400, 600)
	layout.Layout(root)

	p := findBox(root, paragraph)
	if math.Abs(p.ContentHeight-38.4) > 1e-9 {
		t.Errorf("content height = %v, want 38.4", p.ContentHeight)
	}
	if len(p.Children) != 4 {
		t.Fatalf("word boxes = %d, want 4", len(p.Children))
	}
	for index, word := range p.Children {
		wantY := 0.0
		if index >= 2 {
			wantY = 19.2
		}
		if math.Abs(word.Y-wantY) > 1e-9 {
			t.Errorf("word %d y = %v, want %v", index, word.Y, wantY)
		}
	}
	if p.Children[0].Width != 40 || p.Children[1].Width != 40 {
		t.Errorf("word widths = %v, %v", p.Children[0].Width, p.Children[1].Width)
	}
}

func TestIntrinsicWidths(t *testing.T) {
	document, body := testDocument()
	minElement := document.CreateElement("p")
	minElement.SetAttribute("class", "min")
	minElement.AppendChild(document.CreateTextNode("hello world"))
	maxElement := document.CreateElement("p")
	maxElement.SetAttribute("class", "max")
	maxElement.AppendChild(document.CreateTextNode("hello world"))
	clampElement := document.CreateElement("p")
	clampElement.SetAttribute("class", "clamp")
	clampElement.AppendChild(document.CreateTextNode("hello world"))
	shrink := document.CreateElement("div")
	shrink.SetAttribute("class", "shrink")
	fitElement := document.CreateElement("p")
	fitElement.SetAttribute("class", "fit")
	fitElement.AppendChild(document.CreateTextNode("hello world"))
	shrink.AppendChild(fitElement)
	body.AppendChild(minElement)
	body.AppendChild(maxElement)
	body.AppendChild(clampElement)
	body.AppendChild(shrink)

	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		p { display:block; font-size:16px; line-height:19.2px; margin:0; padding:0; border:0 }
		.min { width:min-content }
		.max { width:max-content }
		.clamp { width:max-content; max-width:min-content }
		.shrink { display:block; width:60px }
		.fit { width:fit-content }
	`)
	root := BuildDocument(document, engine)
	layout := New(400, 600)
	layout.Layout(root)

	if got := findBox(root, minElement).ContentWidth; got != 40 {
		t.Errorf("min-content width = %v, want 40", got)
	}
	if got := findBox(root, maxElement).ContentWidth; got != 84 {
		t.Errorf("max-content width = %v, want 84", got)
	}
	if got := findBox(root, clampElement).ContentWidth; got != 40 {
		t.Errorf("max-width min-content = %v, want 40", got)
	}
	if got := findBox(root, fitElement).ContentWidth; got != 60 {
		t.Errorf("fit-content width = %v, want 60", got)
	}
}

func TestParentChildMarginCollapse(t *testing.T) {
	document, body := testDocument()
	outer := document.CreateElement("div")
	outer.SetAttribute("class", "outer")
	inner := document.CreateElement("div")
	inner.SetAttribute("class", "inner")
	outer.AppendChild(inner)
	body.AppendChild(outer)

	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		.outer { display:block; margin:20px 0 0 0; padding:0; border:0 }
		.inner { display:block; margin:30px 0 0 0; padding:0; border:0; height:10px }
	`)
	root := BuildDocument(document, engine)
	layout := New(400, 600)
	layout.Layout(root)
	outerBox := findBox(root, outer)
	innerBox := findBox(root, inner)
	if outerBox.Y != 30 {
		t.Errorf("collapsed outer y = %v, want 30", outerBox.Y)
	}
	if innerBox.Y != 30 {
		t.Errorf("collapsed inner y = %v, want 30", innerBox.Y)
	}
	if outerBox.ContentHeight != 10 {
		t.Errorf("outer content height = %v, want 10", outerBox.ContentHeight)
	}
}

func TestPaddingPreventsMarginCollapse(t *testing.T) {
	document, body := testDocument()
	outer := document.CreateElement("div")
	outer.SetAttribute("class", "outer")
	inner := document.CreateElement("div")
	inner.SetAttribute("class", "inner")
	outer.AppendChild(inner)
	body.AppendChild(outer)

	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		.outer { display:block; margin-top:20px; padding-top:1px; padding-bottom:0; padding-left:0; padding-right:0; border:0 }
		.inner { display:block; margin-top:30px; margin-bottom:0; margin-left:0; margin-right:0; height:10px; padding:0; border:0 }
	`)
	root := BuildDocument(document, engine)
	layout := New(400, 600)
	layout.Layout(root)
	outerBox := findBox(root, outer)
	innerBox := findBox(root, inner)
	if outerBox.Y != 20 || innerBox.Y != 51 {
		t.Errorf("padding case outer y=%v inner y=%v, want 20/51", outerBox.Y, innerBox.Y)
	}
}

func TestOverflowPreventsMarginCollapse(t *testing.T) {
	document, body := testDocument()
	outer := document.CreateElement("div")
	outer.SetAttribute("class", "outer")
	inner := document.CreateElement("div")
	inner.SetAttribute("class", "inner")
	outer.AppendChild(inner)
	body.AppendChild(outer)

	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		.outer { display:block; margin-top:20px; overflow-x:hidden; overflow-y:hidden; padding:0; border:0 }
		.inner { display:block; margin-top:30px; margin-bottom:0; margin-left:0; margin-right:0; height:10px; padding:0; border:0 }
	`)
	root := BuildDocument(document, engine)
	layout := New(400, 600)
	layout.Layout(root)
	if got := findBox(root, inner).Y; got != 50 {
		t.Errorf("BFC inner y = %v, want 50", got)
	}
}

func TestParentChildBottomMarginCollapse(t *testing.T) {
	document, body := testDocument()
	outer := document.CreateElement("div")
	outer.SetAttribute("class", "outer")
	inner := document.CreateElement("div")
	inner.SetAttribute("class", "inner")
	outer.AppendChild(inner)
	next := document.CreateElement("div")
	next.SetAttribute("class", "next")
	body.AppendChild(outer)
	body.AppendChild(next)

	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		.outer { display:block; margin:0; padding:0; border:0 }
		.inner { display:block; margin:0 0 30px 0; height:10px; padding:0; border:0 }
		.next { display:block; height:10px; margin:0 }
	`)
	root := BuildDocument(document, engine)
	layout := New(400, 600)
	layout.Layout(root)
	outerBox := findBox(root, outer)
	innerBox := findBox(root, inner)
	nextBox := findBox(root, next)
	if outerBox.ContentHeight != 10 {
		t.Errorf("outer content height = %v, want 10", outerBox.ContentHeight)
	}
	if innerBox.Y != 0 {
		t.Errorf("inner y = %v, want 0", innerBox.Y)
	}
	if nextBox.Y != 40 {
		t.Errorf("next y = %v, want 40", nextBox.Y)
	}
}

func TestPositioning(t *testing.T) {
	document, body := testDocument()
	container := document.CreateElement("div")
	container.SetAttribute("class", "container")
	relative := document.CreateElement("div")
	relative.SetAttribute("class", "rel")
	child := document.CreateElement("div")
	child.SetAttribute("class", "child")
	relative.AppendChild(child)
	absolute := document.CreateElement("div")
	absolute.SetAttribute("class", "abs")
	stretch := document.CreateElement("div")
	stretch.SetAttribute("class", "stretch")
	absoluteBottom := document.CreateElement("div")
	absoluteBottom.SetAttribute("class", "abs-bottom")
	fixed := document.CreateElement("div")
	fixed.SetAttribute("class", "fixed")
	container.AppendChild(relative)
	container.AppendChild(absolute)
	container.AppendChild(stretch)
	container.AppendChild(absoluteBottom)
	body.AppendChild(container)
	body.AppendChild(fixed)

	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		div { margin:0; padding:0; border:0 }
		.container { position:relative; display:block; width:200px; height:100px }
		.rel { position:relative; left:10px; top:5px; height:10px }
		.child { height:5px }
		.abs { position:absolute; left:20px; top:10px; width:50px; height:20px }
		.stretch { position:absolute; left:10px; right:10px; top:0; height:10px }
		.abs-bottom { position:absolute; left:0; bottom:10px; width:20px; height:30px }
		.fixed { position:fixed; right:10px; bottom:10px; width:40px; height:20px; z-index:5 }
	`)
	root := BuildDocument(document, engine)
	layout := New(400, 600)
	layout.Layout(root)

	relBox := findBox(root, relative)
	childBox := findBox(root, child)
	if relBox.X != 10 || relBox.Y != 5 || childBox.X != 10 || childBox.Y != 5 {
		t.Errorf("relative boxes = rel(%v,%v) child(%v,%v)", relBox.X, relBox.Y, childBox.X, childBox.Y)
	}
	absBox := findBox(root, absolute)
	if absBox.X != 20 || absBox.Y != 10 || absBox.Width != 50 || absBox.Height != 20 {
		t.Errorf("absolute box = (%v,%v) %vx%v", absBox.X, absBox.Y, absBox.Width, absBox.Height)
	}
	stretchBox := findBox(root, stretch)
	if stretchBox.X != 10 || stretchBox.Width != 180 {
		t.Errorf("stretch box x/width = %v/%v, want 10/180", stretchBox.X, stretchBox.Width)
	}
	bottomBox := findBox(root, absoluteBottom)
	if bottomBox.Y != 60 {
		t.Errorf("bottom box y = %v, want 60", bottomBox.Y)
	}
	fixedBox := findBox(root, fixed)
	if fixedBox.X != 350 || fixedBox.Y != 570 {
		t.Errorf("fixed box = (%v,%v), want (350,570)", fixedBox.X, fixedBox.Y)
	}
	if fixedBox.ZIndex != 5 {
		t.Errorf("fixed z-index = %d, want 5", fixedBox.ZIndex)
	}
	if got := findBox(root, container).ContentHeight; got != 100 {
		t.Errorf("container content height = %v, want 100", got)
	}
}

func TestFlexboxRow(t *testing.T) {
	document, body := testDocument()
	flex := document.CreateElement("div")
	flex.SetAttribute("class", "flex row")
	items := []*dom.Node{}
	for index := 0; index < 3; index++ {
		item := document.CreateElement("div")
		item.SetAttribute("class", "item")
		flex.AppendChild(item)
		items = append(items, item)
	}
	body.AppendChild(flex)

	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		.flex { display:flex; width:300px; height:40px; margin:0; padding:0; border:0; align-items:flex-start }
		.row { justify-content:space-between }
		.item { width:50px; height:10px; margin:0; padding:0; border:0 }
	`)
	root := BuildDocument(document, engine)
	layout := New(400, 600)
	layout.Layout(root)

	wantX := []float64{0, 125, 250}
	for index, item := range items {
		box := findBox(root, item)
		if box.X != wantX[index] || box.Y != 0 {
			t.Errorf("row item %d = (%v,%v), want (%v,0)", index, box.X, box.Y, wantX[index])
		}
		if box.Width != 50 {
			t.Errorf("row item %d width = %v, want 50", index, box.Width)
		}
	}
}

func TestFlexboxGrow(t *testing.T) {
	document, body := testDocument()
	flex := document.CreateElement("div")
	flex.SetAttribute("class", "flex")
	first := document.CreateElement("div")
	first.SetAttribute("class", "grow1")
	second := document.CreateElement("div")
	second.SetAttribute("class", "grow2")
	flex.AppendChild(first)
	flex.AppendChild(second)
	body.AppendChild(flex)

	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		.flex { display:flex; width:300px; height:20px; margin:0; padding:0; border:0; align-items:flex-start }
		.grow1 { flex-grow:1; flex-basis:0%; height:10px; margin:0; padding:0; border:0 }
		.grow2 { flex-grow:2; flex-basis:0%; height:10px; margin:0; padding:0; border:0 }
	`)
	root := BuildDocument(document, engine)
	layout := New(400, 600)
	layout.Layout(root)

	firstBox := findBox(root, first)
	secondBox := findBox(root, second)
	if firstBox.Width != 100 || firstBox.X != 0 {
		t.Errorf("grow1 = x:%v w:%v, want 0/100", firstBox.X, firstBox.Width)
	}
	if secondBox.Width != 200 || secondBox.X != 100 {
		t.Errorf("grow2 = x:%v w:%v, want 100/200", secondBox.X, secondBox.Width)
	}
}

func TestFlexboxColumn(t *testing.T) {
	document, body := testDocument()
	flex := document.CreateElement("div")
	flex.SetAttribute("class", "flex")
	first := document.CreateElement("div")
	first.SetAttribute("class", "item1")
	second := document.CreateElement("div")
	second.SetAttribute("class", "item2")
	flex.AppendChild(first)
	flex.AppendChild(second)
	body.AppendChild(flex)

	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		.flex { display:flex; flex-direction:column; width:100px; height:100px; margin:0; padding:0; border:0; align-items:center }
		.item1 { width:20px; height:10px; margin:0; padding:0; border:0 }
		.item2 { width:40px; height:20px; margin:0; padding:0; border:0 }
	`)
	root := BuildDocument(document, engine)
	layout := New(400, 600)
	layout.Layout(root)

	firstBox := findBox(root, first)
	secondBox := findBox(root, second)
	if firstBox.X != 40 || firstBox.Y != 0 {
		t.Errorf("column item1 = (%v,%v), want (40,0)", firstBox.X, firstBox.Y)
	}
	if secondBox.X != 30 || secondBox.Y != 10 {
		t.Errorf("column item2 = (%v,%v), want (30,10)", secondBox.X, secondBox.Y)
	}
}

func TestFlexboxGapAndOrder(t *testing.T) {
	document, body := testDocument()
	flex := document.CreateElement("div")
	flex.SetAttribute("class", "flex")
	first := document.CreateElement("div")
	first.SetAttribute("class", "item order2")
	second := document.CreateElement("div")
	second.SetAttribute("class", "item order1")
	third := document.CreateElement("div")
	third.SetAttribute("class", "item")
	flex.AppendChild(first)
	flex.AppendChild(second)
	flex.AppendChild(third)
	body.AppendChild(flex)

	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		.flex { display:flex; width:300px; height:20px; gap:10px; margin:0; padding:0; border:0; align-items:flex-start; justify-content:flex-start }
		.item { width:50px; height:10px; margin:0; padding:0; border:0 }
		.order1 { order:1 }
		.order2 { order:2 }
	`)
	root := BuildDocument(document, engine)
	layout := New(400, 600)
	layout.Layout(root)

	// The un-ordered third item comes first, then order1, then order2.
	if got := findBox(root, third).X; got != 0 {
		t.Errorf("third x = %v, want 0", got)
	}
	if got := findBox(root, second).X; got != 60 {
		t.Errorf("second x = %v, want 60", got)
	}
	if got := findBox(root, first).X; got != 120 {
		t.Errorf("first x = %v, want 120", got)
	}
}

func TestTableLayout(t *testing.T) {
	document, body := testDocument()
	table := document.CreateElement("table")
	table.SetAttribute("class", "table")
	row1 := document.CreateElement("tr")
	cellA := document.CreateElement("td")
	cellA.AppendChild(document.CreateTextNode("a"))
	cellB := document.CreateElement("td")
	cellB.AppendChild(document.CreateTextNode("bb"))
	row1.AppendChild(cellA)
	row1.AppendChild(cellB)
	row2 := document.CreateElement("tr")
	cellC := document.CreateElement("td")
	cellC.AppendChild(document.CreateTextNode("ccc"))
	cellD := document.CreateElement("td")
	cellD.AppendChild(document.CreateTextNode("d"))
	row2.AppendChild(cellC)
	row2.AppendChild(cellD)
	table.AppendChild(row1)
	table.AppendChild(row2)
	body.AppendChild(table)

	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		table { display:table; margin:0; padding:0; border:0 }
		tr { display:table-row }
		td { display:table-cell; margin:0; padding:0; border:0; font-size:16px; line-height:19.2px }
	`)
	root := BuildDocument(document, engine)
	layout := New(400, 600)
	layout.Layout(root)

	tableBox := findBox(root, table)
	if tableBox.ContentWidth != 40 || tableBox.ContentHeight != 38.4 {
		t.Errorf("table = %vx%v, want 40x38.4", tableBox.ContentWidth, tableBox.ContentHeight)
	}
	row1Box := findBox(root, row1)
	row2Box := findBox(root, row2)
	if row1Box.Y != 0 || row2Box.Y != 19.2 {
		t.Errorf("rows = y:%v and y:%v", row1Box.Y, row2Box.Y)
	}
	if findBox(root, cellA).X != 0 || findBox(root, cellA).Width != 24 {
		t.Errorf("cell A = x:%v w:%v, want 0/24", findBox(root, cellA).X, findBox(root, cellA).Width)
	}
	if findBox(root, cellB).X != 24 || findBox(root, cellB).Width != 16 {
		t.Errorf("cell B = x:%v w:%v, want 24/16", findBox(root, cellB).X, findBox(root, cellB).Width)
	}
}

func TestTableExplicitWidth(t *testing.T) {
	document, body := testDocument()
	table := document.CreateElement("table")
	table.SetAttribute("class", "table")
	row := document.CreateElement("tr")
	first := document.CreateElement("td")
	first.AppendChild(document.CreateTextNode("a"))
	second := document.CreateElement("td")
	second.AppendChild(document.CreateTextNode("bb"))
	row.AppendChild(first)
	row.AppendChild(second)
	table.AppendChild(row)
	body.AppendChild(table)

	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		table { display:table; width:100px; margin:0; padding:0; border:0 }
		tr { display:table-row }
		td { display:table-cell; margin:0; padding:0; border:0; font-size:16px; line-height:19.2px }
	`)
	root := BuildDocument(document, engine)
	layout := New(400, 600)
	layout.Layout(root)
	tableBox := findBox(root, table)
	if tableBox.ContentWidth != 100 {
		t.Errorf("table width = %v, want 100", tableBox.ContentWidth)
	}
	if got := findBox(root, first).Width; math.Abs(got-100.0/3) > 1e-9 {
		t.Errorf("first width = %v, want %v", got, 100.0/3)
	}
	if got := findBox(root, second).Width; math.Abs(got-200.0/3) > 1e-9 {
		t.Errorf("second width = %v, want %v", got, 200.0/3)
	}
}

func TestGridLayout(t *testing.T) {
	document, body := testDocument()
	grid := document.CreateElement("div")
	grid.SetAttribute("class", "grid")
	items := []*dom.Node{}
	for index := 0; index < 3; index++ {
		item := document.CreateElement("div")
		item.SetAttribute("class", "item")
		grid.AppendChild(item)
		items = append(items, item)
	}
	body.AppendChild(grid)

	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		.grid { display:grid; width:160px; height:100px; margin:0; padding:0; border:0; grid-template-columns:50px 100px; grid-template-rows:20px 20px; column-gap:10px; row-gap:10px }
		.item { margin:0; padding:0; border:0; height:20px }
	`)
	root := BuildDocument(document, engine)
	layout := New(400, 600)
	layout.Layout(root)

	wantX := []float64{0, 60, 0}
	wantY := []float64{0, 0, 30}
	wantW := []float64{50, 100, 50}
	for index, item := range items {
		box := findBox(root, item)
		if box.X != wantX[index] || box.Y != wantY[index] || box.Width != wantW[index] {
			t.Errorf("grid item %d = x:%v y:%v w:%v, want %v/%v/%v", index, box.X, box.Y, box.Width, wantX[index], wantY[index], wantW[index])
		}
	}
}

func TestGridFrAndPlacement(t *testing.T) {
	document, body := testDocument()
	grid := document.CreateElement("div")
	grid.SetAttribute("class", "fr")
	first := document.CreateElement("div")
	first.SetAttribute("class", "item")
	second := document.CreateElement("div")
	second.SetAttribute("class", "item")
	grid.AppendChild(first)
	grid.AppendChild(second)
	body.AppendChild(grid)

	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		.fr { display:grid; width:300px; height:20px; margin:0; padding:0; border:0; grid-template-columns:1fr 2fr; gap:0 }
		.item { margin:0; padding:0; border:0; height:20px }
	`)
	root := BuildDocument(document, engine)
	layout := New(400, 600)
	layout.Layout(root)
	if got := findBox(root, first).Width; got != 100 {
		t.Errorf("fr first width = %v, want 100", got)
	}
	if got := findBox(root, second).Width; got != 200 {
		t.Errorf("fr second width = %v, want 200", got)
	}
	if got := findBox(root, second).X; got != 100 {
		t.Errorf("fr second x = %v, want 100", got)
	}

	document, body = testDocument()
	placed := document.CreateElement("div")
	placed.SetAttribute("class", "placed")
	explicit := document.CreateElement("div")
	explicit.SetAttribute("class", "explicit")
	auto := document.CreateElement("div")
	auto.SetAttribute("class", "auto")
	placed.AppendChild(explicit)
	placed.AppendChild(auto)
	body.AppendChild(placed)
	engine = testEngine(document, `
		html, body { margin:0; padding:0 }
		.placed { display:grid; width:160px; height:100px; margin:0; padding:0; border:0; grid-template-columns:50px 100px; grid-template-rows:20px 20px; column-gap:10px; row-gap:10px }
		.explicit, .auto { margin:0; padding:0; border:0; height:20px }
		.explicit { grid-column-start:2; grid-row-start:1 }
	`)
	root = BuildDocument(document, engine)
	layout = New(400, 600)
	layout.Layout(root)
	if got := findBox(root, explicit).X; got != 60 {
		t.Errorf("explicit x = %v, want 60", got)
	}
	if got := findBox(root, auto).X; got != 0 {
		t.Errorf("auto x = %v, want 0", got)
	}
}

func TestFloatLayout(t *testing.T) {
	document, body := testDocument()
	left := document.CreateElement("div")
	left.SetAttribute("class", "float-left")
	right := document.CreateElement("div")
	right.SetAttribute("class", "float-right")
	after := document.CreateElement("div")
	after.SetAttribute("class", "after")
	body.AppendChild(left)
	body.AppendChild(right)
	body.AppendChild(after)

	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		.float-left { float:left; width:50px; height:20px; margin:0; padding:0; border:0 }
		.float-right { float:right; width:50px; height:20px; margin:0; padding:0; border:0 }
		.after { display:block; height:10px; margin:0; padding:0; border:0 }
	`)
	root := BuildDocument(document, engine)
	layout := New(400, 600)
	layout.Layout(root)

	leftBox := findBox(root, left)
	rightBox := findBox(root, right)
	afterBox := findBox(root, after)
	if leftBox.X != 0 || leftBox.Y != 0 {
		t.Errorf("left float = (%v,%v)", leftBox.X, leftBox.Y)
	}
	if rightBox.X != 350 || rightBox.Y != 0 {
		t.Errorf("right float = (%v,%v)", rightBox.X, rightBox.Y)
	}
	if afterBox.Y != 20 {
		t.Errorf("after y = %v, want 20", afterBox.Y)
	}
}

func TestOverflowMetrics(t *testing.T) {
	document, body := testDocument()
	parent := document.CreateElement("div")
	parent.SetAttribute("class", "parent")
	child := document.CreateElement("div")
	child.SetAttribute("class", "child")
	parent.AppendChild(child)
	absolute := document.CreateElement("div")
	absolute.SetAttribute("class", "absolute")
	parent.AppendChild(absolute)
	body.AppendChild(parent)

	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		.parent { position:relative; display:block; width:50px; height:20px; margin:0; padding:0; border:0 }
		.child { display:block; width:40px; height:40px; margin:0; padding:0; border:0 }
		.absolute { position:absolute; left:60px; top:0; width:20px; height:10px; margin:0; padding:0; border:0 }
	`)
	root := BuildDocument(document, engine)
	layout := New(400, 600)
	layout.Layout(root)
	parentBox := findBox(root, parent)
	if parentBox.ScrollWidth != 80 || parentBox.ScrollHeight != 40 {
		t.Errorf("parent scroll = %vx%v, want 80x40", parentBox.ScrollWidth, parentBox.ScrollHeight)
	}
}

func TestTextShaper(t *testing.T) {
	glyphs := (SimpleShaper{}).Shape(TextRun{Text: "ab", FontSize: 16})
	if len(glyphs) != 2 || glyphs[0].Advance != 8 || glyphs[1].Advance != 8 {
		t.Errorf("glyphs = %#v", glyphs)
	}
	glyphs = (SimpleShaper{}).Shape(TextRun{Text: "a b", FontSize: 16, LetterSpacing: 2, WordSpacing: 4})
	if len(glyphs) != 3 || glyphs[0].Advance != 10 || glyphs[1].Advance != 14 || glyphs[2].Advance != 10 {
		t.Errorf("spaced glyphs = %#v", glyphs)
	}

	document, body := testDocument()
	paragraph := document.CreateElement("p")
	paragraph.AppendChild(document.CreateTextNode("a b"))
	body.AppendChild(paragraph)
	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		p { display:block; font-size:16px; line-height:19.2px; letter-spacing:2px; word-spacing:4px; margin:0; padding:0; border:0 }
	`)
	styleMap := engine.ComputeDocumentTree(document)
	layout := New(400, 600)
	width, height := layout.measureText("a b", styleMap[paragraph])
	if width != 34 || height != 19.2 {
		t.Errorf("measured = %vx%v, want 34x19.2", width, height)
	}
}

func TestBidiSegmentation(t *testing.T) {
	runs := SegmentBidi("hello", bidi.LeftToRight)
	if len(runs) != 1 || runs[0].Text != "hello" || runs[0].Direction != bidi.LeftToRight {
		t.Errorf("ltr runs = %#v", runs)
	}
	runs = SegmentBidi("abc \u05d0\u05d1\u05d2", bidi.LeftToRight)
	if len(runs) < 2 {
		t.Fatalf("mixed runs = %#v", runs)
	}
	hasRTL := false
	for _, run := range runs {
		if run.Direction == bidi.RightToLeft && run.Text != "" {
			hasRTL = true
		}
	}
	if !hasRTL {
		t.Errorf("mixed runs have no RTL run: %#v", runs)
	}
}

func TestDocumentLayoutCache(t *testing.T) {
	document, body := testDocument()
	paragraph := document.CreateElement("p")
	paragraph.SetAttribute("class", "item")
	paragraph.AppendChild(document.CreateTextNode("hello"))
	body.AppendChild(paragraph)
	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		p { display:block; width:100px; height:10px; margin:0; padding:0; border:0 }
		.wide { width:200px }
	`)
	cache := NewDocumentLayout(document, engine, 400, 600)
	first := cache.Reflow()
	firstParagraph := findBox(first, paragraph)
	if firstParagraph.Width != 100 {
		t.Errorf("first width = %v, want 100", firstParagraph.Width)
	}
	if cache.Reflow() != first {
		t.Error("unchanged document should reuse the cached box tree")
	}

	paragraph.AddClass("wide")
	second := cache.Reflow()
	if second == first {
		t.Error("mutated document should rebuild the box tree")
	}
	if got := findBox(second, paragraph).Width; got != 200 {
		t.Errorf("restyled width = %v, want 200", got)
	}

	cache.ViewportWidth = 800
	cache.Invalidate()
	third := cache.Reflow()
	if third == second {
		t.Error("invalidated viewport should rebuild the box tree")
	}
}

func TestFlexWrapAndBaseline(t *testing.T) {
	document, body := testDocument()
	flex := document.CreateElement("div")
	flex.SetAttribute("class", "wrap")
	items := []*dom.Node{}
	for index := 0; index < 3; index++ {
		item := document.CreateElement("div")
		item.SetAttribute("class", "item")
		flex.AppendChild(item)
		items = append(items, item)
	}
	body.AppendChild(flex)
	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		.wrap { display:flex; flex-wrap:wrap; width:120px; column-gap:10px; row-gap:0; align-items:flex-start }
		.item { width:50px; height:20px; margin:0; padding:0; border:0 }
	`)
	root := BuildDocument(document, engine)
	layout := New(400, 600)
	layout.Layout(root)
	if findBox(root, items[0]).X != 0 || findBox(root, items[1]).X != 60 {
		t.Errorf("first line x = %v/%v", findBox(root, items[0]).X, findBox(root, items[1]).X)
	}
	if findBox(root, items[2]).X != 0 || findBox(root, items[2]).Y != 20 {
		t.Errorf("wrapped item = (%v,%v), want (0,20)", findBox(root, items[2]).X, findBox(root, items[2]).Y)
	}

	document, body = testDocument()
	baseline := document.CreateElement("div")
	baseline.SetAttribute("class", "baseline")
	small := document.CreateElement("span")
	small.SetAttribute("class", "small")
	small.AppendChild(document.CreateTextNode("x"))
	large := document.CreateElement("span")
	large.SetAttribute("class", "large")
	large.AppendChild(document.CreateTextNode("x"))
	baseline.AppendChild(small)
	baseline.AppendChild(large)
	body.AppendChild(baseline)
	engine = testEngine(document, `
		html, body { margin:0; padding:0 }
		.baseline { display:flex; width:200px; align-items:baseline }
		.small { width:50px; margin:0; padding:0; border:0; font-size:16px }
		.large { width:50px; margin:0; padding:0; border:0; font-size:24px }
	`)
	root = BuildDocument(document, engine)
	layout = New(400, 600)
	layout.Layout(root)
	if got := findBox(root, small).Y; math.Abs(got-6.4) > 1e-9 {
		t.Errorf("small baseline y = %v, want 6.4", got)
	}
	if got := findBox(root, large).Y; got != 0 {
		t.Errorf("large baseline y = %v, want 0", got)
	}
}

func TestScrollContainer(t *testing.T) {
	document, body := testDocument()
	parent := document.CreateElement("div")
	parent.SetAttribute("class", "scroller")
	child := document.CreateElement("div")
	child.SetAttribute("class", "child")
	parent.AppendChild(child)
	body.AppendChild(parent)
	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		.scroller { display:block; width:50px; height:20px; overflow:auto; margin:0; padding:0; border:0 }
		.child { display:block; width:100px; height:40px; margin:0; padding:0; border:0 }
	`)
	root := BuildDocument(document, engine)
	layout := New(400, 600)
	layout.Layout(root)
	scroller := findBox(root, parent)
	if !scroller.ClipX || !scroller.ClipY {
		t.Error("overflow:auto should clip both axes")
	}
	if scroller.ScrollWidth != 100 || scroller.ScrollHeight != 40 {
		t.Errorf("scroll size = %vx%v, want 100x40", scroller.ScrollWidth, scroller.ScrollHeight)
	}
	if scroller.MaxScrollLeft() != 50 || scroller.MaxScrollTop() != 20 {
		t.Errorf("max scroll = %v/%v, want 50/20", scroller.MaxScrollLeft(), scroller.MaxScrollTop())
	}
	scroller.ScrollTo(100, 100)
	if scroller.ScrollLeft != 50 || scroller.ScrollTop != 20 {
		t.Errorf("clamped scroll = %v/%v, want 50/20", scroller.ScrollLeft, scroller.ScrollTop)
	}
	scroller.ScrollBy(-10, -5)
	if scroller.ScrollLeft != 40 || scroller.ScrollTop != 15 {
		t.Errorf("relative scroll = %v/%v, want 40/15", scroller.ScrollLeft, scroller.ScrollTop)
	}

	document, body = testDocument()
	visible := document.CreateElement("div")
	visible.SetAttribute("class", "visible")
	visible.AppendChild(document.CreateTextNode("x"))
	body.AppendChild(visible)
	engine = testEngine(document, `
		html, body { margin:0; padding:0 }
		.visible { display:block; width:50px; height:20px; overflow:visible; margin:0; padding:0; border:0 }
	`)
	root = BuildDocument(document, engine)
	layout = New(400, 600)
	layout.Layout(root)
	if findBox(root, visible).ClipX || findBox(root, visible).ClipY {
		t.Error("overflow:visible should not clip")
	}
}

func TestInlineBidiOrdering(t *testing.T) {
	document, body := testDocument()
	ltr := document.CreateElement("p")
	ltr.SetAttribute("class", "ltr")
	ltr.AppendChild(document.CreateTextNode("hello world"))
	rtl := document.CreateElement("p")
	rtl.SetAttribute("class", "rtl")
	rtl.AppendChild(document.CreateTextNode("שלום עולם"))
	body.AppendChild(ltr)
	body.AppendChild(rtl)
	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		p { display:block; width:400px; font-size:16px; line-height:19.2px; margin:0; padding:0; border:0 }
		.ltr { direction:ltr }
		.rtl { direction:rtl }
	`)
	root := BuildDocument(document, engine)
	layout := New(800, 600)
	layout.Layout(root)
	ltrBox := findBox(root, ltr)
	if len(ltrBox.Children) != 2 || ltrBox.Children[0].Text != "hello" || ltrBox.Children[1].Text != "world" {
		t.Errorf("ltr words = %#v", ltrBox.Children)
	}
	if ltrBox.Children[0].VisualText != "hello" {
		t.Errorf("ltr visual text = %q", ltrBox.Children[0].VisualText)
	}
	rtlBox := findBox(root, rtl)
	if len(rtlBox.Children) != 2 || rtlBox.Children[0].Text != "עולם" || rtlBox.Children[1].Text != "שלום" {
		t.Errorf("rtl words = %#v", rtlBox.Children)
	}
	if rtlBox.Children[0].VisualText != "םלוע" || rtlBox.Children[1].VisualText != "םולש" {
		t.Errorf("rtl visual text = %q/%q", rtlBox.Children[0].VisualText, rtlBox.Children[1].VisualText)
	}
}

func TestFontMetrics(t *testing.T) {
	document, body := testDocument()
	paragraph := document.CreateElement("p")
	body.AppendChild(paragraph)
	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		p { display:block; font-family: monospace; font-size:16px; line-height:19.2px; margin:0; padding:0; border:0 }
	`)
	styleMap := engine.ComputeDocumentTree(document)
	metrics := MetricsForStyle(styleMap[paragraph])
	if metrics.Ascent != 12.8 || metrics.Descent != 3.2 || metrics.AverageAdvance != 9.6 {
		t.Errorf("metrics = %#v", metrics)
	}
	layout := New(400, 600)
	width, _ := layout.measureText("ab", styleMap[paragraph])
	if math.Abs(width-19.2) > 1e-9 {
		t.Errorf("monospace width = %v, want 19.2", width)
	}
}
