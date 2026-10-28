package styles

import (
	"testing"

	"gonetsurf/dom"
	"gonetsurf/parsers/css/colors"
	"gonetsurf/parsers/css/cssom"
	"gonetsurf/parsers/css/mediaqueries"
)

func canonicalColor(input string) string {
	color, ok := colors.Parse(input)
	if !ok {
		return input
	}
	return color.Serialize()
}

func engineFixture(css string) (*dom.Document, *dom.Node, *dom.Node, *dom.Node, *Engine) {
	document := dom.NewDocument()
	html := document.CreateElement("html")
	body := document.CreateElement("body")
	paragraph := document.CreateElement("p")
	body.AppendChild(paragraph)
	html.AppendChild(body)
	document.AppendChild(html)
	engine := NewEngine(document)
	if css != "" {
		engine.AddStylesheet(cssom.Parse(css))
	}
	return document, html, body, paragraph, engine
}

func TestStyleInheritanceAndUA(t *testing.T) {
	document, _, body, paragraph, engine := engineFixture("html { color: red } p { margin-top: 10px }")
	styles := engine.ComputeDocumentTree(document)
	if got := styles[paragraph].GetString("color"); got != canonicalColor("red") {
		t.Errorf("p color = %q, want red", got)
	}
	if got := styles[body].GetString("color"); got != canonicalColor("red") {
		t.Errorf("body color = %q, want red", got)
	}
	if got := styles[paragraph].GetString("margin-top"); got != "10px" {
		t.Errorf("p margin-top = %q, want 10px", got)
	}
	if got := styles[paragraph].GetString("display"); got != "block" {
		t.Errorf("p display = %q, want block", got)
	}
}

func TestStyleCustomProperties(t *testing.T) {
	document, _, _, paragraph, engine := engineFixture(`
		:root { --brand: green }
		p { color: var(--brand); --local: var(--brand) }
	`)
	styles := engine.ComputeDocumentTree(document)
	if got := styles[paragraph].GetString("color"); got != canonicalColor("green") {
		t.Errorf("color = %q, want green", got)
	}
	if got := styles[paragraph].GetString("--local"); got != "green" {
		t.Errorf("--local = %q, want green", got)
	}

	document, _, _, paragraph, engine = engineFixture("p { color: var(--missing, blue) }")
	styles = engine.ComputeDocumentTree(document)
	if got := styles[paragraph].GetString("color"); got != canonicalColor("blue") {
		t.Errorf("fallback color = %q, want blue", got)
	}

	document, _, _, paragraph, engine = engineFixture(`
		p { --a: var(--b); --b: var(--a); color: var(--a, red) }
	`)
	styles = engine.ComputeDocumentTree(document)
	if got := styles[paragraph].GetString("color"); got != canonicalColor("red") {
		t.Errorf("cycle color = %q, want red", got)
	}
	if got := styles[paragraph].Get("--a"); got != nil {
		t.Errorf("cyclic --a = %#v, want nil", got)
	}
}

func TestStyleMediaAndSupports(t *testing.T) {
	document, _, _, paragraph, engine := engineFixture(`
		@media (min-width: 600px) { p { color: blue } }
		@supports (display: grid) { p { margin-top: 5px } }
	`)
	styles := engine.ComputeDocumentTree(document)
	if got := styles[paragraph].GetString("color"); got != canonicalColor("blue") {
		t.Errorf("media color = %q, want blue", got)
	}
	if got := styles[paragraph].GetString("margin-top"); got != "5px" {
		t.Errorf("supports margin-top = %q, want 5px", got)
	}

	environment := mediaqueries.DefaultEnvironment()
	environment.Width = 400
	engine.SetMediaEnvironment(environment)
	styles = engine.ComputeDocumentTree(document)
	if got := styles[paragraph].GetString("color"); got == canonicalColor("blue") {
		t.Errorf("narrow media color = %q, want non-blue", got)
	}
	if got := styles[paragraph].GetString("margin-top"); got != "5px" {
		t.Errorf("supports margin-top after media change = %q, want 5px", got)
	}
}

func TestStyleInlineAndHints(t *testing.T) {
	document, _, _, _, engine := engineFixture("img.first { width: 20px } p { color: red }")
	image := document.CreateElement("img")
	image.SetAttribute("class", "first")
	image.SetAttribute("width", "10")
	document.Body().AppendChild(image)
	paragraph := document.CreateElement("p")
	paragraph.SetAttribute("style", "color: purple")
	document.Body().AppendChild(paragraph)
	styles := engine.ComputeDocumentTree(document)
	if got := styles[image].GetString("width"); got != "20px" {
		t.Errorf("author width = %q, want 20px", got)
	}
	if got := styles[paragraph].GetString("color"); got != canonicalColor("purple") {
		t.Errorf("inline color = %q, want purple", got)
	}

	second := document.CreateElement("img")
	second.SetAttribute("width", "10")
	document.Body().AppendChild(second)
	engine.Rebuild()
	styles = engine.ComputeDocumentTree(document)
	if got := styles[second].GetString("width"); got != "10px" {
		t.Errorf("hint width = %q, want 10px", got)
	}
}

func TestStyleNestedRules(t *testing.T) {
	document, _, _, paragraph, engine := engineFixture(`
		div { & p { color: green } }
		div { p { margin-top: 1px } }
	`)
	div := document.CreateElement("div")
	div.AppendChild(paragraph)
	document.Body().AppendChild(div)
	engine.Rebuild()
	styles := engine.ComputeDocumentTree(document)
	if got := styles[paragraph].GetString("color"); got != canonicalColor("green") {
		t.Errorf("nested color = %q, want green", got)
	}
	if got := styles[paragraph].GetString("margin-top"); got != "1px" {
		t.Errorf("implicit nested margin = %q, want 1px", got)
	}
}

func TestStyleAllShorthand(t *testing.T) {
	document, _, _, paragraph, engine := engineFixture("p { all: initial }")
	styles := engine.ComputeDocumentTree(document)
	if got := styles[paragraph].GetString("display"); got != "inline" {
		t.Errorf("all:initial display = %q, want inline", got)
	}
}

func TestComputedRelativeUnits(t *testing.T) {
	document, _, _, paragraph, engine := engineFixture(`
		html { font-size: 20px }
		body { font-size: 1.5em }
		p {
			font-size: 2em;
			margin-left: 1em;
			width: 10vw;
			line-height: 150%;
			border-top-width: thin;
			opacity: 50%;
			color: blue;
			background-color: currentcolor;
		}
	`)
	span := document.CreateElement("span")
	paragraph.AppendChild(span)
	engine.Rebuild()
	styles := engine.ComputeDocumentTree(document)
	if got := styles[paragraph].GetString("font-size"); got != "60px" {
		t.Errorf("p font-size = %q, want 60px", got)
	}
	if got := styles[span].GetString("font-size"); got != "60px" {
		t.Errorf("span font-size = %q, want 60px", got)
	}
	if got := styles[paragraph].GetString("margin-left"); got != "60px" {
		t.Errorf("margin-left = %q, want 60px", got)
	}
	if got := styles[paragraph].GetString("width"); got != "128px" {
		t.Errorf("width = %q, want 128px", got)
	}
	if got := styles[paragraph].GetString("line-height"); got != "90px" {
		t.Errorf("line-height = %q, want 90px", got)
	}
	if got := styles[paragraph].GetString("border-top-width"); got != "1px" {
		t.Errorf("border-top-width = %q, want 1px", got)
	}
	if got := styles[paragraph].GetString("opacity"); got != "0.5" {
		t.Errorf("opacity = %q, want 0.5", got)
	}
	if got := styles[paragraph].GetString("background-color"); got != canonicalColor("blue") {
		t.Errorf("currentcolor background = %q", got)
	}
}

func TestComputedRemAndKeywords(t *testing.T) {
	document, _, _, paragraph, engine := engineFixture(`
		html { font-size: 20px }
		p { font-size: 1rem; font-weight: normal; line-height: 1.5 }
	`)
	styles := engine.ComputeDocumentTree(document)
	if got := styles[paragraph].GetString("font-size"); got != "20px" {
		t.Errorf("rem font-size = %q, want 20px", got)
	}
	if got := styles[paragraph].GetString("font-weight"); got != "400" {
		t.Errorf("font-weight = %q, want 400", got)
	}
	if got := styles[paragraph].GetString("line-height"); got != "1.5" {
		t.Errorf("line-height = %q, want 1.5", got)
	}
	document, _, _, paragraph, engine = engineFixture("p { font-size: large }")
	styles = engine.ComputeDocumentTree(document)
	if got := styles[paragraph].GetString("font-size"); got != "18px" {
		t.Errorf("large font-size = %q, want 18px", got)
	}
}

func TestComputedCalc(t *testing.T) {
	document, _, _, paragraph, engine := engineFixture(`
		html { font-size: 20px }
		p { width: calc(2em + 10px) }
	`)
	styles := engine.ComputeDocumentTree(document)
	if got := styles[paragraph].GetString("width"); got != "50px" {
		t.Errorf("calc width = %q, want 50px", got)
	}
}

func TestComputedFontWeightRelative(t *testing.T) {
	document, _, _, paragraph, engine := engineFixture(`
		html { font-weight: bold }
		p { font-weight: bolder }
	`)
	styles := engine.ComputeDocumentTree(document)
	if got := styles[paragraph].GetString("font-weight"); got != "900" {
		t.Errorf("bolder weight = %q, want 900", got)
	}
}

func TestIncrementalRestyle(t *testing.T) {
	document := dom.NewDocument()
	body := document.CreateElement("body")
	one := document.CreateElement("div")
	one.SetAttribute("id", "one")
	first := document.CreateElement("p")
	first.SetAttribute("class", "first")
	second := document.CreateElement("p")
	second.SetAttribute("class", "second")
	one.AppendChild(first)
	one.AppendChild(second)
	two := document.CreateElement("div")
	two.SetAttribute("id", "two")
	third := document.CreateElement("p")
	third.SetAttribute("class", "third")
	two.AppendChild(third)
	body.AppendChild(one)
	body.AppendChild(two)
	document.AppendChild(body)

	engine := NewEngine(document)
	engine.AddStylesheet(cssom.Parse(`
		p { color: red }
		.active { color: blue }
		p + p { margin-top: 10px }
	`))
	styles := engine.ComputeDocumentTree(document)
	thirdStyle := styles[third]
	if got := styles[first].GetString("color"); got != canonicalColor("red") {
		t.Fatalf("first color = %q", got)
	}
	first.AddClass("active")
	styles = engine.Restyles(document)
	if got := styles[first].GetString("color"); got != canonicalColor("blue") {
		t.Errorf("restyled first color = %q, want blue", got)
	}
	if styles[third] != thirdStyle {
		t.Error("unrelated subtree should reuse its cached style pointer")
	}
	if got := styles[second].GetString("margin-top"); got != "10px" {
		t.Errorf("sibling margin = %q, want 10px", got)
	}

	added := document.CreateElement("p")
	added.SetAttribute("class", "added")
	one.AppendChild(added)
	styles = engine.Restyles(document)
	if got := styles[added].GetString("color"); got != canonicalColor("red") {
		t.Errorf("added node color = %q, want red", got)
	}

	one.RemoveChild(second)
	engine.Restyles(document)
	if _, ok := styles[second]; ok {
		t.Error("removed node should not remain in the style cache")
	}
	if got := styles[first].GetString("margin-top"); got != "16px" {
		t.Errorf("sibling margin after removal = %q, want the UA 1em margin", got)
	}
}

func TestIncrementalRestyleFallsBackForHas(t *testing.T) {
	document := dom.NewDocument()
	div := document.CreateElement("div")
	flag := document.CreateElement("span")
	flag.SetAttribute("class", "flag")
	div.AppendChild(flag)
	document.AppendChild(div)
	engine := NewEngine(document)
	engine.AddStylesheet(cssom.Parse(`
		div:has(.flag) { color: green }
		.flag { color: blue }
	`))
	styles := engine.ComputeDocumentTree(document)
	if got := styles[div].GetString("color"); got != canonicalColor("green") {
		t.Fatalf("div color = %q, want green", got)
	}
	flag.RemoveClass("flag")
	styles = engine.Restyles(document)
	if got := styles[div].GetString("color"); got == canonicalColor("green") {
		t.Errorf("div color after :has invalidation = %q, want non-green", got)
	}
}

func TestIncrementalRestyleNoMutation(t *testing.T) {
	document, _, _, _, engine := engineFixture("p { color: red }")
	first := engine.ComputeDocumentTree(document)
	second := engine.Restyles(document)
	if len(first) != len(second) {
		t.Errorf("cache size changed without mutation: %d -> %d", len(first), len(second))
	}
}
