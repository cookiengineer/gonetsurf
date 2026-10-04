package selectors

import "strings"

// knownPseudoClasses lists the pseudo-classes Selectors Level 4 defines.
var knownPseudoClasses = map[string]bool{
	"active": true, "any-link": true, "autofill": true, "blank": true,
	"buffering": true, "checked": true, "current": true, "default": true,
	"defined": true, "disabled": true, "empty": true, "enabled": true,
	"first-child": true, "first-of-type": true, "focus": true,
	"focus-visible": true, "focus-within": true, "fullscreen": true,
	"future": true, "hover": true, "in-range": true, "indeterminate": true,
	"invalid": true, "last-child": true, "last-of-type": true, "link": true,
	"local-link": true, "modal": true, "muted": true, "only-child": true,
	"only-of-type": true, "optional": true, "out-of-range": true, "past": true,
	"paused": true, "picture-in-picture": true, "placeholder-shown": true,
	"playing": true, "read-only": true, "read-write": true, "required": true,
	"root": true, "scope": true, "seeking": true, "stalled": true,
	"target": true, "target-within": true, "user-invalid": true,
	"user-valid": true, "valid": true, "visited": true, "volume-locked": true,
	// Functional.
	"dir": true, "has": true, "has-slotted": true, "host": true,
	"host-context": true, "is": true, "lang": true, "not": true,
	"nth-child": true, "nth-col": true, "nth-last-child": true,
	"nth-last-col": true, "nth-last-of-type": true, "nth-of-type": true,
	"state": true, "where": true,
}

// requiresFunctionPseudoClasses must be written with parentheses. :host is
// excluded because it is valid both with and without an argument.
var requiresFunctionPseudoClasses = map[string]bool{
	"dir": true, "has": true, "has-slotted": true, "host-context": true,
	"is": true, "lang": true, "not": true, "nth-child": true, "nth-col": true,
	"nth-last-child": true, "nth-last-col": true, "nth-last-of-type": true,
	"nth-of-type": true, "state": true, "where": true,
}

// knownPseudoElements lists the pseudo-elements Selectors Level 4 defines.
var knownPseudoElements = map[string]bool{
	"after": true, "backdrop": true, "before": true, "checkmark": true,
	"cue": true, "cue-region": true, "details-content": true,
	"file-selector-button": true, "first-letter": true, "first-line": true,
	"grammar-error": true, "highlight": true, "marker": true, "part": true,
	"picker": true, "picker-icon": true, "placeholder": true, "selection": true,
	"slotted": true, "spelling-error": true, "target-text": true,
	"view-transition": true, "view-transition-group": true,
	"view-transition-image-pair": true, "view-transition-new": true,
	"view-transition-old": true,
}

// functionalPseudoElements require an argument.
var functionalPseudoElements = map[string]bool{
	"cue": true, "cue-region": true, "highlight": true, "part": true,
	"slotted": true, "view-transition-group": true,
	"view-transition-image-pair": true, "view-transition-new": true,
	"view-transition-old": true,
}

func isKnownPseudoClass(name string) bool {
	return knownPseudoClasses[strings.ToLower(name)]
}

func requiresFunctionPseudoClass(name string) bool {
	return requiresFunctionPseudoClasses[strings.ToLower(name)]
}

func isKnownPseudoElement(name string) bool {
	return knownPseudoElements[strings.ToLower(name)]
}

func isFunctionalPseudoElement(name string) bool {
	return functionalPseudoElements[strings.ToLower(name)]
}
