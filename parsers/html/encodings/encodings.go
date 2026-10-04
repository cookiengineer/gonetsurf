// Package encodings implements the HTML Standard's encoding sniffing algorithm
// and byte-to-UTF-8 decoding. It handles BOM sniffing, transport charset
// labels, and the 1024-byte meta prescan.
//
// See specifications/html/parsing.html §13.2.3 and the Encoding Standard
// (specifications/web/encoding.html).
package encodings

import (
	"mime"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/htmlindex"
)

// SniffResult is the outcome of encoding sniffing.
type SniffResult struct {
	// Name is the canonical encoding name.
	Name string
	// Certain reports whether the encoding was determined with certainty.
	Certain bool

	coding encoding.Encoding
}

// Encoding returns the encoding. It is never nil for a result returned by
// Sniff.
func (result SniffResult) Encoding() encoding.Encoding {
	return result.coding
}

// Sniff implements the first pass of the encoding sniffing algorithm.
func Sniff(contentType string, data []byte) SniffResult {
	if bomName := sniffBOM(data); bomName != "" {
		if coding := lookup(bomName); coding != nil {
			return SniffResult{Name: canonicalName(coding), Certain: true, coding: coding}
		}
	}
	if label := charsetFromContentType(contentType); label != "" {
		if coding := lookup(label); coding != nil {
			return SniffResult{Name: canonicalName(coding), Certain: true, coding: coding}
		}
	}
	if name := Prescan(data); name != "" {
		if coding := lookup(name); coding != nil {
			return SniffResult{Name: name, coding: coding}
		}
	}
	coding := lookup("windows-1252")
	return SniffResult{Name: "windows-1252", coding: coding}
}

// Decode converts data to UTF-8 using the sniffed encoding.
func (result SniffResult) Decode(data []byte) ([]byte, error) {
	if result.coding == nil || result.coding == encoding.Nop || result.Name == "utf-8" {
		return data, nil
	}
	return result.coding.NewDecoder().Bytes(data)
}

// ApplyMetaChange returns a new result when a mid-stream <meta> declares a
// different, valid encoding. The boolean reports whether the encoding changed.
//
// See specifications/html/parsing.html "changing the encoding".
func (result SniffResult) ApplyMetaChange(label string) (SniffResult, bool) {
	coding := lookup(label)
	if coding == nil {
		return result, false
	}
	name := canonicalName(coding)
	if name == result.Name {
		return result, false
	}
	return SniffResult{Name: name, Certain: true, coding: coding}, true
}

func sniffBOM(data []byte) string {
	switch {
	case len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF:
		return "utf-8"
	case len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF:
		return "utf-16be"
	case len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE:
		return "utf-16le"
	default:
		return ""
	}
}

func charsetFromContentType(contentType string) string {
	if contentType == "" {
		return ""
	}
	_, parameters, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ""
	}
	return parameters["charset"]
}

func lookup(label string) encoding.Encoding {
	coding, err := htmlindex.Get(strings.TrimSpace(label))
	if err != nil {
		return nil
	}
	return coding
}

func canonicalName(coding encoding.Encoding) string {
	name, err := htmlindex.Name(coding)
	if err != nil || name == "" {
		return "utf-8"
	}
	return name
}
