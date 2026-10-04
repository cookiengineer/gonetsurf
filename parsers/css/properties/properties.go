// Package properties is the CSS property registry.
//
// For every supported property it records whether the property is inherited,
// its initial value, its value-definition grammar, and its animation type.
//
// See the individual CSS modules for each property's definition.
package properties

import "sort"

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

var registry = map[string]Property{
	"color":               {Name: "color", Inherited: true, Initial: "canvastext", Grammar: "<color>", Animation: ByComputedValue},
	"background-color":    {Name: "background-color", Initial: "transparent", Grammar: "<color>", Animation: ByComputedValue},
	"display":             {Name: "display", Initial: "inline", Grammar: "[ block | inline | run-in ]? && [ flow | flow-root ]? && list-item | [ block | inline | run-in ]? [ flow | flow-root ]? list-item | list-item [ block | inline | run-in ]? [ flow | flow-root ]? | [ block | inline | run-in ] || [ flow | flow-root | table | flex | grid | ruby ] | inline-block | inline-table | inline-flex | inline-grid | table-row-group | table-header-group | table-footer-group | table-row | table-column-group | table-column | table-cell | table-caption | ruby-base | ruby-text | ruby-base-container | ruby-text-container | contents | none | math", Animation: Discrete},
	"visibility":          {Name: "visibility", Inherited: true, Initial: "visible", Grammar: "visible | hidden | collapse", Animation: Discrete},
	"opacity":             {Name: "opacity", Initial: "1", Grammar: "<number> | <percentage>", Animation: ByComputedValue},
	"position":            {Name: "position", Initial: "static", Grammar: "static | relative | absolute | fixed | sticky", Animation: Discrete},
	"top":                 {Name: "top", Initial: "auto", Grammar: "auto | <length> | <percentage>", Animation: ByComputedValue},
	"right":               {Name: "right", Initial: "auto", Grammar: "auto | <length> | <percentage>", Animation: ByComputedValue},
	"bottom":              {Name: "bottom", Initial: "auto", Grammar: "auto | <length> | <percentage>", Animation: ByComputedValue},
	"left":                {Name: "left", Initial: "auto", Grammar: "auto | <length> | <percentage>", Animation: ByComputedValue},
	"z-index":             {Name: "z-index", Initial: "auto", Grammar: "auto | <integer>", Animation: ByComputedValue},
	"width":               {Name: "width", Initial: "auto", Grammar: "auto | <length-percentage-nonnegative> | min-content | max-content | <fit-content>", Animation: ByComputedValue},
	"height":              {Name: "height", Initial: "auto", Grammar: "auto | <length-percentage-nonnegative> | min-content | max-content | <fit-content>", Animation: ByComputedValue},
	"min-width":           {Name: "min-width", Initial: "auto", Grammar: "auto | <length-percentage-nonnegative> | min-content | max-content | <fit-content>", Animation: ByComputedValue},
	"min-height":          {Name: "min-height", Initial: "auto", Grammar: "auto | <length-percentage-nonnegative> | min-content | max-content | <fit-content>", Animation: ByComputedValue},
	"max-width":           {Name: "max-width", Initial: "none", Grammar: "none | <length-percentage-nonnegative> | min-content | max-content | <fit-content>", Animation: ByComputedValue},
	"max-height":          {Name: "max-height", Initial: "none", Grammar: "none | <length-percentage-nonnegative> | min-content | max-content | <fit-content>", Animation: ByComputedValue},
	"margin-top":          {Name: "margin-top", Initial: "0", Grammar: "auto | <length> | <percentage>", Animation: ByComputedValue},
	"margin-right":        {Name: "margin-right", Initial: "0", Grammar: "auto | <length> | <percentage>", Animation: ByComputedValue},
	"margin-bottom":       {Name: "margin-bottom", Initial: "0", Grammar: "auto | <length> | <percentage>", Animation: ByComputedValue},
	"margin-left":         {Name: "margin-left", Initial: "0", Grammar: "auto | <length> | <percentage>", Animation: ByComputedValue},
	"padding-top":         {Name: "padding-top", Initial: "0", Grammar: "<length-percentage-nonnegative>", Animation: ByComputedValue},
	"padding-right":       {Name: "padding-right", Initial: "0", Grammar: "<length-percentage-nonnegative>", Animation: ByComputedValue},
	"padding-bottom":      {Name: "padding-bottom", Initial: "0", Grammar: "<length-percentage-nonnegative>", Animation: ByComputedValue},
	"padding-left":        {Name: "padding-left", Initial: "0", Grammar: "<length-percentage-nonnegative>", Animation: ByComputedValue},
	"font-family":         {Name: "font-family", Inherited: true, Initial: "serif", Grammar: "[ <family-name> | <generic-family> ]#", Animation: Discrete},
	"font-size":           {Name: "font-size", Inherited: true, Initial: "medium", Grammar: "<absolute-size> | <relative-size> | <length-percentage-nonnegative>", Animation: ByComputedValue},
	"font-style":          {Name: "font-style", Inherited: true, Initial: "normal", Grammar: "normal | italic | oblique <angle>?", Animation: Discrete},
	"font-weight":         {Name: "font-weight", Inherited: true, Initial: "normal", Grammar: "<font-weight-absolute> | bolder | lighter", Animation: ByComputedValue},
	"line-height":         {Name: "line-height", Inherited: true, Initial: "normal", Grammar: "normal | <number> | <length> | <percentage>", Animation: ByComputedValue},
	"text-align":          {Name: "text-align", Inherited: true, Initial: "start", Grammar: "start | end | left | right | center | justify | justify-all | match-parent", Animation: Discrete},
	"text-decoration":     {Name: "text-decoration", Initial: "none", Grammar: "none | [ underline || overline || line-through || blink ] || <color> || <line-style> || [ auto | from-font | <length-percentage> ]", Animation: Discrete},
	"overflow-x":          {Name: "overflow-x", Initial: "visible", Grammar: "visible | hidden | clip | scroll | auto", Animation: Discrete},
	"overflow-y":          {Name: "overflow-y", Initial: "visible", Grammar: "visible | hidden | clip | scroll | auto", Animation: Discrete},
	"border-top-width":    {Name: "border-top-width", Initial: "medium", Grammar: "<line-width>", Animation: ByComputedValue},
	"border-right-width":  {Name: "border-right-width", Initial: "medium", Grammar: "<line-width>", Animation: ByComputedValue},
	"border-bottom-width": {Name: "border-bottom-width", Initial: "medium", Grammar: "<line-width>", Animation: ByComputedValue},
	"border-left-width":   {Name: "border-left-width", Initial: "medium", Grammar: "<line-width>", Animation: ByComputedValue},
	"border-top-color":    {Name: "border-top-color", Initial: "currentcolor", Grammar: "<color>", Animation: ByComputedValue},
	"border-top-style":    {Name: "border-top-style", Initial: "none", Grammar: "<line-style>", Animation: Discrete},
	"animation-name":      {Name: "animation-name", Initial: "none", Grammar: "[ none | <keyframes-name> ]#", Animation: Discrete},
	"animation-duration":  {Name: "animation-duration", Initial: "0s", Grammar: "<time-nonnegative>#", Animation: ByComputedValue},
	"transition-property": {Name: "transition-property", Initial: "all", Grammar: "[ none | <custom-ident> | all ]#", Animation: Discrete},
}

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
	property, ok := registry[name]
	return ok && property.Inherited
}

// Len returns the number of registered properties.
func Len() int {
	return len(registry)
}
