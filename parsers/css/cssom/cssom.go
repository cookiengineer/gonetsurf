// Package cssom implements the CSS Object Model: stylesheets, rules,
// style declarations, and media lists with parsing and serialization.
//
// See specifications/css/cssom-1.html and specifications/css/syntax-3.html
// §10.
package cssom

import (
	"errors"
	"fmt"
	"strings"

	"gonetsurf/parsers/css/mediaqueries"
	"gonetsurf/parsers/css/parser"
	"gonetsurf/parsers/css/selectors"
	"gonetsurf/parsers/css/tokenizer"
)

// Stylesheet is a parsed list of CSS rules.
type Stylesheet struct {
	Rules []Rule
}

// CSSStyleSheet is the CSSOM name for a parsed stylesheet.
type CSSStyleSheet = Stylesheet

// ErrIndexOutOfRange is returned by InsertRule and DeleteRule for invalid
// indices.
var ErrIndexOutOfRange = errors.New("cssom: index out of range")

// ErrNotFound is returned when a media query or property is not present.
var ErrNotFound = errors.New("cssom: not found")

// Rule is a stylesheet rule: a style rule or an at-rule.
type Rule interface {
	rule()
	Serialize() string
}

// StyleRule is a qualified rule with a selector prelude and a declaration
// block.
type StyleRule struct {
	Selectors    *selectors.SelectorList
	Prelude      []parser.ComponentValue
	Declarations []*Declaration
	// Nested holds nested rules (CSS Nesting).
	Nested []Rule
}

func (*StyleRule) rule() {}

// AtRule is an at-rule such as @media or @import.
type AtRule struct {
	Name         string
	Prelude      []parser.ComponentValue
	Block        []parser.ComponentValue
	Declarations []*Declaration
	Rules        []Rule
	// MediaQueries is the live media list for @media rules.
	MediaQueries *MediaList
}

func (*AtRule) rule() {}

// Declaration is a property declaration.
type Declaration struct {
	Name      string
	Value     []parser.ComponentValue
	Important bool
}

// Parse parses a stylesheet, ignoring CDO and CDC tokens between rules.
func Parse(input string) *Stylesheet {
	stylesheet := &Stylesheet{Rules: []Rule{}}
	for _, node := range parser.ParseStylesheet(input) {
		if rule := convertRule(node); rule != nil {
			stylesheet.Rules = append(stylesheet.Rules, rule)
		}
	}
	return stylesheet
}

// ParseRule parses a single rule.
func ParseRule(input string) (Rule, error) {
	node, err := parser.ParseRule(input)
	if err != nil {
		return nil, err
	}
	return convertRule(node), nil
}

func convertRule(node parser.Node) Rule {
	switch typed := node.(type) {
	case *parser.QualifiedRule:
		return convertQualifiedRule(typed)
	case *parser.AtRule:
		if strings.EqualFold(typed.Name, "font-face") {
			return convertFontFaceRule(typed)
		}
		return convertAtRule(typed)
	default:
		return nil
	}
}

func convertQualifiedRule(typed *parser.QualifiedRule) *StyleRule {
	rule := &StyleRule{Prelude: typed.Prelude}
	if list, err := selectors.ParseValues(typed.Prelude); err == nil {
		rule.Selectors = list
	}
	if typed.Block != nil {
		rule.Declarations, rule.Nested = convertBlockContents(typed.Block.Value)
	}
	return rule
}

func convertAtRule(typed *parser.AtRule) *AtRule {
	rule := &AtRule{Name: typed.Name, Prelude: typed.Prelude}
	if strings.EqualFold(typed.Name, "media") {
		rule.MediaQueries, _ = ParseMediaList(strings.TrimSpace(parser.SerializeValues(typed.Prelude)))
	}
	if typed.Block != nil {
		rule.Block = typed.Block.Value
		declarations, rules := convertBlockContents(typed.Block.Value)
		if isGroupingAtRule(typed.Name) {
			rule.Rules = rules
		} else {
			rule.Declarations = declarations
		}
	}
	return rule
}

func convertBlockContents(values []parser.ComponentValue) ([]*Declaration, []Rule) {
	var declarations []*Declaration
	var rules []Rule
	for _, node := range parser.ParseBlockContentsValues(values) {
		switch typed := node.(type) {
		case *parser.Declaration:
			declarations = append(declarations, &Declaration{
				Name:      typed.Name,
				Value:     typed.Value,
				Important: typed.Important,
			})
		case *parser.QualifiedRule:
			rules = append(rules, convertQualifiedRule(typed))
		case *parser.AtRule:
			if rule := convertRule(typed); rule != nil {
				rules = append(rules, rule)
			}
		}
	}
	return declarations, rules
}

func isGroupingAtRule(name string) bool {
	switch strings.ToLower(name) {
	case "media", "supports", "layer", "container", "scope", "document", "starting-style", "keyframes", "-webkit-keyframes":
		return true
	default:
		return false
	}
}

func convertDeclarations(values []parser.ComponentValue) []*Declaration {
	declarations, _ := convertBlockContents(values)
	return declarations
}

// CSSRules returns the stylesheet's rule list.
func (stylesheet *Stylesheet) CSSRules() []Rule {
	return stylesheet.Rules
}

// InsertRule inserts a parsed rule at index.
func (stylesheet *Stylesheet) InsertRule(source string, index int) error {
	return insertRule(&stylesheet.Rules, source, index)
}

// DeleteRule removes the rule at index.
func (stylesheet *Stylesheet) DeleteRule(index int) error {
	return deleteRule(&stylesheet.Rules, index)
}

// Serialize returns the stylesheet as CSS text.
func (stylesheet *Stylesheet) Serialize() string {
	var builder strings.Builder
	for _, rule := range stylesheet.Rules {
		builder.WriteString(rule.Serialize())
	}
	return builder.String()
}

// CSSRules returns the nested rule list.
func (rule *StyleRule) CSSRules() []Rule {
	return rule.Nested
}

// InsertRule inserts a nested rule at index.
func (rule *StyleRule) InsertRule(source string, index int) error {
	return insertRule(&rule.Nested, source, index)
}

// DeleteRule removes the nested rule at index.
func (rule *StyleRule) DeleteRule(index int) error {
	return deleteRule(&rule.Nested, index)
}

// SelectorText returns the rule's selector prelude.
func (rule *StyleRule) SelectorText() string {
	return strings.TrimSpace(parser.SerializeValues(rule.Prelude))
}

// SetSelectorText replaces the selector prelude after validating it.
func (rule *StyleRule) SetSelectorText(text string) error {
	list, err := selectors.Parse(text)
	if err != nil {
		return err
	}
	rule.Selectors = list
	rule.Prelude = parser.ParseComponentValueList(text)
	return nil
}

// Style returns a mutable view of the rule's declarations.
func (rule *StyleRule) Style() *CSSStyleDeclaration {
	return &CSSStyleDeclaration{declarations: &rule.Declarations}
}

// Serialize returns the style rule as CSS text.
func (rule *StyleRule) Serialize() string {
	var builder strings.Builder
	builder.WriteString(rule.SelectorText())
	builder.WriteString(" {")
	for _, declaration := range rule.Declarations {
		builder.WriteString(" ")
		builder.WriteString(declaration.Serialize())
	}
	for _, nested := range rule.Nested {
		builder.WriteString(" ")
		builder.WriteString(strings.TrimSpace(nested.Serialize()))
	}
	builder.WriteString(" }")
	return builder.String()
}

// CSSRules returns the nested rule list.
func (rule *AtRule) CSSRules() []Rule {
	return rule.Rules
}

// InsertRule inserts a nested rule at index.
func (rule *AtRule) InsertRule(source string, index int) error {
	return insertRule(&rule.Rules, source, index)
}

// DeleteRule removes the nested rule at index.
func (rule *AtRule) DeleteRule(index int) error {
	return deleteRule(&rule.Rules, index)
}

// Style returns a mutable view of the at-rule's declarations.
func (rule *AtRule) Style() *CSSStyleDeclaration {
	return &CSSStyleDeclaration{declarations: &rule.Declarations}
}

// Media returns the parsed media list for an @media rule.
func (rule *AtRule) Media() *MediaList {
	if !strings.EqualFold(rule.Name, "media") {
		return &MediaList{}
	}
	if rule.MediaQueries == nil {
		rule.MediaQueries, _ = ParseMediaList(strings.TrimSpace(parser.SerializeValues(rule.Prelude)))
	}
	return rule.MediaQueries
}

// Serialize returns the at-rule as CSS text.
func (rule *AtRule) Serialize() string {
	var builder strings.Builder
	builder.WriteString("@")
	builder.WriteString(rule.Name)
	prelude := strings.TrimSpace(parser.SerializeValues(rule.Prelude))
	if rule.MediaQueries != nil {
		prelude = rule.MediaQueries.MediaText()
	}
	if prelude != "" {
		builder.WriteString(" ")
		builder.WriteString(prelude)
	}
	if rule.Block == nil && len(rule.Rules) == 0 && len(rule.Declarations) == 0 {
		builder.WriteString(";")
		return builder.String()
	}
	var body strings.Builder
	for _, declaration := range rule.Declarations {
		body.WriteString(" ")
		body.WriteString(declaration.Serialize())
	}
	for _, nested := range rule.Rules {
		body.WriteString(" ")
		body.WriteString(strings.TrimSpace(nested.Serialize()))
	}
	content := strings.TrimSpace(body.String())
	if content == "" && rule.Block != nil {
		content = strings.TrimSpace(parser.SerializeValues(rule.Block))
	}
	builder.WriteString(" {")
	if content != "" {
		builder.WriteString(" ")
		builder.WriteString(content)
	}
	builder.WriteString(" }")
	return builder.String()
}

// Serialize returns the declaration as CSS text, including the trailing
// semicolon.
func (declaration *Declaration) Serialize() string {
	value := strings.TrimSpace(parser.SerializeValues(declaration.Value))
	var builder strings.Builder
	builder.WriteString(declaration.Name)
	builder.WriteString(": ")
	builder.WriteString(value)
	if declaration.Important {
		builder.WriteString(" !important")
	}
	builder.WriteString(";")
	return builder.String()
}

func insertRule(rules *[]Rule, source string, index int) error {
	if index < 0 || index > len(*rules) {
		return ErrIndexOutOfRange
	}
	rule, err := ParseRule(source)
	if err != nil {
		return err
	}
	*rules = append(*rules, nil)
	copy((*rules)[index+1:], (*rules)[index:])
	(*rules)[index] = rule
	return nil
}

func deleteRule(rules *[]Rule, index int) error {
	if index < 0 || index >= len(*rules) {
		return ErrIndexOutOfRange
	}
	*rules = append((*rules)[:index], (*rules)[index+1:]...)
	return nil
}

// CSSStyleDeclaration is a mutable list of declarations.
type CSSStyleDeclaration struct {
	declarations *[]*Declaration
}

// Length returns the number of declarations.
func (style *CSSStyleDeclaration) Length() int {
	return len(*style.declarations)
}

// Item returns the property name at index, or an empty string.
func (style *CSSStyleDeclaration) Item(index int) string {
	if index < 0 || index >= len(*style.declarations) {
		return ""
	}
	return (*style.declarations)[index].Name
}

// GetPropertyValue returns the serialized value for a property.
func (style *CSSStyleDeclaration) GetPropertyValue(name string) string {
	for _, declaration := range *style.declarations {
		if strings.EqualFold(declaration.Name, name) {
			return strings.TrimSpace(parser.SerializeValues(declaration.Value))
		}
	}
	return ""
}

// GetPropertyPriority returns "important" for important declarations.
func (style *CSSStyleDeclaration) GetPropertyPriority(name string) string {
	for _, declaration := range *style.declarations {
		if strings.EqualFold(declaration.Name, name) && declaration.Important {
			return "important"
		}
	}
	return ""
}

// SetProperty sets or replaces a declaration.
func (style *CSSStyleDeclaration) SetProperty(name, value, priority string) error {
	priority = strings.ToLower(strings.TrimSpace(priority))
	if priority != "" && priority != "important" {
		return fmt.Errorf("cssom: invalid priority %q", priority)
	}
	declaration := &Declaration{
		Name:      name,
		Value:     parser.ParseComponentValueList(value),
		Important: priority == "important",
	}
	for index, existing := range *style.declarations {
		if strings.EqualFold(existing.Name, name) {
			(*style.declarations)[index] = declaration
			return nil
		}
	}
	*style.declarations = append(*style.declarations, declaration)
	return nil
}

// RemoveProperty removes a declaration and returns its old value.
func (style *CSSStyleDeclaration) RemoveProperty(name string) string {
	for index, declaration := range *style.declarations {
		if strings.EqualFold(declaration.Name, name) {
			value := strings.TrimSpace(parser.SerializeValues(declaration.Value))
			*style.declarations = append((*style.declarations)[:index], (*style.declarations)[index+1:]...)
			return value
		}
	}
	return ""
}

// CSSText returns the declaration block as CSS text.
func (style *CSSStyleDeclaration) CSSText() string {
	var builder strings.Builder
	for index, declaration := range *style.declarations {
		if index > 0 {
			builder.WriteString(" ")
		}
		builder.WriteString(declaration.Serialize())
	}
	return builder.String()
}

// SetCSSText replaces the declaration block.
func (style *CSSStyleDeclaration) SetCSSText(text string) {
	*style.declarations = convertDeclarations(parser.ParseComponentValueList(text))
}

// Serialize returns the declaration block as CSS text.
func (style *CSSStyleDeclaration) Serialize() string {
	return style.CSSText()
}

// ParseDeclarations parses a declaration block into a mutable declaration
// object.
func ParseDeclarations(input string) *CSSStyleDeclaration {
	declarations := convertDeclarations(parser.ParseComponentValueList(input))
	return &CSSStyleDeclaration{declarations: &declarations}
}

// MediaList is a mutable list of media queries.
type MediaList struct {
	items []string
}

// ParseMediaList parses and normalizes a comma-separated media query list.
func ParseMediaList(text string) (*MediaList, error) {
	list := &MediaList{}
	if strings.TrimSpace(text) == "" {
		return list, nil
	}
	for _, part := range strings.Split(text, ",") {
		list.items = append(list.items, normalizeMediaQuery(part))
	}
	return list, nil
}

func normalizeMediaQuery(text string) string {
	compact := strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(text))), " ")
	if compact == "" {
		return "not all"
	}
	if _, err := mediaqueries.Parse(compact); err != nil {
		return "not all"
	}
	return compact
}

// MediaText returns the normalized media query list.
func (list *MediaList) MediaText() string {
	return strings.Join(list.items, ", ")
}

// SetMediaText replaces the media query list.
func (list *MediaList) SetMediaText(text string) {
	parsed, _ := ParseMediaList(text)
	list.items = parsed.items
}

// Length returns the number of media queries.
func (list *MediaList) Length() int {
	return len(list.items)
}

// Item returns the media query at index, or an empty string.
func (list *MediaList) Item(index int) string {
	if index < 0 || index >= len(list.items) {
		return ""
	}
	return list.items[index]
}

// Matches reports whether any query in the list matches the environment.
func (list *MediaList) Matches(environment mediaqueries.Environment) bool {
	if list == nil || len(list.items) == 0 {
		return true
	}
	for _, item := range list.items {
		query, err := mediaqueries.Parse(item)
		if err == nil && query.Matches(environment) {
			return true
		}
	}
	return false
}

// AppendMedium appends a normalized media query unless it is already present.
func (list *MediaList) AppendMedium(text string) {
	query := normalizeMediaQuery(text)
	for _, existing := range list.items {
		if existing == query {
			return
		}
	}
	list.items = append(list.items, query)
}

// DeleteMedium removes a normalized media query or returns ErrNotFound.
func (list *MediaList) DeleteMedium(text string) error {
	query := normalizeMediaQuery(text)
	for index, existing := range list.items {
		if existing == query {
			list.items = append(list.items[:index], list.items[index+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

// ExpectIdent reports whether a token is an identifier with the given
// ASCII-case-insensitive value.
func ExpectIdent(token tokenizer.Token, name string) bool {
	if token.Type != tokenizer.TokenIdent {
		return false
	}
	return strings.EqualFold(token.Value, name)
}
