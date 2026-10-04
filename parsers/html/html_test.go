package html

import (
	"errors"
	"testing"

	"gonetsurf/dom"
	"gonetsurf/urls"
)

func TestParseDocument(t *testing.T) {
	source := `<!doctype html><html lang="en"><head><title>Hi</title></head>` +
		`<body><p id="a" class="x y">one <em>two</em></p><!-- c --></body></html>`
	document, err := ParseString(source, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if document.Quirks != dom.NoQuirks {
		t.Errorf("quirks = %v", document.Quirks)
	}
	if document.Title() != "Hi" {
		t.Errorf("title = %q", document.Title())
	}
	paragraph := document.GetElementById("a")
	if paragraph == nil {
		t.Fatal("missing paragraph")
	}
	if paragraph.TextContent() != "one two" {
		t.Errorf("text = %q", paragraph.TextContent())
	}
	if paragraph.ParentElement() == nil || paragraph.ParentElement().LocalName() != "body" {
		t.Error("parent mismatch")
	}
	if len(document.Body().Children()) != 2 {
		t.Errorf("body children = %d", len(document.Body().Children()))
	}
	if document.DocumentElement().AttributeValue("lang") != "en" {
		t.Error("attribute mismatch")
	}
}

func TestQuirks(t *testing.T) {
	tests := []struct {
		source string
		want   dom.QuirksMode
	}{
		{"<!doctype html><html><body></body></html>", dom.NoQuirks},
		{"<html><body></body></html>", dom.Quirks},
		{`<!doctype html public "-//W3C//DTD HTML 4.01 Transitional//EN"><html></html>`, dom.Quirks},
		{`<!doctype html public "-//W3C//DTD HTML 4.01 Transitional//EN" "http://www.w3.org/TR/html4/loose.dtd"><html></html>`, dom.LimitedQuirks},
		{`<!doctype html public "-//W3C//DTD XHTML 1.0 Transitional//EN" "x"><html></html>`, dom.LimitedQuirks},
	}
	for _, test := range tests {
		document, err := ParseString(test.source, ParseOptions{})
		if err != nil {
			t.Fatalf("ParseString(%q): %v", test.source, err)
		}
		if document.Quirks != test.want {
			t.Errorf("ParseString(%q).Quirks = %v, want %v", test.source, document.Quirks, test.want)
		}
	}
}

func TestNodeBudget(t *testing.T) {
	source := "<div><span><b><i>deep</i></b></span></div>"
	_, err := ParseString(source, ParseOptions{NodeBudget: 3})
	if !errors.Is(err, ErrNodeBudget) {
		t.Fatalf("expected ErrNodeBudget, got %v", err)
	}
}

func TestTemplateContents(t *testing.T) {
	document, err := ParseString("<template><p>inside</p></template>", ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	template := document.QuerySelector("template")
	if template == nil {
		t.Fatal("missing template")
	}
	if len(template.Children()) != 0 {
		t.Error("template children must move into contents")
	}
	contents := template.TemplateContents()
	if contents == nil || contents.TextContent() != "inside" {
		t.Errorf("template contents = %v", contents)
	}
}

func TestFragmentAndInnerHTML(t *testing.T) {
	document, _ := ParseString("<div id='host'></div>", ParseOptions{})
	host := document.GetElementById("host")
	if err := host.SetInnerHTML("<span>a</span><span>b</span>"); err != nil {
		t.Fatal(err)
	}
	if len(host.Children()) != 2 {
		t.Fatalf("children = %d", len(host.Children()))
	}
	serialized := host.InnerHTML()
	if serialized != "<span>a</span><span>b</span>" {
		t.Errorf("innerHTML = %q", serialized)
	}
}

func TestOriginFromURL(t *testing.T) {
	target, _ := urls.Parse("https://example.com/page")
	document, err := ParseString("<p>x</p>", ParseOptions{URL: target})
	if err != nil {
		t.Fatal(err)
	}
	if document.Origin.String() != "https://example.com" {
		t.Errorf("origin = %q", document.Origin.String())
	}
}

func TestForeignContent(t *testing.T) {
	document, err := ParseString(`<svg><circle r="1"/></svg>`, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	svg := document.QuerySelector("svg")
	if svg == nil || svg.Namespace != dom.NamespaceSVG {
		t.Fatalf("svg = %+v", svg)
	}
}
