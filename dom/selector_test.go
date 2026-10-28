package dom

import "testing"

func selectorFixture() (*Document, *Node, *Node, *Node, *Node) {
	document := NewDocument()
	html := document.CreateElement("html")
	html.SetAttribute("lang", "en-US")
	body := document.CreateElement("body")
	main := document.CreateElement("div")
	main.SetAttribute("id", "main")
	main.SetAttribute("class", "container")
	main.SetAttribute("data-role", "main")
	first := document.CreateElement("p")
	first.SetAttribute("class", "first")
	first.SetAttribute("title", "Hello")
	first.AppendChild(document.CreateTextNode("first"))
	second := document.CreateElement("p")
	second.SetAttribute("class", "second")
	second.AppendChild(document.CreateTextNode("second"))
	span := document.CreateElement("span")
	main.AppendChild(first)
	main.AppendChild(second)
	main.AppendChild(span)
	list := document.CreateElement("ul")
	list.SetAttribute("id", "list")
	for index, class := range []string{"one", "two", "three"} {
		item := document.CreateElement("li")
		item.SetAttribute("class", class)
		if index == 0 {
			item.SetAttribute("lang", "en-GB")
		}
		list.AppendChild(item)
	}
	link := document.CreateElement("a")
	link.SetAttribute("href", "/x")
	link.AppendChild(document.CreateTextNode("link"))
	input := document.CreateElement("input")
	input.SetAttribute("type", "checkbox")
	input.SetAttribute("checked", "")
	textInput := document.CreateElement("input")
	textInput.SetAttribute("type", "text")
	disabled := document.CreateElement("input")
	disabled.SetAttribute("disabled", "")
	body.AppendChild(main)
	body.AppendChild(list)
	body.AppendChild(link)
	body.AppendChild(input)
	body.AppendChild(textInput)
	body.AppendChild(disabled)
	html.AppendChild(body)
	document.AppendChild(html)
	return document, main, first, second, span
}

func TestSelectorCombinatorsAndAttributes(t *testing.T) {
	document, main, first, _, _ := selectorFixture()
	tests := []struct {
		selector string
		node     *Node
		want     bool
	}{
		{"div#main.container", main, true},
		{"body > div#main", main, true},
		{"body > p", first, false},
		{"div p", first, true},
		{"div > p.first", first, true},
		{"[data-role]", main, true},
		{"[data-role=main]", main, true},
		{"[title='Hello']", first, true},
		{"[title^='Hel']", first, true},
		{"[title$='lo']", first, true},
		{"[title*='ell']", first, true},
		{"[title='hello' i]", first, true},
		{"[class~='container']", main, true},
		{"[lang|=en]", first, false},
		{"html[lang|=en]", document.DocumentElement(), true},
		{"p:not(.first)", first, false},
		{"p:is(.first, .second)", first, true},
		{"p:where(.first)", first, true},
		{"div:has(> p.first)", main, true},
		{"div:has(> span)", main, true},
		{"div:has(.missing)", main, false},
	}
	for _, test := range tests {
		if got := test.node.Matches(test.selector); got != test.want {
			t.Errorf("Matches(%q) = %v, want %v", test.selector, got, test.want)
		}
	}
}

func TestSelectorStructuralPseudoClasses(t *testing.T) {
	document, main, first, second, span := selectorFixture()
	list := document.GetElementById("list")
	items := list.Children()
	one, two, three := items[0], items[1], items[2]

	tests := []struct {
		selector string
		node     *Node
		want     bool
	}{
		{":root", document.DocumentElement(), true},
		{"p:first-child", first, true},
		{"p:last-child", second, false},
		{"span:last-child", span, true},
		{"span:empty", span, true},
		{"li:nth-child(2)", two, true},
		{"li:nth-child(odd)", one, true},
		{"li:nth-child(odd)", two, false},
		{"li:nth-child(2n+1)", three, true},
		{"li:nth-last-child(1)", three, true},
		{"li:nth-of-type(2)", two, true},
		{"li:only-child", one, false},
		{"li:first-of-type", one, true},
		{"li:last-of-type", three, true},
		{"li:only-of-type", one, false},
		{"li:nth-child(2 of .two, .three)", three, true},
	}
	for _, test := range tests {
		if got := test.node.Matches(test.selector); got != test.want {
			t.Errorf("Matches(%q) = %v, want %v", test.selector, got, test.want)
		}
	}
	if got := document.QuerySelectorAll(":scope"); len(got) != 1 || got[0] != document.DocumentElement() {
		t.Errorf("document :scope = %v", got)
	}
	if got := main.QuerySelectorAll(":scope > p"); len(got) != 2 {
		t.Errorf("scope children = %d, want 2", len(got))
	}
	if got := main.QuerySelectorAll("div p"); len(got) != 2 {
		t.Errorf("descendants = %d, want 2", len(got))
	}
}

func TestSelectorSiblingCombinators(t *testing.T) {
	document, _, _, _, _ := selectorFixture()
	list := document.GetElementById("list")
	items := list.Children()
	if got := document.QuerySelectorAll("li + li"); len(got) != 2 {
		t.Errorf("li + li = %d, want 2", len(got))
	}
	if got := document.QuerySelectorAll("li.one ~ li"); len(got) != 2 {
		t.Errorf("li.one ~ li = %d, want 2", len(got))
	}
	if got := document.QuerySelector("li.two + li"); got != items[2] {
		t.Errorf("li.two + li = %v", got)
	}
	if got := document.QuerySelector("li.three + li"); got != nil {
		t.Errorf("li.three + li = %v, want nil", got)
	}
}

func TestSelectorHasScoping(t *testing.T) {
	document := NewDocument()
	outer := document.CreateElement("div")
	outer.SetAttribute("class", "x")
	anchor := document.CreateElement("div")
	anchor.SetAttribute("id", "anchor")
	child := document.CreateElement("span")
	child.SetAttribute("class", "y")
	anchor.AppendChild(child)
	outer.AppendChild(anchor)
	document.AppendChild(outer)
	if anchor.Matches(":has(.x .y)") {
		t.Error(":has(.x .y) must not reach outside the anchor")
	}
	if !anchor.Matches(":has(.y)") {
		t.Error(":has(.y) should match a descendant")
	}
	if !anchor.Matches(":has(> span.y)") {
		t.Error(":has(> span.y) should match a child")
	}
	if anchor.Matches(":has(> .missing)") {
		t.Error(":has(> .missing) should not match")
	}
	if outer.Matches(":has(.x .y)") {
		t.Error(":has(.x .y) must not treat the anchor as a descendant")
	}
	if !outer.Matches(":has(.y)") {
		t.Error(":has(.y) should match a descendant")
	}
}

func TestSelectorStatesAndLanguages(t *testing.T) {
	document, _, first, _, _ := selectorFixture()
	html := document.DocumentElement()
	link := document.QuerySelector("a[href]")
	input := document.QuerySelector("input[type=checkbox]")
	textInput := document.QuerySelector("input[type=text]")
	disabled := document.QuerySelectorAll("input")[2]
	tests := []struct {
		selector string
		node     *Node
		want     bool
	}{
		{":lang(en)", html, true},
		{":lang(en-US)", first, true},
		{":lang(en-GB)", first, false},
		{":link", link, true},
		{":any-link", link, true},
		{":checked", input, true},
		{":disabled", disabled, true},
		{":enabled", input, true},
		{":read-write", textInput, true},
		{":read-write", input, false},
	}
	for _, test := range tests {
		if got := test.node.Matches(test.selector); got != test.want {
			t.Errorf("Matches(%q) = %v, want %v", test.selector, got, test.want)
		}
	}
}

func TestSelectorNamespaces(t *testing.T) {
	document := NewDocument()
	svg := document.CreateElementNS(NamespaceSVG, "svg")
	rect := document.CreateElementNS(NamespaceSVG, "rect")
	svg.AppendChild(rect)
	document.AppendChild(svg)
	if !rect.Matches("rect") {
		t.Error("plain type selector should match")
	}
	if !rect.Matches("*|rect") {
		t.Error("*|rect should match")
	}
	if rect.Matches("html|rect") {
		t.Error("html|rect should not match")
	}
}

func TestSelectorInvalid(t *testing.T) {
	_, main, _, _, _ := selectorFixture()
	if main.Matches("div >") {
		t.Error("invalid selector must not match")
	}
	if main.QuerySelector("div >") != nil {
		t.Error("invalid selector must not return a node")
	}
}

func BenchmarkSelectorMatching(b *testing.B) {
	document, _, _, _, _ := selectorFixture()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		_ = document.QuerySelectorAll("div > p.first, li:nth-child(2n+1):not(.two)")
	}
}
