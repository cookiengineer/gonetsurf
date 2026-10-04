package dom

import "strings"

// SandboxFlags is a set of sandboxing restrictions.
type SandboxFlags uint32

// Sandboxing flags. A restrictive context starts with all flags set; allow-*
// tokens clear them.
const (
	SandboxNavigation SandboxFlags = 1 << iota
	SandboxAuxiliaryNavigation
	SandboxTopLevelNavigation
	SandboxTopLevelNavigationWithoutUserActivation
	SandboxPlugins
	SandboxOrigin
	SandboxForms
	SandboxPointerLock
	SandboxScripts
	SandboxAutomaticFeatures
	SandboxDocumentDomain
	SandboxModals
	SandboxOrientationLock
	SandboxPresentation
	SandboxDownloads
)

// allSandboxFlags is the restrictive default.
const allSandboxFlags = SandboxNavigation | SandboxAuxiliaryNavigation |
	SandboxTopLevelNavigation | SandboxTopLevelNavigationWithoutUserActivation |
	SandboxPlugins | SandboxOrigin | SandboxForms | SandboxPointerLock |
	SandboxScripts | SandboxAutomaticFeatures | SandboxDocumentDomain |
	SandboxModals | SandboxOrientationLock | SandboxPresentation | SandboxDownloads

// ParseSandboxDirective parses an iframe sandbox attribute value into a
// sandboxing flag set.
func ParseSandboxDirective(tokens string) SandboxFlags {
	flags := allSandboxFlags
	for _, token := range strings.Fields(tokens) {
		switch strings.ToLower(token) {
		case "allow-same-origin":
			flags &^= SandboxOrigin
		case "allow-scripts":
			flags &^= SandboxScripts
		case "allow-forms":
			flags &^= SandboxForms
		case "allow-popups":
			flags &^= SandboxAuxiliaryNavigation
		case "allow-top-navigation":
			flags &^= SandboxTopLevelNavigation
			flags &^= SandboxTopLevelNavigationWithoutUserActivation
		case "allow-top-navigation-by-user-activation":
			flags &^= SandboxTopLevelNavigationWithoutUserActivation
		case "allow-pointer-lock":
			flags &^= SandboxPointerLock
		case "allow-modals":
			flags &^= SandboxModals
		case "allow-orientation-lock":
			flags &^= SandboxOrientationLock
		case "allow-presentation":
			flags &^= SandboxPresentation
		case "allow-downloads":
			flags &^= SandboxDownloads
		case "allow-popups-to-escape-sandbox":
			// Applies to popups, not the document itself.
		case "allow-scripts-to-escape-sandbox":
			// No effect in the current model.
		}
	}
	return flags
}

// AllowsScripts reports whether scripts are permitted.
func (flags SandboxFlags) AllowsScripts() bool {
	return flags&SandboxScripts == 0
}

// AllowsSameOrigin reports whether the document keeps its origin.
func (flags SandboxFlags) AllowsSameOrigin() bool {
	return flags&SandboxOrigin == 0
}

// AllowsNavigation reports whether navigation is permitted.
func (flags SandboxFlags) AllowsNavigation() bool {
	return flags&SandboxNavigation == 0
}
