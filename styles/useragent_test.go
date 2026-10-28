package styles

import (
	"testing"

	"gonetsurf/dom"
	"gonetsurf/parsers/css/cascade"
	"gonetsurf/parsers/css/parser"
)

func testElement(document *dom.Document, name string) *dom.Node {
	element := document.CreateElement(name)
	document.AppendChild(element)
	return element
}

func TestDefaultDisplayMatrix(t *testing.T) {
	document := dom.NewDocument()
	expected := map[string]string{
		"html":     "block",
		"body":     "block",
		"div":      "block",
		"p":        "block",
		"h1":       "block",
		"ul":       "block",
		"ol":       "block",
		"li":       "list-item",
		"table":    "table",
		"caption":  "table-caption",
		"colgroup": "table-column-group",
		"col":      "table-column",
		"thead":    "table-header-group",
		"tbody":    "table-row-group",
		"tfoot":    "table-footer-group",
		"tr":       "table-row",
		"td":       "table-cell",
		"th":       "table-cell",
		"head":     "none",
		"script":   "none",
		"style":    "none",
		"meta":     "none",
		"link":     "none",
		"title":    "none",
		"template": "none",
		"datalist": "none",
	}
	for name, want := range expected {
		element := testElement(document, name)
		got, ok := UserAgentDisplay(element)
		if !ok || got != want {
			t.Errorf("%s display = %q, %v; want %q", name, got, ok, want)
		}
	}
	hidden := document.CreateElement("div")
	hidden.SetAttribute("hidden", "")
	if got, ok := UserAgentDisplay(hidden); !ok || got != "none" {
		t.Errorf("hidden display = %q, %v; want none", got, ok)
	}
	dialog := document.CreateElement("dialog")
	dialog.SetAttribute("open", "")
	if _, ok := UserAgentDisplay(dialog); ok {
		t.Error("open dialog should not inherit the closed-dialog rule")
	}
	if _, ok := UserAgentDisplay(spanElement(document)); ok {
		t.Error("span has no UA display declaration")
	}
}

func spanElement(document *dom.Document) *dom.Node {
	return document.CreateElement("span")
}

func TestUserAgentLinkAndEmphasis(t *testing.T) {
	document := dom.NewDocument()
	link := document.CreateElement("a")
	link.SetAttribute("href", "/x")
	if value, ok := UserAgentValue(link, "text-decoration"); !ok || parser.SerializeValues(value) != "underline" {
		t.Errorf("link text-decoration = %q, %v", parser.SerializeValues(value), ok)
	}
	strong := document.CreateElement("strong")
	if value, ok := UserAgentValue(strong, "font-weight"); !ok || parser.SerializeValues(value) != "bold" {
		t.Errorf("strong font-weight = %q, %v", parser.SerializeValues(value), ok)
	}
}

func hintValue(declarations []cascade.Declaration, property string) string {
	for _, declaration := range declarations {
		if declaration.Property == property {
			return parser.SerializeValues(declaration.Value)
		}
	}
	return ""
}

func TestPresentationalHints(t *testing.T) {
	document := dom.NewDocument()
	image := document.CreateElement("img")
	image.SetAttribute("width", "10")
	image.SetAttribute("height", "20")
	image.SetAttribute("hspace", "5")
	image.SetAttribute("vspace", "3")
	image.SetAttribute("border", "1")
	image.SetAttribute("align", "left")
	hints := PresentationalHints(image)
	if got := hintValue(hints, "width"); got != "10px" {
		t.Errorf("width hint = %q", got)
	}
	if got := hintValue(hints, "height"); got != "20px" {
		t.Errorf("height hint = %q", got)
	}
	if got := hintValue(hints, "margin-left"); got != "5px" {
		t.Errorf("margin-left hint = %q", got)
	}
	if got := hintValue(hints, "margin-top"); got != "3px" {
		t.Errorf("margin-top hint = %q", got)
	}
	if got := hintValue(hints, "border"); got != "1px solid" {
		t.Errorf("border hint = %q", got)
	}
	if got := hintValue(hints, "float"); got != "left" {
		t.Errorf("float hint = %q", got)
	}

	table := document.CreateElement("table")
	table.SetAttribute("bgcolor", "red")
	table.SetAttribute("border", "2")
	table.SetAttribute("width", "50%")
	table.SetAttribute("align", "center")
	hints = PresentationalHints(table)
	if got := hintValue(hints, "background-color"); got != "red" {
		t.Errorf("bgcolor hint = %q", got)
	}
	if got := hintValue(hints, "border"); got != "2px solid" {
		t.Errorf("table border hint = %q", got)
	}
	if got := hintValue(hints, "width"); got != "50%" {
		t.Errorf("table width hint = %q", got)
	}
	if got := hintValue(hints, "margin-left"); got != "auto" {
		t.Errorf("table margin-left hint = %q", got)
	}

	font := document.CreateElement("font")
	font.SetAttribute("color", "blue")
	font.SetAttribute("face", "Arial")
	font.SetAttribute("size", "4")
	hints = PresentationalHints(font)
	if got := hintValue(hints, "color"); got != "blue" {
		t.Errorf("font color hint = %q", got)
	}
	if got := hintValue(hints, "font-family"); got != "Arial" {
		t.Errorf("font family hint = %q", got)
	}
	if got := hintValue(hints, "font-size"); got != "large" {
		t.Errorf("font size hint = %q", got)
	}

	cell := document.CreateElement("td")
	cell.SetAttribute("nowrap", "")
	if got := hintValue(PresentationalHints(cell), "white-space"); got != "nowrap" {
		t.Errorf("nowrap hint = %q", got)
	}

	svg := document.CreateElementNS(dom.NamespaceSVG, "rect")
	if len(PresentationalHints(svg)) != 0 {
		t.Error("presentational hints must not apply to SVG elements")
	}
}
