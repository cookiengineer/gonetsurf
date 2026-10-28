package styles

import (
	"sort"
	"strings"

	"gonetsurf/dom"
	"gonetsurf/parsers/css/selectors"
)

// Matcher matches parsed selector lists against DOM elements and maintains
// candidate indexes for stylesheet application.
type Matcher struct {
	root        *dom.Node
	byID        map[string][]*dom.Node
	byClass     map[string][]*dom.Node
	byTag       map[string][]*dom.Node
	byAttribute map[string][]*dom.Node
	order       map[*dom.Node]int
	all         []*dom.Node
}

// NewMatcher builds indexes for root and its descendants.
func NewMatcher(root *dom.Node) *Matcher {
	matcher := &Matcher{root: root}
	matcher.Rebuild()
	return matcher
}

// NewDocumentMatcher builds indexes for a document's element tree.
func NewDocumentMatcher(document *dom.Document) *Matcher {
	if document == nil {
		return NewMatcher(nil)
	}
	return NewMatcher(&document.Node)
}

// Root returns the matcher's root node.
func (matcher *Matcher) Root() *dom.Node {
	return matcher.root
}

// Rebuild recomputes the document-order element list and all indexes.
func (matcher *Matcher) Rebuild() {
	matcher.byID = map[string][]*dom.Node{}
	matcher.byClass = map[string][]*dom.Node{}
	matcher.byTag = map[string][]*dom.Node{}
	matcher.byAttribute = map[string][]*dom.Node{}
	matcher.order = map[*dom.Node]int{}
	matcher.all = nil
	if matcher.root == nil {
		return
	}
	if matcher.root.Type == dom.ElementNode {
		matcher.add(matcher.root)
	}
	matcher.root.Descendants(func(node *dom.Node) bool {
		if node.Type == dom.ElementNode {
			matcher.add(node)
		}
		return true
	})
}

func (matcher *Matcher) add(element *dom.Node) {
	matcher.order[element] = len(matcher.all)
	matcher.all = append(matcher.all, element)
	matcher.byTag[strings.ToLower(element.Data)] = append(matcher.byTag[strings.ToLower(element.Data)], element)
	if id := element.ID(); id != "" {
		matcher.byID[id] = append(matcher.byID[id], element)
	}
	for _, class := range element.ClassList() {
		matcher.byClass[class] = append(matcher.byClass[class], element)
	}
	for _, attribute := range element.Attr {
		name := strings.ToLower(attribute.Name)
		matcher.byAttribute[name] = append(matcher.byAttribute[name], element)
	}
}

// All returns the elements in document order.
func (matcher *Matcher) All() []*dom.Node {
	return matcher.all
}

// Match reports whether element matches the selector list.
func (matcher *Matcher) Match(element *dom.Node, list *selectors.SelectorList) bool {
	return element.MatchesSelectorList(list)
}

// Elements returns all elements matching the selector list in document order.
func (matcher *Matcher) Elements(list *selectors.SelectorList) []*dom.Node {
	if list == nil {
		return nil
	}
	seen := map[*dom.Node]bool{}
	var result []*dom.Node
	for _, complexSelector := range list.Selectors {
		if complexSelector.Leading != nil {
			continue
		}
		for _, candidate := range matcher.candidates(complexSelector) {
			if seen[candidate] {
				continue
			}
			if candidate.MatchesSelectorList(list) {
				seen[candidate] = true
				result = append(result, candidate)
			}
		}
	}
	sort.Slice(result, func(left, right int) bool {
		return matcher.order[result[left]] < matcher.order[result[right]]
	})
	return result
}

// candidates narrows a complex selector using the most selective simple
// selector in its rightmost compound.
func (matcher *Matcher) candidates(complexSelector selectors.ComplexSelector) []*dom.Node {
	if len(complexSelector.Compounds) == 0 {
		return matcher.all
	}
	rightmost := complexSelector.Compounds[len(complexSelector.Compounds)-1]
	var best []*dom.Node
	consider := func(candidates []*dom.Node) {
		if candidates == nil {
			return
		}
		if best == nil || len(candidates) < len(best) {
			best = candidates
		}
	}
	for _, part := range rightmost.Parts {
		switch typed := part.(type) {
		case *selectors.IDSelector:
			consider(matcher.byID[typed.Name])
		case *selectors.ClassSelector:
			consider(matcher.byClass[typed.Name])
		case *selectors.TypeSelector:
			if !typed.Universal {
				consider(matcher.byTag[strings.ToLower(typed.Name)])
			}
		case *selectors.AttributeSelector:
			consider(matcher.byAttribute[strings.ToLower(typed.Name)])
		}
	}
	if best == nil {
		return matcher.all
	}
	return best
}
