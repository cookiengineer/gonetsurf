package styles

import (
	_ "embed"
	"strings"
	"sync"

	"gonetsurf/dom"
	"gonetsurf/parsers/css/cascade"
	"gonetsurf/parsers/css/cssom"
	"gonetsurf/parsers/css/parser"
)

//go:embed useragent.css
var userAgentCSS string

var (
	userAgentOnce  sync.Once
	userAgentSheet *cssom.Stylesheet
)

// UserAgentStylesheet returns the parsed built-in UA stylesheet.
func UserAgentStylesheet() *cssom.Stylesheet {
	userAgentOnce.Do(func() {
		userAgentSheet = cssom.Parse(userAgentCSS)
	})
	return userAgentSheet
}

// UserAgentValue returns the UA-origin winning value for a property on an
// element, if the built-in sheet declares one.
func UserAgentValue(element *dom.Node, property string) ([]parser.ComponentValue, bool) {
	sheet := UserAgentStylesheet()
	if sheet == nil || element == nil || element.Type != dom.ElementNode {
		return nil, false
	}
	declarations := []cascade.Declaration{}
	order := 0
	for _, rule := range sheet.CSSRules() {
		styleRule, ok := rule.(*cssom.StyleRule)
		if !ok || styleRule.Selectors == nil {
			continue
		}
		if !element.MatchesSelectorList(styleRule.Selectors) {
			order += len(styleRule.Declarations)
			continue
		}
		for _, declaration := range styleRule.Declarations {
			if !strings.EqualFold(declaration.Name, property) {
				order++
				continue
			}
			declarations = append(declarations, cascade.Declaration{
				Property:    declaration.Name,
				Value:       declaration.Value,
				Important:   declaration.Important,
				Origin:      cascade.OriginUserAgent,
				Specificity: styleRule.Selectors.MaxSpecificity(),
				Order:       order,
			})
			order++
		}
	}
	winner, ok := cascade.Cascade(declarations).Get(property)
	if !ok {
		return nil, false
	}
	return winner.Value, true
}

// UserAgentDisplay returns the UA-origin display value for an element.
func UserAgentDisplay(element *dom.Node) (string, bool) {
	value, ok := UserAgentValue(element, "display")
	if !ok {
		return "", false
	}
	return strings.ToLower(strings.TrimSpace(parser.SerializeValues(value))), true
}
