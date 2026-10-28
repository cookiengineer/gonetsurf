package styles

import (
	"strings"

	"gonetsurf/dom"
	"gonetsurf/parsers/css/cascade"
	"gonetsurf/parsers/css/conditions"
	"gonetsurf/parsers/css/cssom"
	"gonetsurf/parsers/css/mediaqueries"
	"gonetsurf/parsers/css/nesting"
	"gonetsurf/parsers/css/parser"
	"gonetsurf/parsers/css/properties"
	"gonetsurf/parsers/css/selectors"
)

// Style is a computed style for one element.
type Style struct {
	element *dom.Node
	values  map[string][]parser.ComponentValue
}

// Element returns the style's element.
func (style *Style) Element() *dom.Node {
	return style.element
}

// Get returns the computed value for a property, or nil.
func (style *Style) Get(property string) []parser.ComponentValue {
	if style == nil {
		return nil
	}
	return style.values[property]
}

// GetString returns the computed value serialized as CSS text.
func (style *Style) GetString(property string) string {
	return strings.TrimSpace(parser.SerializeValues(style.Get(property)))
}

// Has reports whether the property was computed.
func (style *Style) Has(property string) bool {
	if style == nil {
		return false
	}
	_, ok := style.values[property]
	return ok
}

// Names returns the computed property names.
func (style *Style) Names() []string {
	names := make([]string, 0, len(style.values))
	for name := range style.values {
		names = append(names, name)
	}
	return names
}

// Engine resolves styles for a DOM tree.
type Engine struct {
	document     *dom.Document
	matcher      *Matcher
	stylesheets  []*cssom.Stylesheet
	environment  mediaqueries.Environment
	rootFontSize float64
	cache        map[*dom.Node]*Style
	cacheVersion uint64
	hasHas       bool
}

// NewEngine creates a style engine for a document. The returned matcher is
// rebuilt when the document is mutated.
func NewEngine(document *dom.Document) *Engine {
	return &Engine{
		document:     document,
		matcher:      NewDocumentMatcher(document),
		environment:  mediaqueries.DefaultEnvironment(),
		rootFontSize: 16,
		cache:        map[*dom.Node]*Style{},
	}
}

// Matcher returns the engine's selector matcher.
func (engine *Engine) Matcher() *Matcher {
	return engine.matcher
}

// AddStylesheet adds an author stylesheet.
func (engine *Engine) AddStylesheet(stylesheet *cssom.Stylesheet) {
	if stylesheet == nil {
		return
	}
	engine.stylesheets = append(engine.stylesheets, stylesheet)
	engine.hasHas = engine.hasHas || stylesheetUsesHas(stylesheet)
	engine.cache = nil
	engine.cacheVersion = 0
}

// SetMediaEnvironment replaces the media-query environment.
func (engine *Engine) SetMediaEnvironment(environment mediaqueries.Environment) {
	engine.environment = environment
	engine.cache = nil
	engine.cacheVersion = 0
}

// Rebuild refreshes the selector indexes after document mutations.
func (engine *Engine) Rebuild() {
	engine.matcher.Rebuild()
	engine.cache = nil
	engine.cacheVersion = 0
}

// Compute computes the style for one element. Parent may be nil for a root.
func (engine *Engine) Compute(element *dom.Node, parent *Style) *Style {
	if element == nil || element.Type != dom.ElementNode {
		return nil
	}
	declarations := []cascade.Declaration{}
	order := 0
	for _, rule := range UserAgentStylesheet().CSSRules() {
		engine.collectRule(rule, cascade.OriginUserAgent, "", element, &declarations, &order)
	}
	for _, stylesheet := range engine.stylesheets {
		for _, rule := range stylesheet.CSSRules() {
			engine.collectRule(rule, cascade.OriginAuthor, "", element, &declarations, &order)
		}
	}
	declarations = append(declarations, PresentationalHints(element)...)
	if inline, ok := element.GetAttribute("style"); ok {
		for _, node := range parser.ParseListOfDeclarations(inline) {
			declaration, ok := node.(*parser.Declaration)
			if !ok {
				continue
			}
			declarations = append(declarations, cascade.Declaration{
				Property:    declaration.Name,
				Value:       declaration.Value,
				Important:   declaration.Important,
				Origin:      cascade.OriginAuthor,
				Specificity: selectors.Specificity{A: 1},
				Order:       order,
			})
			order++
		}
	}
	result := cascade.Cascade(declarations)
	propertyNames := map[string]bool{}
	for _, name := range []string{"font-family", "font-size", "font-style", "font-weight", "line-height", "color"} {
		propertyNames[name] = true
	}
	if _, hasAll := result.Get("all"); hasAll {
		for _, name := range properties.Names() {
			propertyNames[name] = true
		}
	}
	for _, declaration := range declarations {
		if !strings.EqualFold(declaration.Property, "all") {
			propertyNames[declaration.Property] = true
		}
	}
	var parentValues cascade.InheritedValues
	if parent != nil {
		parentValues = cascade.InheritedValues(parent.values)
		for name := range parent.values {
			propertyNames[name] = true
		}
	}
	values := map[string][]parser.ComponentValue{}
	for name := range propertyNames {
		values[name] = result.ResolveWithCustom(name, parentValues, initialValue(name))
	}
	context := computedContext{
		RootFontSize:   engine.rootFontSize,
		ViewportWidth:  engine.environment.Width,
		ViewportHeight: engine.environment.Height,
	}
	values = computeStyleValues(values, parent, context)
	return &Style{element: element, values: values}
}

// ComputeDocumentTree computes styles for a document tree.
func (engine *Engine) ComputeDocumentTree(document *dom.Document) map[*dom.Node]*Style {
	if document == nil {
		return map[*dom.Node]*Style{}
	}
	return engine.Restyles(document)
}

// Restyles incrementally recomputes styles after document mutations. It
// rebuilds selector indexes, then recomputes only the affected parent
// subtrees. Stylesheets using :has() fall back to a full restyle because
// descendant changes can invalidate ancestors.
func (engine *Engine) Restyles(document *dom.Document) map[*dom.Node]*Style {
	if document == nil {
		return map[*dom.Node]*Style{}
	}
	if engine.cache == nil {
		engine.cache = map[*dom.Node]*Style{}
	}
	if engine.cacheVersion == document.MutationVersion() && len(engine.cache) > 0 {
		return engine.cache
	}
	if engine.hasHas || len(engine.cache) == 0 {
		return engine.ComputeTree(&document.Node)
	}
	engine.matcher.Rebuild()
	for node := range engine.cache {
		if !node.IsConnected() {
			delete(engine.cache, node)
		}
	}
	dirty := document.DirtyNodes()
	roots := map[*dom.Node]bool{}
	full := false
	for _, node := range dirty {
		switch node.Type {
		case dom.DocumentNode:
			full = true
		case dom.ElementNode:
			if !node.IsConnected() {
				continue
			}
			if parent := node.ParentElement(); parent != nil {
				roots[parent] = true
			} else {
				roots[node] = true
			}
		default:
			if parent := node.Parent(); parent != nil && parent.Type == dom.ElementNode && parent.IsConnected() {
				roots[parent] = true
			}
		}
	}
	if full || len(roots) == 0 {
		return engine.ComputeTree(&document.Node)
	}
	for root := range roots {
		var parentStyle *Style
		if parent := root.ParentElement(); parent != nil {
			parentStyle = engine.cache[parent]
		}
		engine.computeSubtree(root, parentStyle, engine.cache)
	}
	document.ClearDirty()
	engine.cacheVersion = document.MutationVersion()
	return engine.cache
}

func (engine *Engine) computeSubtree(node *dom.Node, parent *Style, cache map[*dom.Node]*Style) {
	if node.Type == dom.ElementNode {
		style := engine.Compute(node, parent)
		cache[node] = style
		parent = style
		if owner := node.OwnerDocument(); owner != nil && owner.DocumentElement() == node {
			engine.rootFontSize = styleFontSize(style)
		}
	}
	for _, child := range node.Children() {
		engine.computeSubtree(child, parent, cache)
	}
}

// ComputeTree computes styles for root and its element descendants.
func (engine *Engine) ComputeTree(root *dom.Node) map[*dom.Node]*Style {
	result := map[*dom.Node]*Style{}
	if root == nil {
		return result
	}
	var visit func(*dom.Node, *Style)
	visit = func(node *dom.Node, parent *Style) {
		if node.Type == dom.ElementNode {
			style := engine.Compute(node, parent)
			result[node] = style
			parent = style
			if owner := node.OwnerDocument(); owner != nil && owner.DocumentElement() == node {
				engine.rootFontSize = styleFontSize(style)
			}
		}
		for _, child := range node.Children() {
			visit(child, parent)
		}
	}
	visit(root, nil)
	if engine.document != nil && root == &engine.document.Node {
		engine.cache = result
		engine.cacheVersion = engine.document.MutationVersion()
		engine.document.ClearDirty()
	}
	return result
}

func (engine *Engine) collectRule(rule cssom.Rule, origin cascade.Origin, parentSelector string, element *dom.Node, declarations *[]cascade.Declaration, order *int) {
	switch typed := rule.(type) {
	case *cssom.StyleRule:
		selectorText := typed.SelectorText()
		if parentSelector != "" {
			selectorText = nesting.ResolveSelector(selectorText, parentSelector)
		}
		if list, err := selectors.Parse(selectorText); err == nil && element.MatchesSelectorList(list) {
			specificity := list.MaxSpecificity()
			for _, declaration := range typed.Declarations {
				*declarations = append(*declarations, cascade.Declaration{
					Property:    declaration.Name,
					Value:       declaration.Value,
					Important:   declaration.Important,
					Origin:      origin,
					Specificity: specificity,
					Order:       *order,
				})
				*order++
			}
		}
		for _, nested := range typed.CSSRules() {
			engine.collectRule(nested, origin, selectorText, element, declarations, order)
		}
	case *cssom.AtRule:
		if !engine.atRuleApplies(typed) {
			return
		}
		for _, nested := range typed.CSSRules() {
			engine.collectRule(nested, origin, parentSelector, element, declarations, order)
		}
	}
}

func (engine *Engine) atRuleApplies(rule *cssom.AtRule) bool {
	switch strings.ToLower(rule.Name) {
	case "media":
		return rule.Media().Matches(engine.environment)
	case "supports":
		condition, err := conditions.ParseSupports(strings.TrimSpace(parser.SerializeValues(rule.Prelude)))
		if err != nil {
			return false
		}
		return condition.Evaluate(supportChecker{})
	case "layer", "container", "scope", "document", "starting-style":
		return true
	default:
		return false
	}
}

type supportChecker struct{}

func (supportChecker) SupportsDeclaration(property, value string) bool {
	valid, _ := properties.Validate(property, value)
	return valid
}

func (supportChecker) SupportsSelector(selector string) bool {
	_, err := selectors.Parse(selector)
	return err == nil
}

func stylesheetUsesHas(stylesheet *cssom.Stylesheet) bool {
	if stylesheet == nil {
		return false
	}
	return rulesUseHas(stylesheet.CSSRules())
}

func rulesUseHas(rules []cssom.Rule) bool {
	for _, rule := range rules {
		switch typed := rule.(type) {
		case *cssom.StyleRule:
			if strings.Contains(strings.ToLower(typed.SelectorText()), ":has(") {
				return true
			}
			if rulesUseHas(typed.CSSRules()) {
				return true
			}
		case *cssom.AtRule:
			if rulesUseHas(typed.CSSRules()) {
				return true
			}
		}
	}
	return false
}

func styleFontSize(style *Style) float64 {
	if size, ok := parseComputedPixels(style.GetString("font-size")); ok && size > 0 {
		return size
	}
	return 16
}

func initialValue(property string) []parser.ComponentValue {
	if cascade.IsCustom(property) {
		return nil
	}
	registered, ok := properties.Lookup(property)
	if !ok {
		return nil
	}
	return parser.ParseComponentValueList(registered.Initial)
}
