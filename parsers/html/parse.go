package html

import (
	"bytes"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"gonetsurf/dom"
	"gonetsurf/parsers/html/encodings"
	"gonetsurf/urls"
)

// DefaultNodeBudget is the maximum number of DOM nodes created for a document.
const DefaultNodeBudget = 1_000_000

// ParseOptions configures document parsing.
type ParseOptions struct {
	URL         *urls.URL
	ContentType string
	NodeBudget  int
	// Scripting enables the HTML parser's scripting flag, which changes how
	// <noscript> content is parsed. It is false by default.
	Scripting bool
}

// Parse decodes and parses an HTML byte stream into a DOM document.
func Parse(data []byte, options ParseOptions) (*dom.Document, error) {
	sniffed := encodings.Sniff(options.ContentType, data)
	decoded, err := sniffed.Decode(data)
	if err != nil {
		decoded = data
	}
	root, err := xhtml.ParseWithOptions(bytes.NewReader(decoded), xhtml.ParseOptionEnableScripting(options.Scripting))
	if err != nil {
		return nil, err
	}
	return buildDocument(root, options)
}

// ParseString parses an HTML string.
func ParseString(source string, options ParseOptions) (*dom.Document, error) {
	return Parse([]byte(source), options)
}

func buildDocument(root *xhtml.Node, options ParseOptions) (*dom.Document, error) {
	document := dom.NewDocument()
	document.URL = options.URL
	document.ContentType = options.ContentType
	if options.URL != nil {
		document.Origin = options.URL.Origin()
	}
	document.Quirks = DetermineQuirks(root)
	document.ReadyState = dom.ReadyLoading

	budget := options.NodeBudget
	if budget <= 0 {
		budget = DefaultNodeBudget
	}
	nodeConverter := &converter{document: document, budget: budget}
	for child := root.FirstChild; child != nil; child = child.NextSibling {
		converted, err := nodeConverter.convert(child)
		if err != nil {
			return nil, err
		}
		if converted != nil {
			document.AppendChild(converted)
		}
	}
	document.ReadyState = dom.ReadyComplete
	return document, nil
}

// ParseFragment parses an HTML fragment in the context of a node and returns
// the resulting child nodes.
func ParseFragment(context *dom.Node, source []byte) ([]*dom.Node, error) {
	contextNode := &xhtml.Node{
		Type:      xhtml.ElementNode,
		Data:      context.Data,
		Namespace: fromNamespace(context.Namespace),
	}
	contextNode.DataAtom = atom.Lookup([]byte(context.Data))
	parsed, err := xhtml.ParseFragment(bytes.NewReader(source), contextNode)
	if err != nil {
		return nil, err
	}
	document := context.OwnerDocument()
	if document == nil {
		document = dom.NewDocument()
	}
	nodeConverter := &converter{document: document, budget: DefaultNodeBudget}
	var result []*dom.Node
	for _, node := range parsed {
		converted, err := nodeConverter.convert(node)
		if err != nil {
			return nil, err
		}
		if converted != nil {
			result = append(result, converted)
		}
	}
	return result, nil
}

func init() {
	dom.FragmentParser = func(context *dom.Node, source string) ([]*dom.Node, error) {
		return ParseFragment(context, []byte(source))
	}
	dom.Serializer = Serialize
}
