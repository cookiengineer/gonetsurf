// Package cascade implements the CSS cascade and inheritance ordering.
//
// It orders declarations by origin, importance, cascade layer, specificity,
// and document order, and records the winning value for each property.
//
// See specifications/css/cascade-4.html, specifications/css/cascade-5.html,
// specifications/css/css-variables-1.html.
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

// Result holds the winning declaration for each property.
type Result struct {
	winners map[string]Declaration
}

// IsCustom reports whether a property name is a custom property.
func IsCustom(name string) bool {
	return strings.HasPrefix(name, "--")
}

// Cascade selects the winning declaration for every property.
func Cascade(declarations []Declaration) *Result {
	result := &Result{winners: map[string]Declaration{}}
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

// Resolve computes the cascaded value of a property, applying inheritance and
// the defaulting keywords.
func (result *Result) Resolve(property string, parent []parser.ComponentValue, initial []parser.ComponentValue) []parser.ComponentValue {
	declaration, ok := result.Get(property)
	if !ok {
		if properties.IsInherited(property) && parent != nil {
			return parent
		}
		return initial
	}
	keyword := singleKeyword(declaration.Value)
	switch keyword {
	case "inherit":
		return parent
	case "initial":
		return initial
	case "unset":
		if properties.IsInherited(property) {
			return parent
		}
		return initial
	case "revert", "revert-layer":
		return initial
	default:
		return declaration.Value
	}
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
