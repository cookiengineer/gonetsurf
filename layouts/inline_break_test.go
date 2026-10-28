package layouts

import (
	"strings"
	"testing"
)

func TestJustifyDoesNotStretchForcedBreakLine(t *testing.T) {
	document, body := testDocument()
	paragraph := document.CreateElement("p")
	paragraph.AppendChild(document.CreateTextNode("alpha beta"))
	paragraph.AppendChild(document.CreateElement("br"))
	paragraph.AppendChild(document.CreateTextNode("gamma delta"))
	body.AppendChild(paragraph)
	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		p { display:block; width:300px; margin:0; text-align:justify }
	`)
	root := BuildDocument(document, engine)
	New(400, 400).Layout(root)
	paragraphBox := FindBox(root, paragraph)
	if paragraphBox == nil {
		t.Fatal("paragraph box missing")
	}
	for _, child := range paragraphBox.Children {
		if child.Kind != TextBox || !strings.Contains(child.Text, "beta") {
			continue
		}
		if child.X > 120 {
			t.Errorf("forced-break line was justified: beta.X = %v", child.X)
		}
		return
	}
	t.Fatal("beta text box missing")
}

func TestCalcWidthResolvesAgainstContainingBlock(t *testing.T) {
	document, body := testDocument()
	container := document.CreateElement("div")
	container.SetAttribute("id", "container")
	cell := document.CreateElement("div")
	cell.SetAttribute("id", "cell")
	cell.SetAttribute("style", "width:calc(100% - 64px)")
	container.AppendChild(cell)
	body.AppendChild(container)
	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		#container { display:block; width:1000px }
		#cell { display:block }
	`)
	root := BuildDocument(document, engine)
	New(1200, 800).Layout(root)
	containerBox := FindBox(root, container)
	cellBox := FindBox(root, cell)
	if containerBox == nil || cellBox == nil {
		t.Fatal("boxes missing")
	}
	if cellBox.Width != 936 {
		t.Errorf("calc width = %v, want 936", cellBox.Width)
	}
}

func TestAutoMarginsCenterBlock(t *testing.T) {
	document, body := testDocument()
	container := document.CreateElement("div")
	container.SetAttribute("id", "container")
	child := document.CreateElement("div")
	child.SetAttribute("id", "child")
	container.AppendChild(child)
	body.AppendChild(container)
	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		#container { display:block; width:1000px }
		#child { display:block; width:640px; margin:0 auto }
	`)
	root := BuildDocument(document, engine)
	New(1200, 800).Layout(root)
	childBox := FindBox(root, child)
	if childBox == nil {
		t.Fatal("child box missing")
	}
	if childBox.X != 180 {
		t.Errorf("centered X = %v, want 180", childBox.X)
	}
}

func TestClippedOverflowDoesNotExpandDocument(t *testing.T) {
	document, body := testDocument()
	scroller := document.CreateElement("div")
	scroller.SetAttribute("id", "scroller")
	content := document.CreateElement("div")
	content.SetAttribute("id", "content")
	scroller.AppendChild(content)
	body.AppendChild(scroller)
	engine := testEngine(document, `
		html, body { margin:0; padding:0; height:100% }
		#scroller { display:block; width:200px; height:100px; overflow:hidden }
		#content { display:block; width:200px; height:1000px }
	`)
	root := BuildDocument(document, engine)
	New(200, 100).Layout(root)
	if root.ScrollHeight > 100 {
		t.Errorf("root scroll height = %v, want viewport height", root.ScrollHeight)
	}
	scrollerBox := FindBox(root, scroller)
	if scrollerBox == nil || scrollerBox.ScrollHeight < 1000 {
		t.Errorf("inner scroll height = %v", scrollerBox.ScrollHeight)
	}
}

func TestMultiKeywordFlexDisplay(t *testing.T) {
	document, body := testDocument()
	header := document.CreateElement("header")
	first := document.CreateElement("div")
	second := document.CreateElement("div")
	third := document.CreateElement("div")
	header.AppendChild(first)
	header.AppendChild(second)
	header.AppendChild(third)
	body.AppendChild(header)
	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		header { display:block flex; justify-content:space-between; align-items:center; height:100px }
		header div { display:block; width:20px; height:20px }
	`)
	root := BuildDocument(document, engine)
	New(300, 200).Layout(root)
	firstBox := FindBox(root, first)
	thirdBox := FindBox(root, third)
	if firstBox == nil || thirdBox == nil {
		t.Fatal("flex children missing")
	}
	if firstBox.X != 0 || thirdBox.X != 280 {
		t.Errorf("flex positions = %v / %v", firstBox.X, thirdBox.X)
	}
	if firstBox.Y != 40 {
		t.Errorf("flex cross-axis position = %v, want 40", firstBox.Y)
	}
}

func TestAbsoluteBoxStretchesBetweenTopAndBottom(t *testing.T) {
	document, body := testDocument()
	container := document.CreateElement("div")
	container.SetAttribute("id", "container")
	aside := document.CreateElement("aside")
	aside.SetAttribute("id", "aside")
	container.AppendChild(aside)
	body.AppendChild(container)
	engine := testEngine(document, `
		html, body { margin:0; padding:0 }
		#container { display:block; position:relative; width:200px; height:100px; overflow:hidden }
		#aside { display:block; position:absolute; left:0; top:0; bottom:0; width:20px }
	`)
	root := BuildDocument(document, engine)
	New(300, 200).Layout(root)
	asideBox := FindBox(root, aside)
	if asideBox == nil || asideBox.Y != 0 || asideBox.Height != 100 {
		t.Errorf("aside box = %#v", asideBox)
	}
}
