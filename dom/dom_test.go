package dom

import "testing"

func TestTreeConstruction(t *testing.T) {
	document := NewDocument()
	html := document.CreateElement("html")
	body := document.CreateElement("body")
	paragraph := document.CreateElement("p")
	paragraph.SetAttribute("id", "intro")
	paragraph.SetAttribute("class", "lead highlight")
	paragraph.AppendChild(document.CreateTextNode("hello "))
	emphasis := document.CreateElement("em")
	emphasis.AppendChild(document.CreateTextNode("world"))
	paragraph.AppendChild(emphasis)

	body.AppendChild(paragraph)
	html.AppendChild(body)
	document.AppendChild(html)

	if paragraph.Parent() != body || body.Parent() != html {
		t.Fatal("unexpected parent links")
	}
	if document.DocumentElement() != html {
		t.Fatal("DocumentElement mismatch")
	}
	if document.Body() != body {
		t.Fatal("Body mismatch")
	}
	if got := paragraph.TextContent(); got != "hello world" {
		t.Errorf("TextContent = %q", got)
	}
	if document.GetElementById("intro") != paragraph {
		t.Error("GetElementById failed")
	}
	if !paragraph.HasClass("highlight") || paragraph.HasClass("missing") {
		t.Error("class matching failed")
	}
	if document.QuerySelector("p#intro.lead") != paragraph {
		t.Error("QuerySelector failed")
	}
	if len(document.QuerySelectorAll("p, em")) != 2 {
		t.Error("QuerySelectorAll failed")
	}
}

func TestMutation(t *testing.T) {
	document := NewDocument()
	root := document.CreateElement("div")
	document.AppendChild(root)
	first := document.CreateElement("span")
	second := document.CreateElement("b")
	root.AppendChild(first)
	root.InsertBefore(second, first)
	if root.FirstChild() != second || root.LastChild() != first {
		t.Fatalf("unexpected order: %v %v", root.FirstChild().TagName(), root.LastChild().TagName())
	}
	root.RemoveChild(second)
	if root.FirstChild() != first {
		t.Error("RemoveChild failed")
	}
	root.SetTextContent("plain")
	if root.FirstChild().Type != TextNode || root.TextContent() != "plain" {
		t.Error("SetTextContent failed")
	}
}

func TestCloneNode(t *testing.T) {
	document := NewDocument()
	root := document.CreateElement("div")
	root.SetAttribute("data-x", "1")
	root.AppendChild(document.CreateTextNode("child"))
	clone := root.CloneNode(true)
	if clone.Parent() != nil {
		t.Error("clone must be detached")
	}
	if clone.AttributeValue("data-x") != "1" || clone.TextContent() != "child" {
		t.Errorf("clone = %+v", clone)
	}
}

func TestTemplateAndShadow(t *testing.T) {
	document := NewDocument()
	element := document.CreateElement("template")
	element.AppendChild(document.CreateTextNode("placeholder"))
	contents := document.CreateDocumentFragment()
	contents.AppendChild(document.CreateTextNode("content"))
	element.SetTemplateContents(contents)
	if element.TemplateContents() == nil || element.TemplateContents().TextContent() != "content" {
		t.Error("template contents failed")
	}
	shadow := element.AttachShadow()
	if shadow == nil || element.ShadowRoot() != shadow {
		t.Error("shadow scaffolding failed")
	}
}

func TestSandboxDirective(t *testing.T) {
	flags := ParseSandboxDirective("allow-scripts allow-same-origin")
	if flags&SandboxScripts != 0 || flags&SandboxOrigin != 0 {
		t.Error("allow-* tokens should clear restrictions")
	}
	if flags&SandboxForms == 0 {
		t.Error("unmentioned restrictions must remain set")
	}
}

func TestOriginAccess(t *testing.T) {
	first := NewDocument()
	first.Origin = OriginFromParts("https", "example.com", "443")
	second := NewDocument()
	second.Origin = OriginFromParts("https", "example.com", "443")
	if !first.SameOrigin(second) {
		t.Error("expected same origin")
	}
	third := NewDocument()
	third.Origin = OriginFromParts("http", "example.com", "80")
	if first.SameOrigin(third) {
		t.Error("different schemes must not share an origin")
	}
	if first.SameOrigin(nil) {
		t.Error("nil document must not share an origin")
	}
}
