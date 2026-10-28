package layouts

import (
	"math"
	"testing"

	"gonetsurf/dom"
	"gonetsurf/parsers/css/cssom"
	"gonetsurf/styles"
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
