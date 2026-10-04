package html

import (
	"strings"

	xhtml "golang.org/x/net/html"
	"gonetsurf/dom"
)

// quirksPublicIDPrefixes are public identifier prefixes that force quirks mode.
//
// See the HTML Standard's quirks-mode list.
var quirksPublicIDPrefixes = []string{
	"+//silmaril//dtd html pro v0r11 19970101//",
	"-//advasoft ltd//dtd html 3.0 aswedit + extensions//",
	"-//as//dtd html 3.0 aswedit + extensions//",
	"-//ietf//dtd html 2.0 level 1//",
	"-//ietf//dtd html 2.0 level 2//",
	"-//ietf//dtd html 2.0 strict level 1//",
	"-//ietf//dtd html 2.0 strict level 2//",
	"-//ietf//dtd html 2.0 strict//",
	"-//ietf//dtd html 2.0//",
	"-//ietf//dtd html 2.1e//",
	"-//ietf//dtd html 3.0//",
	"-//ietf//dtd html 3.2 final//",
	"-//ietf//dtd html 3.2//",
	"-//ietf//dtd html 3//",
	"-//ietf//dtd html level 0//",
	"-//ietf//dtd html level 1//",
	"-//ietf//dtd html level 2//",
	"-//ietf//dtd html level 3//",
	"-//ietf//dtd html strict level 0//",
	"-//ietf//dtd html strict level 1//",
	"-//ietf//dtd html strict level 2//",
	"-//ietf//dtd html strict level 3//",
	"-//ietf//dtd html strict//",
	"-//ietf//dtd html//",
	"-//metrius//dtd metrius presentational//",
	"-//microsoft//dtd internet explorer 2.0 html strict//",
	"-//microsoft//dtd internet explorer 2.0 html//",
	"-//microsoft//dtd internet explorer 3.0 html strict//",
	"-//microsoft//dtd internet explorer 3.0 html//",
	"-//netscape comm. corp.//dtd html//",
	"-//netscape comm. corp.//dtd strict html//",
	"-//o'reilly and associates//dtd html 2.0//",
	"-//o'reilly and associates//dtd html extended 1.0//",
	"-//o'reilly and associates//dtd html extended relaxed 1.0//",
	"-//softquad software//dtd hotmetal pro 6.0::19990601::extensions to html 4.0//",
	"-//softquad//dtd hotmetal pro 4.0::19971010::extensions to html 4.0//",
	"-//spyglass//dtd html 2.0 extended//",
	"-//sq//dtd html 2.0 hotmetal + extensions//",
	"-//sun microsystems corp.//dtd hotjava html//",
	"-//sun microsystems corp.//dtd hotjava strict html//",
	"-//w3c//dtd html 3 1995-03-24//",
	"-//w3c//dtd html 3.2 draft//",
	"-//w3c//dtd html 3.2 final//",
	"-//w3c//dtd html 3.2//",
	"-//w3c//dtd html 3.2s draft//",
	"-//w3c//dtd html 4.0 frameset//",
	"-//w3c//dtd html 4.0 transitional//",
	"-//w3c//dtd html experimental 19960712//",
	"-//w3c//dtd html experimental 970421//",
	"-//w3c//dtd w3 html//",
	"-//w3o//dtd w3 html 3.0//",
	"-//webtechs//dtd mozilla html 2.0//",
	"-//webtechs//dtd mozilla html//",
}

// DetermineQuirks classifies the document's parsing mode from its doctype.
func DetermineQuirks(root *xhtml.Node) dom.QuirksMode {
	var doctype *xhtml.Node
	for child := root.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == xhtml.DoctypeNode {
			doctype = child
			break
		}
	}
	if doctype == nil {
		return dom.Quirks
	}
	name := strings.ToLower(doctype.Data)
	if name != "html" {
		return dom.Quirks
	}
	publicID := ""
	systemID := ""
	for _, attribute := range doctype.Attr {
		switch strings.ToLower(attribute.Key) {
		case "public":
			publicID = strings.ToLower(attribute.Val)
		case "system":
			systemID = strings.ToLower(attribute.Val)
		}
	}
	for _, prefix := range quirksPublicIDPrefixes {
		if strings.HasPrefix(publicID, prefix) {
			return dom.Quirks
		}
	}
	switch {
	case strings.HasPrefix(publicID, "-//w3c//dtd html 4.01 frameset//"),
		strings.HasPrefix(publicID, "-//w3c//dtd html 4.01 transitional//"):
		if systemID == "" {
			return dom.Quirks
		}
		return dom.LimitedQuirks
	case strings.HasPrefix(publicID, "-//w3c//dtd xhtml 1.0 frameset//"),
		strings.HasPrefix(publicID, "-//w3c//dtd xhtml 1.0 transitional//"):
		return dom.LimitedQuirks
	}
	return dom.NoQuirks
}
