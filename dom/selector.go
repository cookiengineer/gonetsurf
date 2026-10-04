package dom

import "strings"

// This file implements a small subset of Selectors for Phase 3: compound
// selectors (tag, #id, .class, [attr], [attr=value]) and comma-separated
// lists. The full Selectors Level 4 engine replaces it in Phase 5.

type attributeSelector struct {
	name     string
	value    string
	hasValue bool
}

type compoundSelector struct {
	tag     string
	id      string
	classes []string
	attrs   []attributeSelector
}

func parseSelectorList(selector string) []compoundSelector {
	var result []compoundSelector
	for _, part := range strings.Split(selector, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		compound, ok := parseCompound(part)
		if !ok {
			return nil
		}
		result = append(result, compound)
	}
	return result
}

func parseCompound(input string) (compoundSelector, bool) {
	var compound compoundSelector
	position := 0
	for position < len(input) {
		character := input[position]
		switch {
		case character == '*':
			position++
		case character == '#':
			position++
			start := position
			for position < len(input) && isIdentifierCharacter(input[position]) {
				position++
			}
			if position == start {
				return compound, false
			}
			compound.id = input[start:position]
		case character == '.':
			position++
			start := position
			for position < len(input) && isIdentifierCharacter(input[position]) {
				position++
			}
			if position == start {
				return compound, false
			}
			compound.classes = append(compound.classes, input[start:position])
		case character == '[':
			end := strings.IndexByte(input[position:], ']')
			if end < 0 {
				return compound, false
			}
			content := input[position+1 : position+end]
			position += end + 1
			attribute := attributeSelector{}
			if equals := strings.IndexByte(content, '='); equals >= 0 {
				attribute.name = strings.TrimSpace(content[:equals])
				attribute.value = strings.Trim(strings.TrimSpace(content[equals+1:]), `"'`)
				attribute.hasValue = true
			} else {
				attribute.name = strings.TrimSpace(content)
			}
			if attribute.name == "" {
				return compound, false
			}
			compound.attrs = append(compound.attrs, attribute)
		case isIdentifierStart(character):
			start := position
			for position < len(input) && isIdentifierCharacter(input[position]) {
				position++
			}
			compound.tag = strings.ToLower(input[start:position])
		default:
			return compound, false
		}
	}
	return compound, true
}

func isIdentifierStart(character byte) bool {
	return (character >= 'a' && character <= 'z') ||
		(character >= 'A' && character <= 'Z') ||
		character == '_' || character == '-'
}

func isIdentifierCharacter(character byte) bool {
	return isIdentifierStart(character) || (character >= '0' && character <= '9')
}

func matchesSelectorList(node *Node, selector string) bool {
	list := parseSelectorList(selector)
	if len(list) == 0 {
		return false
	}
	for _, compound := range list {
		if matchesCompound(node, compound) {
			return true
		}
	}
	return false
}

func matchesCompound(node *Node, compound compoundSelector) bool {
	if node.Type != ElementNode {
		return false
	}
	if compound.tag != "" && compound.tag != "*" && node.Data != compound.tag {
		return false
	}
	if compound.id != "" && node.ID() != compound.id {
		return false
	}
	for _, class := range compound.classes {
		if !node.HasClass(class) {
			return false
		}
	}
	for _, attribute := range compound.attrs {
		value, ok := node.GetAttribute(attribute.name)
		if !ok {
			return false
		}
		if attribute.hasValue && value != attribute.value {
			return false
		}
	}
	return true
}

func (node *Node) querySelector(selector string) *Node {
	var found *Node
	node.Descendants(func(candidate *Node) bool {
		if matchesSelectorList(candidate, selector) {
			found = candidate
			return false
		}
		return true
	})
	return found
}

func (node *Node) querySelectorAll(selector string) []*Node {
	var matches []*Node
	node.Descendants(func(candidate *Node) bool {
		if matchesSelectorList(candidate, selector) {
			matches = append(matches, candidate)
		}
		return true
	})
	return matches
}

// Matches reports whether the element matches a simple selector list.
func (node *Node) Matches(selector string) bool {
	return matchesSelectorList(node, selector)
}

// QuerySelector returns the first descendant matching a simple selector.
func (node *Node) QuerySelector(selector string) *Node {
	return node.querySelector(selector)
}

// QuerySelectorAll returns all descendants matching a simple selector.
func (node *Node) QuerySelectorAll(selector string) []*Node {
	return node.querySelectorAll(selector)
}
