package styles

import (
	"testing"

	"gonetsurf/dom"
	"gonetsurf/parsers/css/selectors"
)

func matcherFixture() (*dom.Document, *Matcher) {
	document := dom.NewDocument()
	body := document.CreateElement("body")
	main := document.CreateElement("div")
	main.SetAttribute("id", "main")
	main.SetAttribute("class", "container")
	main.SetAttribute("data-role", "main")
	first := document.CreateElement("p")
	first.SetAttribute("class", "first")
	second := document.CreateElement("p")
	second.SetAttribute("class", "second")
	main.AppendChild(first)
	main.AppendChild(second)
	list := document.CreateElement("ul")
	for _, class := range []string{"one", "two", "three"} {
		item := document.CreateElement("li")
		item.SetAttribute("class", class)
		list.AppendChild(item)
	}
	body.AppendChild(main)
	body.AppendChild(list)
	document.AppendChild(body)
	return document, NewDocumentMatcher(document)
}

func mustParseSelectors(t *testing.T, input string) *selectors.SelectorList {
	t.Helper()
	list, err := selectors.Parse(input)
	if err != nil {
		t.Fatalf("selectors.Parse(%q): %v", input, err)
	}
	return list
}

func TestMatcherElements(t *testing.T) {
	document, matcher := matcherFixture()
	tests := []struct {
		selector string
		count    int
	}{
		{"div > p.first", 1},
		{"div p", 2},
		{"li:nth-child(2n+1)", 2},
		{"[data-role=main]", 1},
		{".container", 1},
		{"#main", 1},
		{"p, li", 5},
		{"p:not(.first)", 1},
	}
	for _, test := range tests {
		if got := matcher.Elements(mustParseSelectors(t, test.selector)); len(got) != test.count {
			t.Errorf("Elements(%q) = %d, want %d", test.selector, len(got), test.count)
		}
	}
	main := document.GetElementById("main")
	if !matcher.Match(main, mustParseSelectors(t, "div.container")) {
		t.Error("Match failed")
	}
	if len(matcher.byID["main"]) != 1 || len(matcher.byClass["one"]) != 1 || len(matcher.byTag["p"]) != 2 || len(matcher.byAttribute["data-role"]) != 1 {
		t.Errorf("indexes: id=%d class=%d tag=%d attr=%d", len(matcher.byID["main"]), len(matcher.byClass["one"]), len(matcher.byTag["p"]), len(matcher.byAttribute["data-role"]))
	}
}

func TestMatcherRebuild(t *testing.T) {
	document, matcher := matcherFixture()
	body := document.Body()
	extra := document.CreateElement("p")
	extra.SetAttribute("class", "extra")
	body.AppendChild(extra)
	matcher.Rebuild()
	if len(matcher.byTag["p"]) != 3 {
		t.Errorf("p index after rebuild = %d, want 3", len(matcher.byTag["p"]))
	}
	if got := matcher.Elements(mustParseSelectors(t, "p.extra")); len(got) != 1 || got[0] != extra {
		t.Errorf("extra matches = %v", got)
	}
}

func TestMatcherHas(t *testing.T) {
	document, matcher := matcherFixture()
	main := document.GetElementById("main")
	main.AppendChild(document.CreateElement("span"))
	if got := matcher.Elements(mustParseSelectors(t, "div:has(span)")); len(got) != 1 || got[0] != main {
		t.Errorf(":has matches = %v", got)
	}
}

func BenchmarkMatcherElements(b *testing.B) {
	_, matcher := matcherFixture()
	list, _ := selectors.Parse("div > p, li:nth-child(2n+1), [data-role=main]")
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		_ = matcher.Elements(list)
	}
}
