// Package dom implements the browser's document object model: a mutable,
// namespace-aware tree built from the golang.org/x/net/html parser.
//
// See specifications/web/dom.html and specifications/html/dom.html.
package dom

// Namespace identifiers used by the DOM.
const (
	NamespaceHTML   = "http://www.w3.org/1999/xhtml"
	NamespaceSVG    = "http://www.w3.org/2000/svg"
	NamespaceMathML = "http://www.w3.org/1998/Math/MathML"
	NamespaceXLink  = "http://www.w3.org/1999/xlink"
	NamespaceXML    = "http://www.w3.org/XML/1998/namespace"
	NamespaceXMLNS  = "http://www.w3.org/2000/xmlns/"
)
