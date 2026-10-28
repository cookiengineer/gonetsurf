package dom

import "gonetsurf/urls"

// QuirksMode is the document's HTML parsing mode.
type QuirksMode uint8

// Quirks modes.
const (
	NoQuirks QuirksMode = iota
	LimitedQuirks
	Quirks
)

// ReadyState is the document load state.
type ReadyState uint8

// Ready states.
const (
	ReadyLoading ReadyState = iota
	ReadyInteractive
	ReadyComplete
)

// Document is a DOM document.
type Document struct {
	Node

	URL         *urls.URL
	ContentType string
	Quirks      QuirksMode
	Origin      urls.Origin
	Sandbox     SandboxFlags
	ReadyState  ReadyState

	idIndex           map[string]*Node
	indexDirty        bool
	mutationVersion   uint64
	structuralVersion uint64
	dirtyNodes        map[*Node]bool
}

// NewDocument returns a new empty document.
func NewDocument() *Document {
	document := &Document{}
	document.Type = DocumentNode
	document.owner = document
	document.indexDirty = true
	return document
}

// CreateElement creates an HTML element.
func (document *Document) CreateElement(localName string) *Node {
	return document.CreateElementNS(NamespaceHTML, localName)
}

// CreateElementNS creates an element in a namespace.
func (document *Document) CreateElementNS(namespace, localName string) *Node {
	node := NewElement(namespace, localName)
	node.owner = document
	document.markDirty()
	return node
}

// CreateTextNode creates a text node.
func (document *Document) CreateTextNode(data string) *Node {
	node := NewText(data)
	node.owner = document
	return node
}

// CreateComment creates a comment node.
func (document *Document) CreateComment(data string) *Node {
	node := NewComment(data)
	node.owner = document
	return node
}

// CreateDocumentFragment creates a document fragment.
func (document *Document) CreateDocumentFragment() *Node {
	node := NewDocumentFragment()
	node.owner = document
	return node
}

// CreateDocumentType creates a document type node.
func (document *Document) CreateDocumentType(name, publicID, systemID string) *Node {
	node := &Node{Type: DocumentTypeNode, Data: name, PublicID: publicID, SystemID: systemID}
	node.owner = document
	return node
}

// markDirty invalidates derived indexes and advances the mutation version.
func (document *Document) markDirty() {
	document.indexDirty = true
	document.mutationVersion++
}

func (document *Document) markStructural() {
	document.structuralVersion++
	document.markDirty()
}

// StructuralVersion returns a counter incremented only by tree-shape changes.
func (document *Document) StructuralVersion() uint64 {
	return document.structuralVersion
}

func (document *Document) markNodeDirty(node *Node) {
	if document.dirtyNodes == nil {
		document.dirtyNodes = map[*Node]bool{}
	}
	document.dirtyNodes[node] = true
}

// MutationVersion returns a monotonically increasing mutation counter.
func (document *Document) MutationVersion() uint64 {
	return document.mutationVersion
}

// DirtyNodes returns the nodes mutated since the last ClearDirty call.
func (document *Document) DirtyNodes() []*Node {
	nodes := make([]*Node, 0, len(document.dirtyNodes))
	for node := range document.dirtyNodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// ClearDirty clears the dirty-node set.
func (document *Document) ClearDirty() {
	document.dirtyNodes = nil
}

// rebuildIndex rebuilds the id index.
func (document *Document) rebuildIndex() {
	document.idIndex = make(map[string]*Node)
	document.Descendants(func(node *Node) bool {
		if node.Type == ElementNode {
			if id := node.ID(); id != "" {
				if _, exists := document.idIndex[id]; !exists {
					document.idIndex[id] = node
				}
			}
		}
		return true
	})
	document.indexDirty = false
}

// GetElementById returns the first element with the given id.
func (document *Document) GetElementById(id string) *Node {
	if document.indexDirty || document.idIndex == nil {
		document.rebuildIndex()
	}
	return document.idIndex[id]
}

// DocumentElement returns the root <html> element.
func (document *Document) DocumentElement() *Node {
	for _, child := range document.children {
		if child.Type == ElementNode {
			return child
		}
	}
	return nil
}

// Head returns the first <head> element.
func (document *Document) Head() *Node {
	return document.firstElementByLocalName("head")
}

// Body returns the first <body> element.
func (document *Document) Body() *Node {
	return document.firstElementByLocalName("body")
}

// Title returns the document title per the HTML Standard.
func (document *Document) Title() string {
	var title string
	document.Descendants(func(node *Node) bool {
		if node.Type == ElementNode && node.Namespace == NamespaceHTML && node.Data == "title" {
			title = node.TextContent()
			return false
		}
		return true
	})
	return title
}

func (document *Document) firstElementByLocalName(localName string) *Node {
	for _, child := range document.children {
		if child.Type != ElementNode {
			continue
		}
		if child.Namespace == NamespaceHTML && child.Data == localName {
			return child
		}
		for _, grandchild := range child.children {
			if grandchild.Type == ElementNode && grandchild.Namespace == NamespaceHTML && grandchild.Data == localName {
				return grandchild
			}
		}
	}
	return nil
}

// OriginFromParts builds a tuple origin. It is primarily used by tests and
// sandbox construction.
func OriginFromParts(scheme, host, port string) urls.Origin {
	return urls.Origin{Scheme: scheme, Host: host, Port: port}
}

// SameOrigin reports whether another document shares this document's origin.
func (document *Document) SameOrigin(other *Document) bool {
	if other == nil {
		return false
	}
	if document.Origin.Opaque || other.Origin.Opaque {
		return false
	}
	return document.Origin == other.Origin
}

// QuerySelector returns the first descendant matching a simple selector list.
func (document *Document) QuerySelector(selector string) *Node {
	return document.Node.querySelector(selector)
}

// QuerySelectorAll returns all descendants matching a simple selector list.
func (document *Document) QuerySelectorAll(selector string) []*Node {
	return document.Node.querySelectorAll(selector)
}
