package dom

import (
	"strings"

	"gonetsurf/parsers/css/selectors"
)

// Matches reports whether the element matches a selector list.
func (node *Node) Matches(selector string) bool {
	list, err := selectors.Parse(selector)
	if err != nil || list == nil {
		return false
	}
	return node.MatchesSelectorList(list)
}

// MatchesSelectorList reports whether the element matches a parsed selector
// list. The element itself is used as :scope.
func (node *Node) MatchesSelectorList(list *selectors.SelectorList) bool {
	return node.matchesSelectorList(list, node)
}

func (node *Node) matchesSelectorList(list *selectors.SelectorList, scope *Node) bool {
	if node == nil || node.Type != ElementNode || list == nil {
		return false
	}
	for _, complexSelector := range list.Selectors {
		if complexSelector.Leading != nil {
			continue
		}
		if node.matchesComplex(complexSelector, scope) {
			return true
		}
	}
	return false
}

func (node *Node) matchesComplex(complexSelector selectors.ComplexSelector, scope *Node) bool {
	if len(complexSelector.Compounds) == 0 {
		return false
	}
	_, ok := node.matchComplexAt(complexSelector, len(complexSelector.Compounds)-1, scope)
	return ok
}

func (node *Node) matchComplexAt(complexSelector selectors.ComplexSelector, index int, scope *Node) (*Node, bool) {
	if !node.matchesCompound(complexSelector.Compounds[index], scope) {
		return nil, false
	}
	if index == 0 {
		return node, true
	}
	combinator := complexSelector.Combinators[index-1]
	switch combinator {
	case selectors.Descendant:
		for ancestor := node.ParentElement(); ancestor != nil; ancestor = ancestor.ParentElement() {
			if leftmost, ok := ancestor.matchComplexAt(complexSelector, index-1, scope); ok {
				return leftmost, true
			}
		}
	case selectors.Child:
		if parent := node.ParentElement(); parent != nil {
			return parent.matchComplexAt(complexSelector, index-1, scope)
		}
	case selectors.NextSibling:
		if sibling := previousElementSibling(node); sibling != nil {
			return sibling.matchComplexAt(complexSelector, index-1, scope)
		}
	case selectors.SubsequentSibling:
		for sibling := previousElementSibling(node); sibling != nil; sibling = previousElementSibling(sibling) {
			if leftmost, ok := sibling.matchComplexAt(complexSelector, index-1, scope); ok {
				return leftmost, true
			}
		}
	}
	return nil, false
}

func (node *Node) matchesCompound(compound selectors.CompoundSelector, scope *Node) bool {
	for _, part := range compound.Parts {
		switch typed := part.(type) {
		case *selectors.TypeSelector:
			if !node.matchesTypeSelector(typed) {
				return false
			}
		case *selectors.IDSelector:
			if node.ID() != typed.Name {
				return false
			}
		case *selectors.ClassSelector:
			if !node.HasClass(typed.Name) {
				return false
			}
		case *selectors.AttributeSelector:
			if !node.matchesAttributeSelector(typed) {
				return false
			}
		case *selectors.PseudoClass:
			if !node.matchesPseudoClass(typed, scope) {
				return false
			}
		case *selectors.PseudoElement:
			return false
		case *selectors.NestingSelector:
			if node != scope {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func (node *Node) matchesTypeSelector(selector *selectors.TypeSelector) bool {
	if selector.Namespace != "" && selector.Namespace != "*" && node.Namespace != selector.Namespace {
		return false
	}
	if selector.Universal {
		return true
	}
	if node.Namespace == NamespaceHTML {
		return strings.EqualFold(node.Data, selector.Name)
	}
	return node.Data == selector.Name
}

func (node *Node) matchesAttributeSelector(selector *selectors.AttributeSelector) bool {
	for _, attribute := range node.Attr {
		if !attributeNamespaceMatches(selector.Namespace, attribute) {
			continue
		}
		nameMatches := attribute.Name == selector.Name
		if node.Namespace == NamespaceHTML {
			nameMatches = strings.EqualFold(attribute.Name, selector.Name)
		}
		if !nameMatches {
			continue
		}
		if selector.Operator == "" {
			return true
		}
		actual := attribute.Value
		expected := selector.Value
		if selector.CaseInsensitive {
			actual = strings.ToLower(actual)
			expected = strings.ToLower(expected)
		}
		if attributeValueMatches(selector.Operator, actual, expected) {
			return true
		}
	}
	return false
}

func attributeNamespaceMatches(namespace string, attribute Attribute) bool {
	if namespace == "*" {
		return true
	}
	if namespace == "" {
		return attribute.Namespace == ""
	}
	return attribute.Namespace == namespace || attribute.Prefix == namespace
}

func attributeValueMatches(operator, actual, expected string) bool {
	switch operator {
	case "=":
		return actual == expected
	case "~=":
		if expected == "" {
			return false
		}
		for _, part := range strings.Fields(actual) {
			if part == expected {
				return true
			}
		}
	case "|=":
		return actual == expected || strings.HasPrefix(actual, expected+"-")
	case "^=":
		return expected != "" && strings.HasPrefix(actual, expected)
	case "$=":
		return expected != "" && strings.HasSuffix(actual, expected)
	case "*=":
		return expected != "" && strings.Contains(actual, expected)
	}
	return false
}

func (node *Node) matchesPseudoClass(pseudo *selectors.PseudoClass, scope *Node) bool {
	name := strings.ToLower(pseudo.Name)
	switch name {
	case "scope":
		return node == scopeElement(scope)
	case "root":
		owner := node.OwnerDocument()
		return owner != nil && owner.DocumentElement() == node
	case "empty":
		return node.isEmpty()
	case "first-child":
		index, _ := elementIndex(node, false, nil, scope)
		return index == 1
	case "last-child":
		index, count := elementIndex(node, false, nil, scope)
		return index == count
	case "only-child":
		index, count := elementIndex(node, false, nil, scope)
		return index == 1 && count == 1
	case "first-of-type":
		index, _ := elementIndex(node, true, nil, scope)
		return index == 1
	case "last-of-type":
		index, count := elementIndex(node, true, nil, scope)
		return index == count
	case "only-of-type":
		index, count := elementIndex(node, true, nil, scope)
		return index == 1 && count == 1
	case "nth-child", "nth-last-child", "nth-of-type", "nth-last-of-type":
		if pseudo.AnB == nil {
			return false
		}
		byType := name == "nth-of-type" || name == "nth-last-of-type"
		index, count := elementIndex(node, byType, pseudo.OfSelector, scope)
		if index == 0 {
			return false
		}
		if strings.HasPrefix(name, "nth-last") {
			index = count - index + 1
		}
		return matchesAnB(*pseudo.AnB, index)
	case "is", "where":
		return node.matchesSelectorList(pseudo.Selectors, scope)
	case "not":
		return !node.matchesSelectorList(pseudo.Selectors, scope)
	case "has":
		return node.matchesHas(pseudo.Selectors)
	case "lang":
		return node.matchesLanguage(pseudo.Argument)
	case "dir":
		direction := node.direction()
		return strings.EqualFold(direction, strings.Trim(pseudo.Argument, "\"'"))
	case "link", "any-link":
		return node.isLink()
	case "visited":
		return false
	case "enabled":
		return isFormControl(node) && !isDisabled(node)
	case "disabled":
		return isFormControl(node) && isDisabled(node)
	case "checked":
		return node.isChecked()
	case "required":
		return isFormControl(node) && node.HasAttribute("required")
	case "optional":
		return isFormControl(node) && !node.HasAttribute("required")
	case "read-write":
		return node.isEditable()
	case "read-only":
		return !node.isEditable()
	case "defined":
		return true
	case "default", "indeterminate", "placeholder-shown", "valid", "invalid",
		"in-range", "out-of-range", "user-valid", "user-invalid", "target",
		"target-within", "focus", "focus-visible", "focus-within", "hover",
		"active", "current", "past", "future", "playing", "paused",
		"seeking", "buffering", "stalled", "muted", "volume-locked",
		"picture-in-picture", "fullscreen", "modal", "autofill", "blank",
		"local-link", "state", "host", "host-context", "has-slotted":
		return false
	default:
		return false
	}
}

func scopeElement(scope *Node) *Node {
	if scope == nil {
		return nil
	}
	if scope.Type == DocumentNode {
		for _, child := range scope.Children() {
			if child.Type == ElementNode {
				return child
			}
		}
		return nil
	}
	return scope
}

func (node *Node) matchesHas(list *selectors.SelectorList) bool {
	if list == nil {
		return false
	}
	for _, complexSelector := range list.Selectors {
		if node.matchesRelative(complexSelector) {
			return true
		}
	}
	return false
}

func (node *Node) matchesRelative(complexSelector selectors.ComplexSelector) bool {
	if len(complexSelector.Compounds) == 0 {
		return false
	}
	combinator := selectors.Descendant
	if complexSelector.Leading != nil {
		combinator = *complexSelector.Leading
	}
	var candidates []*Node
	switch combinator {
	case selectors.Descendant:
		node.Descendants(func(candidate *Node) bool {
			if candidate.Type == ElementNode {
				candidates = append(candidates, candidate)
			}
			return true
		})
		if compoundContainsScope(complexSelector.Compounds[0]) {
			candidates = append([]*Node{node}, candidates...)
		}
	case selectors.Child:
		for _, child := range node.Children() {
			if child.Type == ElementNode {
				candidates = append(candidates, child)
			}
		}
	case selectors.NextSibling:
		if sibling := nextElementSibling(node); sibling != nil {
			candidates = append(candidates, sibling)
		}
	case selectors.SubsequentSibling:
		for sibling := nextElementSibling(node); sibling != nil; sibling = nextElementSibling(sibling) {
			candidates = append(candidates, sibling)
		}
	default:
		return false
	}
	for _, candidate := range candidates {
		if candidate.matchesComplexForward(complexSelector, 0, node) {
			return true
		}
	}
	return false
}

func (node *Node) matchesComplexForward(complexSelector selectors.ComplexSelector, index int, scope *Node) bool {
	if !node.matchesCompound(complexSelector.Compounds[index], scope) {
		return false
	}
	if index == len(complexSelector.Compounds)-1 {
		return true
	}
	combinator := complexSelector.Combinators[index]
	switch combinator {
	case selectors.Descendant:
		matched := false
		node.Descendants(func(candidate *Node) bool {
			if candidate.Type == ElementNode && candidate.matchesComplexForward(complexSelector, index+1, scope) {
				matched = true
				return false
			}
			return true
		})
		return matched
	case selectors.Child:
		for _, child := range node.Children() {
			if child.Type == ElementNode && child.matchesComplexForward(complexSelector, index+1, scope) {
				return true
			}
		}
	case selectors.NextSibling:
		if sibling := nextElementSibling(node); sibling != nil {
			return sibling.matchesComplexForward(complexSelector, index+1, scope)
		}
	case selectors.SubsequentSibling:
		for sibling := nextElementSibling(node); sibling != nil; sibling = nextElementSibling(sibling) {
			if sibling.matchesComplexForward(complexSelector, index+1, scope) {
				return true
			}
		}
	}
	return false
}

func compoundContainsScope(compound selectors.CompoundSelector) bool {
	for _, part := range compound.Parts {
		pseudo, ok := part.(*selectors.PseudoClass)
		if !ok {
			continue
		}
		if strings.EqualFold(pseudo.Name, "scope") {
			return true
		}
		if pseudo.Selectors != nil {
			for _, complexSelector := range pseudo.Selectors.Selectors {
				for _, nested := range complexSelector.Compounds {
					if compoundContainsScope(nested) {
						return true
					}
				}
			}
		}
	}
	return false
}

func (node *Node) isEmpty() bool {
	for _, child := range node.Children() {
		if child.Type == ElementNode || child.Type == TextNode {
			return false
		}
	}
	return true
}

func (node *Node) matchesLanguage(argument string) bool {
	language := strings.ToLower(strings.Trim(argument, "\"'"))
	if language == "" {
		return false
	}
	for current := node; current != nil; current = current.ParentElement() {
		if value, ok := current.GetAttribute("lang"); ok {
			value = strings.ToLower(value)
			return value == language || strings.HasPrefix(value, language+"-")
		}
	}
	return false
}

func (node *Node) direction() string {
	for current := node; current != nil; current = current.ParentElement() {
		if value, ok := current.GetAttribute("dir"); ok {
			value = strings.ToLower(value)
			if value == "ltr" || value == "rtl" {
				return value
			}
		}
	}
	return "ltr"
}

func (node *Node) isLink() bool {
	if node.Namespace != NamespaceHTML {
		return false
	}
	switch node.Data {
	case "a", "area", "link":
		_, ok := node.GetAttribute("href")
		return ok
	}
	return false
}

func isFormControl(node *Node) bool {
	if node.Namespace != NamespaceHTML {
		return false
	}
	switch node.Data {
	case "button", "input", "select", "textarea", "optgroup", "option", "fieldset":
		return true
	}
	return false
}

func isDisabled(node *Node) bool {
	if node.HasAttribute("disabled") {
		return true
	}
	if node.Data == "option" {
		if parent := node.ParentElement(); parent != nil && parent.Data == "optgroup" && parent.HasAttribute("disabled") {
			return true
		}
	}
	return false
}

func (node *Node) isChecked() bool {
	if node.Namespace != NamespaceHTML {
		return false
	}
	switch node.Data {
	case "input":
		inputType, _ := node.GetAttribute("type")
		inputType = strings.ToLower(inputType)
		return (inputType == "checkbox" || inputType == "radio") && node.HasAttribute("checked")
	case "option":
		return node.HasAttribute("selected")
	}
	return false
}

func (node *Node) isEditable() bool {
	if contenteditable, ok := node.GetAttribute("contenteditable"); ok {
		return !strings.EqualFold(contenteditable, "false")
	}
	if node.Namespace != NamespaceHTML {
		return false
	}
	switch node.Data {
	case "textarea":
		return !node.HasAttribute("readonly") && !isDisabled(node)
	case "input":
		if node.HasAttribute("readonly") || isDisabled(node) {
			return false
		}
		inputType, _ := node.GetAttribute("type")
		switch strings.ToLower(inputType) {
		case "text", "search", "url", "tel", "email", "password", "date", "month",
			"week", "time", "datetime-local", "number":
			return true
		}
	}
	return false
}

func matchesAnB(anB selectors.AnB, index int) bool {
	if anB.A == 0 {
		return index == anB.B
	}
	if anB.A > 0 {
		difference := index - anB.B
		return difference >= 0 && difference%anB.A == 0
	}
	difference := anB.B - index
	return difference >= 0 && difference%(-anB.A) == 0
}

func elementIndex(node *Node, byType bool, ofSelector *selectors.SelectorList, scope *Node) (int, int) {
	siblings := elementSiblings(node)
	filtered := make([]*Node, 0, len(siblings))
	for _, sibling := range siblings {
		if byType && !sameElementType(sibling, node) {
			continue
		}
		if ofSelector != nil && !sibling.matchesSelectorList(ofSelector, scope) {
			continue
		}
		filtered = append(filtered, sibling)
	}
	for index, sibling := range filtered {
		if sibling == node {
			return index + 1, len(filtered)
		}
	}
	return 0, len(filtered)
}

func elementSiblings(node *Node) []*Node {
	if node.Parent() == nil {
		return nil
	}
	siblings := make([]*Node, 0, len(node.Parent().Children()))
	for _, child := range node.Parent().Children() {
		if child.Type == ElementNode {
			siblings = append(siblings, child)
		}
	}
	return siblings
}

func sameElementType(left, right *Node) bool {
	return left.Namespace == right.Namespace && left.Data == right.Data
}

func previousElementSibling(node *Node) *Node {
	if node.Parent() == nil {
		return nil
	}
	var previous *Node
	for _, sibling := range node.Parent().Children() {
		if sibling == node {
			return previous
		}
		if sibling.Type == ElementNode {
			previous = sibling
		}
	}
	return nil
}

func nextElementSibling(node *Node) *Node {
	if node.Parent() == nil {
		return nil
	}
	found := false
	for _, sibling := range node.Parent().Children() {
		if found && sibling.Type == ElementNode {
			return sibling
		}
		if sibling == node {
			found = true
		}
	}
	return nil
}

func (node *Node) querySelector(selector string) *Node {
	list, err := selectors.Parse(selector)
	if err != nil || list == nil {
		return nil
	}
	return node.QuerySelectorList(list)
}

func (node *Node) querySelectorAll(selector string) []*Node {
	list, err := selectors.Parse(selector)
	if err != nil || list == nil {
		return nil
	}
	return node.QuerySelectorAllList(list)
}

// QuerySelector returns the first descendant matching a selector.
func (node *Node) QuerySelector(selector string) *Node {
	return node.querySelector(selector)
}

// QuerySelectorAll returns all descendants matching a selector.
func (node *Node) QuerySelectorAll(selector string) []*Node {
	return node.querySelectorAll(selector)
}

// QuerySelectorList returns the first descendant matching a parsed selector
// list. The receiver is used as :scope.
func (node *Node) QuerySelectorList(list *selectors.SelectorList) *Node {
	var found *Node
	node.Descendants(func(candidate *Node) bool {
		if candidate.Type == ElementNode && candidate.matchesSelectorList(list, node) {
			found = candidate
			return false
		}
		return true
	})
	return found
}

// QuerySelectorAllList returns all descendants matching a parsed selector
// list. The receiver is used as :scope.
func (node *Node) QuerySelectorAllList(list *selectors.SelectorList) []*Node {
	var matches []*Node
	node.Descendants(func(candidate *Node) bool {
		if candidate.Type == ElementNode && candidate.matchesSelectorList(list, node) {
			matches = append(matches, candidate)
		}
		return true
	})
	return matches
}
