package styles

import (
	"strconv"
	"strings"

	"gonetsurf/dom"
	"gonetsurf/parsers/css/cascade"
	"gonetsurf/parsers/css/parser"
)

// PresentationalHints maps legacy HTML presentational attributes to
// author-origin declarations. Hints sort before normal author rules.
func PresentationalHints(element *dom.Node) []cascade.Declaration {
	if element == nil || element.Type != dom.ElementNode || element.Namespace != dom.NamespaceHTML {
		return nil
	}
	declarations := []cascade.Declaration{}
	order := -1000
	add := func(property, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		declarations = append(declarations, cascade.Declaration{
			Property: property,
			Value:    parser.ParseComponentValueList(value),
			Origin:   cascade.OriginAuthor,
			Order:    order,
		})
		order++
	}
	localName := strings.ToLower(element.Data)
	if value, ok := element.GetAttribute("align"); ok {
		switch localName {
		case "div", "p", "h1", "h2", "h3", "h4", "h5", "h6", "td", "th", "tr", "thead", "tbody", "tfoot":
			add("text-align", value)
		case "table":
			if strings.EqualFold(value, "center") {
				add("margin-left", "auto")
				add("margin-right", "auto")
			} else if strings.EqualFold(value, "left") || strings.EqualFold(value, "right") {
				add("float", strings.ToLower(value))
			}
		case "img", "object", "embed", "iframe":
			if strings.EqualFold(value, "left") || strings.EqualFold(value, "right") {
				add("float", strings.ToLower(value))
			} else if strings.EqualFold(value, "center") {
				add("display", "block")
				add("margin-left", "auto")
				add("margin-right", "auto")
			}
		}
	}
	if value, ok := element.GetAttribute("valign"); ok {
		add("vertical-align", value)
	}
	if value, ok := element.GetAttribute("bgcolor"); ok {
		add("background-color", value)
	}
	if value, ok := element.GetAttribute("color"); ok && localName == "font" {
		add("color", value)
	}
	if value, ok := element.GetAttribute("face"); ok && localName == "font" {
		add("font-family", value)
	}
	if value, ok := element.GetAttribute("size"); ok && localName == "font" {
		if size, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
			add("font-size", fontSizeForHTMLSize(size))
		}
	}
	if value, ok := element.GetAttribute("width"); ok {
		add("width", htmlDimension(value))
	}
	if value, ok := element.GetAttribute("height"); ok {
		add("height", htmlDimension(value))
	}
	if value, ok := element.GetAttribute("border"); ok {
		switch localName {
		case "table", "img", "object":
			border := htmlDimension(value)
			if isNumericValue(value) {
				border += " solid"
			}
			add("border", border)
		}
	}
	if value, ok := element.GetAttribute("hspace"); ok {
		margin := htmlDimension(value)
		add("margin-left", margin)
		add("margin-right", margin)
	}
	if value, ok := element.GetAttribute("vspace"); ok {
		margin := htmlDimension(value)
		add("margin-top", margin)
		add("margin-bottom", margin)
	}
	if element.HasAttribute("nowrap") && (localName == "td" || localName == "th") {
		add("white-space", "nowrap")
	}
	if value, ok := element.GetAttribute("cellspacing"); ok && localName == "table" {
		add("border-spacing", htmlDimension(value))
	}
	return declarations
}

func htmlDimension(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if value == "0" {
		return "0"
	}
	if strings.HasSuffix(value, "%") {
		return value
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return value
		}
	}
	return value + "px"
}

func isNumericValue(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func fontSizeForHTMLSize(size int) string {
	switch {
	case size <= 1:
		return "x-small"
	case size == 2:
		return "small"
	case size == 3:
		return "medium"
	case size == 4:
		return "large"
	case size == 5:
		return "x-large"
	case size == 6:
		return "xx-large"
	default:
		return "xxx-large"
	}
}
