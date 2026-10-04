package dom

import "errors"

// ErrNoParser is returned when fragment parsing is used before parsers/html has
// registered itself.
var ErrNoParser = errors.New("dom: no fragment parser registered")

// FragmentParser parses an HTML fragment in the context of a node and returns
// the resulting child nodes. It is assigned by parsers/html to avoid an import
// cycle.
var FragmentParser func(context *Node, source string) ([]*Node, error)

// Serializer serializes a node's children to HTML. It is assigned by
// parsers/html.
var Serializer func(node *Node) string

// InnerHTML returns the serialized children of the node.
func (node *Node) InnerHTML() string {
	if Serializer == nil {
		return ""
	}
	return Serializer(node)
}

// SetInnerHTML replaces the node's children with a parsed HTML fragment.
func (node *Node) SetInnerHTML(source string) error {
	if FragmentParser == nil {
		return ErrNoParser
	}
	children, err := FragmentParser(node, source)
	if err != nil {
		return err
	}
	node.RemoveChildren()
	for _, child := range children {
		node.AppendChild(child)
	}
	return nil
}
