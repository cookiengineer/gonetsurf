package html

import (
	"errors"
	"strings"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"gonetsurf/dom"
)

// ErrNodeBudget is returned when a document exceeds its node budget.
var ErrNodeBudget = errors.New("html: node budget exceeded")

type converter struct {
	document *dom.Document
	budget   int
	count    int
}

func (converter *converter) convert(node *xhtml.Node) (*dom.Node, error) {
	converter.count++
	if converter.count > converter.budget {
		return nil, ErrNodeBudget
	}

	switch node.Type {
	case xhtml.ElementNode:
		namespace := toNamespace(node.Namespace)
		element := converter.document.CreateElementNS(namespace, node.Data)
		for _, attribute := range node.Attr {
			element.Attr = append(element.Attr, dom.Attribute{
				Namespace: attribute.Namespace,
				Name:      attribute.Key,
				Value:     attribute.Val,
			})
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			converted, err := converter.convert(child)
			if err != nil {
				return nil, err
			}
			if converted != nil {
				element.AppendChild(converted)
			}
		}
		if namespace == dom.NamespaceHTML && node.Data == "template" {
			contents := converter.document.CreateDocumentFragment()
			for _, child := range append([]*dom.Node(nil), element.Children()...) {
				contents.AppendChild(child)
			}
			element.SetTemplateContents(contents)
		}
		return element, nil

	case xhtml.TextNode:
		return converter.document.CreateTextNode(node.Data), nil

	case xhtml.CommentNode:
		return converter.document.CreateComment(node.Data), nil

	case xhtml.DoctypeNode:
		publicID := ""
		systemID := ""
		for _, attribute := range node.Attr {
			switch attribute.Key {
			case "public":
				publicID = attribute.Val
			case "system":
				systemID = attribute.Val
			}
		}
		return converter.document.CreateDocumentType(node.Data, publicID, systemID), nil

	default:
		return nil, nil
	}
}

// toNamespace maps an x/net/html namespace to a DOM namespace.
func toNamespace(namespace string) string {
	switch namespace {
	case "", "html":
		return dom.NamespaceHTML
	case "svg":
		return dom.NamespaceSVG
	case "math":
		return dom.NamespaceMathML
	default:
		return namespace
	}
}

// fromNamespace maps a DOM namespace back to the x/net/html representation.
func fromNamespace(namespace string) string {
	switch namespace {
	case dom.NamespaceHTML:
		return ""
	case dom.NamespaceSVG:
		return "svg"
	case dom.NamespaceMathML:
		return "math"
	default:
		return namespace
	}
}

// toXHTML converts a DOM node back to an x/net/html node for serialization.
func toXHTML(node *dom.Node) *xhtml.Node {
	if node == nil {
		return nil
	}
	converted := &xhtml.Node{Data: node.Data, Namespace: fromNamespace(node.Namespace)}
	switch node.Type {
	case dom.DocumentNode:
		converted.Type = xhtml.DocumentNode
	case dom.DocumentTypeNode:
		converted.Type = xhtml.DoctypeNode
	case dom.ElementNode:
		converted.Type = xhtml.ElementNode
		converted.DataAtom = atom.Lookup([]byte(node.Data))
	case dom.TextNode:
		converted.Type = xhtml.TextNode
	case dom.CommentNode:
		converted.Type = xhtml.CommentNode
	case dom.DocumentFragmentNode:
		converted.Type = xhtml.DocumentNode
	default:
		return nil
	}
	for _, attribute := range node.Attr {
		converted.Attr = append(converted.Attr, xhtml.Attribute{
			Namespace: attribute.Namespace,
			Key:       attribute.Name,
			Val:       attribute.Value,
		})
	}
	for _, child := range node.Children() {
		if childNode := toXHTML(child); childNode != nil {
			converted.AppendChild(childNode)
		}
	}
	return converted
}

// Serialize returns the serialized children of a node.
func Serialize(node *dom.Node) string {
	var builder strings.Builder
	for _, child := range node.Children() {
		if converted := toXHTML(child); converted != nil {
			_ = xhtml.Render(&builder, converted)
		}
	}
	return builder.String()
}
