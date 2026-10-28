// Package properties is the CSS property registry.
//
// For every supported property it records whether the property is inherited,
// its initial value, its value-definition grammar, and its animation type.
//
// See the individual CSS modules for each property's definition.
//
//go:generate go run gonetsurf/cmd/propertygen -specs ../../../specifications/css -wpt ../../../testdata/upstream -output registry_gen.go
package properties

import (
	"sort"
	"strings"
)

// AnimationType classifies how a property animates.
type AnimationType uint8

const (
	// NotAnimatable marks properties that cannot be animated.
	NotAnimatable AnimationType = iota
	// Discrete marks properties that animate in discrete steps.
	Discrete
	// ByComputedValue marks properties that animate by interpolation.
	ByComputedValue
)

// Property describes a registered CSS property.
type Property struct {
	Name      string
	Inherited bool
	Initial   string
	Grammar   string
	Animation AnimationType
}

var registry = generatedRegistry

// Lookup returns the registry entry for a property name.
func Lookup(name string) (Property, bool) {
	property, ok := registry[name]
	return property, ok
}

// Names returns the sorted list of registered property names.
func Names() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// IsInherited reports whether the named property inherits by default.
func IsInherited(name string) bool {
	if strings.HasPrefix(name, "--") {
		return true
	}
	property, ok := registry[name]
	return ok && property.Inherited
}

// Len returns the number of registered properties.
func Len() int {
	return len(registry)
}
