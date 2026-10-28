package adapters

import (
	"strings"

	"gonetsurf/dom"
	"gonetsurf/parsers/css/conditions"
	"gonetsurf/parsers/css/cssom"
	"gonetsurf/parsers/css/parser"
	"gonetsurf/parsers/css/tokenizer"
	"gonetsurf/styles"
	"gonetsurf/urls"
)

const (
	defaultImportDepth = 16
	defaultSheetBytes  = 8 << 20
)

// StylesheetLoader builds a style engine from inline and linked stylesheets.
// Fetch is called for external sheets and nested @import rules.
type StylesheetLoader struct {
	Fetch    func(*urls.URL) ([]byte, error)
	MaxDepth int
	MaxBytes int
}

// Engine returns an engine containing all applicable stylesheets in document
// order. Failed external loads are skipped.
func (loader StylesheetLoader) Engine(document *dom.Document, base *urls.URL) *styles.Engine {
	engine := styles.NewEngine(document)
	if document == nil {
		return engine
	}
	maximumDepth := loader.MaxDepth
	if maximumDepth <= 0 {
		maximumDepth = defaultImportDepth
	}
	visited := make(map[string]bool)
	document.Node.Descendants(func(node *dom.Node) bool {
		if node.Type != dom.ElementNode || node.Namespace != dom.NamespaceHTML {
			return true
		}
		switch strings.ToLower(node.Data) {
		case "style":
			loader.addCSS(engine, node.TextContent(), base, 0, maximumDepth, visited)
		case "link":
			if _, disabled := node.GetAttribute("disabled"); disabled {
				return true
			}
			rel, _ := node.GetAttribute("rel")
			if !hasToken(rel, "stylesheet") {
				return true
			}
			href, ok := node.GetAttribute("href")
			if !ok || strings.TrimSpace(href) == "" {
				return true
			}
			target, err := urls.ParseReference(base, href)
			if err != nil {
				return true
			}
			loader.addURL(engine, target, 0, maximumDepth, visited)
		}
		return true
	})
	return engine
}

func (loader StylesheetLoader) addURL(engine *styles.Engine, target *urls.URL, depth, maximumDepth int, visited map[string]bool) {
	if engine == nil || target == nil || loader.Fetch == nil || depth > maximumDepth {
		return
	}
	key := target.String()
	if visited[key] {
		return
	}
	visited[key] = true
	body, err := loader.Fetch(target)
	if err != nil {
		return
	}
	maximumBytes := loader.MaxBytes
	if maximumBytes <= 0 {
		maximumBytes = defaultSheetBytes
	}
	if len(body) > maximumBytes {
		return
	}
	loader.addCSS(engine, string(body), target, depth, maximumDepth, visited)
}

func (loader StylesheetLoader) addCSS(engine *styles.Engine, source string, base *urls.URL, depth, maximumDepth int, visited map[string]bool) {
	if engine == nil || strings.TrimSpace(source) == "" {
		return
	}
	stylesheet := cssom.Parse(source)
	if stylesheet == nil {
		return
	}
	absolutizeRules(stylesheet.CSSRules(), base)
	for _, rule := range stylesheet.CSSRules() {
		atRule, ok := rule.(*cssom.AtRule)
		if !ok || !strings.EqualFold(atRule.Name, "import") {
			continue
		}
		importRule, err := conditions.ParseImport(parser.SerializeValues(atRule.Prelude))
		if err != nil || importRule.URL == "" {
			continue
		}
		target, err := urls.ParseReference(base, importRule.URL)
		if err != nil {
			continue
		}
		loader.addURL(engine, target, depth+1, maximumDepth, visited)
	}
	engine.AddStylesheet(stylesheet)
}

func absolutizeRules(rules []cssom.Rule, base *urls.URL) {
	for _, rule := range rules {
		switch typed := rule.(type) {
		case *cssom.StyleRule:
			absolutizeDeclarations(typed.Declarations, base)
			absolutizeRules(typed.Nested, base)
		case *cssom.AtRule:
			absolutizeDeclarations(typed.Declarations, base)
			absolutizeRules(typed.Rules, base)
			if typed.Block != nil {
				absolutizeValues(typed.Block, base)
			}
		case *cssom.FontFaceRule:
			absolutizeDeclarations(typed.Declarations, base)
			for index := range typed.Sources {
				if typed.Sources[index].URL == "" {
					continue
				}
				if target, err := urls.ParseReference(base, typed.Sources[index].URL); err == nil {
					typed.Sources[index].URL = target.String()
				}
			}
		}
	}
}

func absolutizeDeclarations(declarations []*cssom.Declaration, base *urls.URL) {
	for _, declaration := range declarations {
		if declaration == nil {
			continue
		}
		absolutizeValues(declaration.Value, base)
	}
}

func absolutizeValues(values []parser.ComponentValue, base *urls.URL) {
	if base == nil {
		return
	}
	for index, value := range values {
		switch typed := value.(type) {
		case parser.Token:
			if typed.Type != tokenizer.TokenURL || strings.TrimSpace(typed.Value) == "" {
				continue
			}
			target, err := urls.ParseReference(base, typed.Value)
			if err != nil {
				continue
			}
			typed.Value = target.String()
			typed.Representation = "url(" + quoteCSSString(typed.Value) + ")"
			values[index] = typed
		case *parser.Function:
			if strings.EqualFold(typed.Name, "url") {
				address := strings.TrimSpace(parser.SerializeValues(typed.Value))
				address = strings.Trim(address, "\"'")
				target, err := urls.ParseReference(base, address)
				if err != nil {
					continue
				}
				typed.Value = []parser.ComponentValue{parser.Token{Token: tokenizer.Token{
					Type:           tokenizer.TokenString,
					Value:          target.String(),
					Representation: quoteCSSString(target.String()),
				}}}
				continue
			}
			absolutizeValues(typed.Value, base)
		case *parser.Block:
			absolutizeValues(typed.Value, base)
		}
	}
}

func quoteCSSString(value string) string {
	var builder strings.Builder
	builder.WriteByte('"')
	for _, character := range value {
		switch character {
		case '\\', '"':
			builder.WriteByte('\\')
			builder.WriteRune(character)
		case '\n':
			builder.WriteString("\\a ")
		case '\r':
			builder.WriteString("\\d ")
		case '\f':
			builder.WriteString("\\c ")
		default:
			builder.WriteRune(character)
		}
	}
	builder.WriteByte('"')
	return builder.String()
}

func hasToken(value, token string) bool {
	for _, field := range strings.Fields(value) {
		if strings.EqualFold(field, token) {
			return true
		}
	}
	return false
}
