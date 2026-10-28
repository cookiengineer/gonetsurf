// Package cascade implements the CSS cascade and inheritance ordering.
//
// It orders declarations by origin, importance, cascade layer, specificity,
// and document order, and resolves custom properties, var() substitution,
// inheritance, and the defaulting keywords.
//
// See specifications/css/cascade-4.html, specifications/css/cascade-5.html,
// specifications/css/variables-1.html.
package cascade

import (
	"strings"

	"gonetsurf/parsers/css/parser"
	"gonetsurf/parsers/css/properties"
	"gonetsurf/parsers/css/selectors"
	"gonetsurf/parsers/css/tokenizer"
)

// Origin is a declaration's cascade origin.
type Origin uint8

const (
	// OriginUserAgent is the browser's user-agent origin.
	OriginUserAgent Origin = iota
	// OriginUser is the user origin.
	OriginUser
	// OriginAuthor is the author origin.
	OriginAuthor
)

// Declaration is a single declaration participating in the cascade.
type Declaration struct {
	Property    string
	Value       []parser.ComponentValue
	Important   bool
	Origin      Origin
	Specificity selectors.Specificity
	Order       int
	// Layer is the cascade layer index (author origin).
	Layer int
	// Animation marks declarations produced by animations.
	Animation bool
	// Transition marks declarations produced by transitions.
	Transition bool
}

// Result holds the winning declaration for each property and the full
// declaration list used by revert and revert-layer.
type Result struct {
	winners      map[string]Declaration
	declarations []Declaration
}

// InheritedValues maps property names to the computed values inherited from
// an element's parent.
type InheritedValues map[string][]parser.ComponentValue

// IsCustom reports whether a property name is a custom property.
func IsCustom(name string) bool {
	return strings.HasPrefix(name, "--")
}

// Cascade selects the winning declaration for every property.
func Cascade(declarations []Declaration) *Result {
	result := &Result{
		winners:      map[string]Declaration{},
		declarations: append([]Declaration(nil), declarations...),
	}
	for _, declaration := range declarations {
		existing, ok := result.winners[declaration.Property]
		if !ok || declaration.beats(existing) {
			result.winners[declaration.Property] = declaration
		}
	}
	return result
}

// Get returns the winning declaration for a property.
func (result *Result) Get(property string) (Declaration, bool) {
	declaration, ok := result.winners[property]
	return declaration, ok
}

// Properties returns the names of all cascaded properties.
func (result *Result) Properties() []string {
	names := make([]string, 0, len(result.winners))
	for name := range result.winners {
		names = append(names, name)
	}
	return names
}

// originLevel ranks origins together with importance. Higher wins.
func (declaration Declaration) originLevel() int {
	if declaration.Transition {
		return 7
	}
	if declaration.Animation {
		return 3
	}
	if declaration.Important {
		switch declaration.Origin {
		case OriginUserAgent:
			return 6
		case OriginUser:
			return 5
		default:
			return 4
		}
	}
	switch declaration.Origin {
	case OriginUserAgent:
		return 0
	case OriginUser:
		return 1
	default:
		return 2
	}
}

func (declaration Declaration) beats(other Declaration) bool {
	leftLevel := declaration.originLevel()
	rightLevel := other.originLevel()
	if leftLevel != rightLevel {
		return leftLevel > rightLevel
	}
	if declaration.Layer != other.Layer {
		// Later layers win for normal declarations, earlier layers win for
		// important declarations.
		if declaration.Important {
			return declaration.Layer < other.Layer
		}
		return declaration.Layer > other.Layer
	}
	if comparison := declaration.Specificity.Compare(other.Specificity); comparison != 0 {
		return comparison > 0
	}
	return declaration.Order > other.Order
}

// effectiveWinner applies the all shorthand to non-custom properties.
func (result *Result) effectiveWinner(property string) (Declaration, bool) {
	propertyWinner, hasProperty := result.winners[property]
	if IsCustom(property) {
		return propertyWinner, hasProperty
	}
	allWinner, hasAll := result.winners["all"]
	if !hasAll || (hasProperty && !allWinner.beats(propertyWinner)) {
		return propertyWinner, hasProperty
	}
	allWinner.Property = property
	return allWinner, true
}

// Resolve computes the cascaded value of a property, applying inheritance,
// defaulting keywords, and var() substitution.
func (result *Result) Resolve(property string, parent []parser.ComponentValue, initial []parser.ComponentValue) []parser.ComponentValue {
	parentValues := InheritedValues{}
	if parent != nil {
		parentValues[property] = parent
	}
	value, _ := result.ResolveChecked(property, parentValues, initial)
	return value
}

// ResolveWithCustom computes a value using inherited values for custom
// properties as well as the property itself.
func (result *Result) ResolveWithCustom(property string, parent InheritedValues, initial []parser.ComponentValue) []parser.ComponentValue {
	value, _ := result.ResolveChecked(property, parent, initial)
	return value
}

// ResolveChecked computes a value and reports whether the declaration was
// valid at computed-value time. A false result means the value fell back to
// inherit or initial because a var() reference was invalid or cyclic.
func (result *Result) ResolveChecked(property string, parent InheritedValues, initial []parser.ComponentValue) ([]parser.ComponentValue, bool) {
	return result.resolve(property, parent, initial, map[string]bool{})
}

func (result *Result) resolve(property string, parent InheritedValues, initial []parser.ComponentValue, stack map[string]bool) ([]parser.ComponentValue, bool) {
	winner, ok := result.effectiveWinner(property)
	if !ok {
		return inheritedOrInitial(property, parent, initial), true
	}
	keyword := singleKeyword(winner.Value)
	switch keyword {
	case "inherit":
		return parentValue(property, parent, initial), true
	case "initial":
		return initialValue(property, initial), true
	case "unset":
		return inheritedOrInitial(property, parent, initial), true
	case "revert":
		return result.revert(property, winner, parent, initial, stack)
	case "revert-layer":
		return result.revertLayer(property, winner, parent, initial, stack)
	}
	value, valid := result.substitute(winner.Value, parent, stack)
	if !valid {
		return inheritedOrInitial(property, parent, initial), false
	}
	return value, true
}

func (result *Result) revert(property string, current Declaration, parent InheritedValues, initial []parser.ComponentValue, stack map[string]bool) ([]parser.ComponentValue, bool) {
	declarations := make([]Declaration, 0, len(result.declarations))
	for _, declaration := range result.declarations {
		if declaration.Origin < current.Origin {
			declarations = append(declarations, declaration)
		}
	}
	if len(declarations) == 0 {
		return inheritedOrInitial(property, parent, initial), true
	}
	return Cascade(declarations).resolve(property, parent, initial, stack)
}

func (result *Result) revertLayer(property string, current Declaration, parent InheritedValues, initial []parser.ComponentValue, stack map[string]bool) ([]parser.ComponentValue, bool) {
	if current.Layer == 0 {
		return result.revert(property, current, parent, initial, stack)
	}
	declarations := make([]Declaration, 0, len(result.declarations))
	for _, declaration := range result.declarations {
		if declaration.Origin != current.Origin {
			continue
		}
		if declaration.Layer < current.Layer {
			declarations = append(declarations, declaration)
		}
	}
	if len(declarations) == 0 {
		return result.revert(property, current, parent, initial, stack)
	}
	return Cascade(declarations).resolve(property, parent, initial, stack)
}

func (result *Result) substitute(values []parser.ComponentValue, parent InheritedValues, stack map[string]bool) ([]parser.ComponentValue, bool) {
	substituted := make([]parser.ComponentValue, 0, len(values))
	for _, value := range values {
		function, ok := value.(*parser.Function)
		if ok && strings.EqualFold(function.Name, "var") {
			name, fallback, ok := parseVarArguments(function.Value)
			if !ok {
				return nil, false
			}
			custom, valid := result.resolveCustom(name, parent, stack)
			if !valid {
				if fallback == nil {
					return nil, false
				}
				custom, valid = result.substitute(fallback, parent, stack)
				if !valid {
					return nil, false
				}
			}
			substituted = append(substituted, custom...)
			continue
		}
		if function, ok := value.(*parser.Function); ok {
			inner, valid := result.substitute(function.Value, parent, stack)
			if !valid {
				return nil, false
			}
			substituted = append(substituted, &parser.Function{Name: function.Name, Value: inner})
			continue
		}
		if block, ok := value.(*parser.Block); ok {
			inner, valid := result.substitute(block.Value, parent, stack)
			if !valid {
				return nil, false
			}
			substituted = append(substituted, &parser.Block{Associated: block.Associated, Value: inner})
			continue
		}
		substituted = append(substituted, value)
	}
	return substituted, true
}

func (result *Result) resolveCustom(name string, parent InheritedValues, stack map[string]bool) ([]parser.ComponentValue, bool) {
	if stack[name] {
		return nil, false
	}
	stack[name] = true
	defer delete(stack, name)
	winner, ok := result.effectiveWinner(name)
	if !ok {
		if value, inherited := parent[name]; inherited && value != nil {
			return value, true
		}
		return nil, false
	}
	switch singleKeyword(winner.Value) {
	case "inherit", "unset":
		if value, inherited := parent[name]; inherited && value != nil {
			return value, true
		}
		return nil, false
	case "initial":
		return nil, false
	case "revert":
		return result.revert(name, winner, parent, nil, stack)
	case "revert-layer":
		return result.revertLayer(name, winner, parent, nil, stack)
	}
	return result.substitute(winner.Value, parent, stack)
}

func parseVarArguments(values []parser.ComponentValue) (string, []parser.ComponentValue, bool) {
	trimmed := trimWhitespace(values)
	if len(trimmed) == 0 {
		return "", nil, false
	}
	nameToken, ok := parser.AsToken(trimmed[0])
	if !ok || nameToken.Type != tokenizer.TokenIdent || !IsCustom(nameToken.Value) {
		return "", nil, false
	}
	index := 1
	for index < len(trimmed) && parser.IsWhitespace(trimmed[index]) {
		index++
	}
	if index >= len(trimmed) {
		return nameToken.Value, nil, true
	}
	token, ok := parser.AsToken(trimmed[index])
	if !ok || token.Type != tokenizer.TokenComma {
		return "", nil, false
	}
	return nameToken.Value, trimWhitespace(trimmed[index+1:]), true
}

func inheritedOrInitial(property string, parent InheritedValues, initial []parser.ComponentValue) []parser.ComponentValue {
	if properties.IsInherited(property) || IsCustom(property) {
		if value, ok := parent[property]; ok && value != nil {
			return value
		}
		if IsCustom(property) {
			return nil
		}
	}
	return initialValue(property, initial)
}

func parentValue(property string, parent InheritedValues, initial []parser.ComponentValue) []parser.ComponentValue {
	if value, ok := parent[property]; ok && value != nil {
		return value
	}
	return initialValue(property, initial)
}

func initialValue(property string, initial []parser.ComponentValue) []parser.ComponentValue {
	if IsCustom(property) {
		return nil
	}
	return initial
}

func singleKeyword(value []parser.ComponentValue) string {
	trimmed := trimWhitespace(value)
	if len(trimmed) != 1 {
		return ""
	}
	token, ok := parser.AsToken(trimmed[0])
	if !ok || token.Type != tokenizer.TokenIdent {
		return ""
	}
	return strings.ToLower(token.Value)
}

func trimWhitespace(values []parser.ComponentValue) []parser.ComponentValue {
	start := 0
	for start < len(values) && parser.IsWhitespace(values[start]) {
		start++
	}
	end := len(values)
	for end > start && parser.IsWhitespace(values[end-1]) {
		end--
	}
	return values[start:end]
}
