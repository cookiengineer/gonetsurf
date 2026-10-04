package dom

import "strings"

// NodeType identifies the kind of a node.
type NodeType uint8

// Node types.
const (
	DocumentNode NodeType = iota
	DocumentTypeNode
	ElementNode
	TextNode
	CommentNode
	DocumentFragmentNode
	ProcessingInstructionNode
)

// Attribute is an element attribute, qualified name, and value.
type Attribute struct {
	Namespace string
	Prefix    string
	Name      string
	Value     string
}

// Node is a node in the document tree.
type Node struct {
	Type      NodeType
	Data      string
	Namespace string
	Attr      []Attribute

	// PublicID and SystemID are set on document type nodes.
	PublicID string
	SystemID string

	parent   *Node
	children []*Node
	owner    *Document

	templateContents *Node
	shadowRoot       *Node
}

// NewElement returns a new element in the given namespace.
func NewElement(namespace, localName string) *Node {
	return &Node{Type: ElementNode, Namespace: namespace, Data: localName}
}

// NewText returns a new text node.
func NewText(data string) *Node {
	return &Node{Type: TextNode, Data: data}
}

// NewComment returns a new comment node.
func NewComment(data string) *Node {
	return &Node{Type: CommentNode, Data: data}
}

// NewDocumentFragment returns a new document fragment node.
func NewDocumentFragment() *Node {
	return &Node{Type: DocumentFragmentNode}
}

// Parent returns the node's parent, or nil.
func (node *Node) Parent() *Node {
	return node.parent
}

// Children returns the node's child nodes.
func (node *Node) Children() []*Node {
	return node.children
}

// FirstChild returns the first child, or nil.
func (node *Node) FirstChild() *Node {
	if len(node.children) == 0 {
		return nil
	}
	return node.children[0]
}

// LastChild returns the last child, or nil.
func (node *Node) LastChild() *Node {
	if len(node.children) == 0 {
		return nil
	}
	return node.children[len(node.children)-1]
}

// NextSibling returns the next sibling, or nil.
func (node *Node) NextSibling() *Node {
	if node.parent == nil {
		return nil
	}
	siblings := node.parent.children
	for index, sibling := range siblings {
		if sibling == node && index+1 < len(siblings) {
			return siblings[index+1]
		}
	}
	return nil
}

// PreviousSibling returns the previous sibling, or nil.
func (node *Node) PreviousSibling() *Node {
	if node.parent == nil {
		return nil
	}
	siblings := node.parent.children
	for index, sibling := range siblings {
		if sibling == node && index > 0 {
			return siblings[index-1]
		}
	}
	return nil
}

// OwnerDocument returns the document that owns the node.
func (node *Node) OwnerDocument() *Document {
	return node.owner
}

// ParentElement returns the parent when it is an element, or nil.
func (node *Node) ParentElement() *Node {
	if node.parent != nil && node.parent.Type == ElementNode {
		return node.parent
	}
	return nil
}

// IsConnected reports whether the node is connected to a document tree.
func (node *Node) IsConnected() bool {
	current := node
	for current != nil {
		if current.Type == DocumentNode {
			return true
		}
		current = current.parent
	}
	return false
}

// AppendChild appends child, detaching it from its previous parent.
func (node *Node) AppendChild(child *Node) {
	node.InsertBefore(child, nil)
}

// InsertBefore inserts child before reference. A nil reference appends.
func (node *Node) InsertBefore(child, reference *Node) {
	if child == nil || child == node {
		return
	}
	if child.parent != nil {
		child.parent.removeChild(child)
	}
	child.parent = node
	child.setOwner(node.owner)
	if reference == nil {
		node.children = append(node.children, child)
		return
	}
	for index, existing := range node.children {
		if existing == reference {
			node.children = append(node.children, nil)
			copy(node.children[index+1:], node.children[index:])
			node.children[index] = child
			return
		}
	}
	node.children = append(node.children, child)
}

// RemoveChild removes a direct child.
func (node *Node) RemoveChild(child *Node) {
	if child == nil || child.parent != node {
		return
	}
	node.removeChild(child)
}

func (node *Node) removeChild(child *Node) {
	for index, existing := range node.children {
		if existing == child {
			node.children = append(node.children[:index], node.children[index+1:]...)
			break
		}
	}
	child.parent = nil
	child.setOwner(nil)
	if node.owner != nil {
		node.owner.markDirty()
	}
}

// ReplaceChild replaces oldChild with newChild.
func (node *Node) ReplaceChild(newChild, oldChild *Node) {
	if oldChild == nil || oldChild.parent != node {
		return
	}
	node.InsertBefore(newChild, oldChild)
	node.RemoveChild(oldChild)
}

// CloneNode returns a copy of the node. When deep is true, descendants are
// copied as well.
func (node *Node) CloneNode(deep bool) *Node {
	clone := &Node{
		Type:      node.Type,
		Data:      node.Data,
		Namespace: node.Namespace,
	}
	if len(node.Attr) > 0 {
		clone.Attr = append([]Attribute(nil), node.Attr...)
	}
	if deep {
		for _, child := range node.children {
			clone.AppendChild(child.CloneNode(true))
		}
	}
	return clone
}

func (node *Node) setOwner(owner *Document) {
	node.owner = owner
	for _, child := range node.children {
		child.setOwner(owner)
	}
	if node.templateContents != nil {
		node.templateContents.setOwner(owner)
	}
	if node.shadowRoot != nil {
		node.shadowRoot.setOwner(owner)
	}
}

// Descendants calls visit for every descendant in tree order. Returning false
// stops the traversal.
func (node *Node) Descendants(visit func(*Node) bool) {
	for _, child := range node.children {
		if !visit(child) {
			return
		}
		child.Descendants(visit)
	}
}

// Ancestors calls visit for every ancestor from the parent upward.
func (node *Node) Ancestors(visit func(*Node) bool) {
	for current := node.parent; current != nil; current = current.parent {
		if !visit(current) {
			return
		}
	}
}

// TextContent returns the concatenated text of the node and its descendants.
// Comment nodes are excluded, matching the DOM.
func (node *Node) TextContent() string {
	switch node.Type {
	case TextNode:
		return node.Data
	case CommentNode, ProcessingInstructionNode:
		return ""
	default:
		var builder strings.Builder
		var collect func(*Node)
		collect = func(current *Node) {
			if current.Type == TextNode {
				builder.WriteString(current.Data)
				return
			}
			for _, child := range current.children {
				collect(child)
			}
		}
		for _, child := range node.children {
			collect(child)
		}
		return builder.String()
	}
}

// SetTextContent replaces all children with a single text node.
func (node *Node) SetTextContent(text string) {
	node.RemoveChildren()
	if text != "" {
		node.AppendChild(NewText(text))
	}
}

// RemoveChildren removes all child nodes.
func (node *Node) RemoveChildren() {
	for _, child := range node.children {
		child.parent = nil
		child.setOwner(nil)
	}
	node.children = nil
	if node.owner != nil {
		node.owner.markDirty()
	}
}

// TemplateContents returns the template element's content fragment.
func (node *Node) TemplateContents() *Node {
	return node.templateContents
}

// SetTemplateContents sets the template element's content fragment.
func (node *Node) SetTemplateContents(fragment *Node) {
	node.templateContents = fragment
	fragment.parent = node
	fragment.setOwner(node.owner)
}

// ShadowRoot returns the node's shadow root, if one is attached.
func (node *Node) ShadowRoot() *Node {
	return node.shadowRoot
}

// AttachShadow attaches a new empty shadow root (scaffolding).
func (node *Node) AttachShadow() *Node {
	if node.shadowRoot != nil {
		return node.shadowRoot
	}
	shadow := NewDocumentFragment()
	node.shadowRoot = shadow
	shadow.parent = node
	shadow.setOwner(node.owner)
	return shadow
}

// LocalName returns the element's local name.
func (node *Node) LocalName() string {
	return node.Data
}

// TagName returns the element's qualified name. For HTML elements this is
// uppercased, matching the DOM.
func (node *Node) TagName() string {
	if node.Namespace == NamespaceHTML {
		return strings.ToUpper(node.Data)
	}
	return node.Data
}
