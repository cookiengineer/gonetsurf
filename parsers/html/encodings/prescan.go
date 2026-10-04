package encodings

import (
	"bytes"
	"strings"
)

// Prescan implements the HTML Standard's "prescan a byte stream to determine
// its encoding" algorithm. It returns the canonical encoding name declared by
// a <meta> element, or the empty string.
//
// See specifications/html/parsing.html §13.2.3.
func Prescan(data []byte) string {
	position := 0
	for position < len(data) {
		if data[position] != '<' {
			position++
			continue
		}
		if hasPrefixFold(data[position:], "<!--") {
			end := bytes.Index(data[position+4:], []byte("-->"))
			if end < 0 {
				return ""
			}
			position += 4 + end + 3
			continue
		}
		if isMetaTagStart(data, position) {
			attributes, next, ok := parseTagAttributes(data, position+5)
			if !ok {
				return ""
			}
			if name := metaEncoding(attributes); name != "" {
				return name
			}
			position = next
			continue
		}
		if hasPrefix(data[position:], "<!") || hasPrefix(data[position:], "<?") {
			end := bytes.IndexByte(data[position:], '>')
			if end < 0 {
				return ""
			}
			position += end + 1
			continue
		}
		if isTagStart(data, position) {
			nameStart := position + 1
			if data[nameStart] == '/' {
				nameStart++
			}
			next := nameStart
			for next < len(data) && !isASCIISpace(data[next]) && data[next] != '>' {
				next++
			}
			_, after, ok := parseTagAttributes(data, next)
			if !ok {
				return ""
			}
			position = after
			continue
		}
		position++
	}
	return ""
}

// isMetaTagStart reports whether a case-insensitive "<meta" followed by a
// space or slash begins at position.
func isMetaTagStart(data []byte, position int) bool {
	if position+5 >= len(data) {
		return false
	}
	if !hasPrefixFold(data[position:], "<meta") {
		return false
	}
	next := data[position+5]
	return next == '/' || isASCIISpace(next)
}

// isTagStart reports whether an ASCII letter or "/" followed by a letter
// begins at position.
func isTagStart(data []byte, position int) bool {
	if position+1 >= len(data) {
		return false
	}
	if isASCIIAlpha(data[position+1]) {
		return true
	}
	return data[position+1] == '/' && position+2 < len(data) && isASCIIAlpha(data[position+2])
}

func isASCIIAlpha(value byte) bool {
	return (value >= 'A' && value <= 'Z') || (value >= 'a' && value <= 'z')
}

// metaEncoding implements the "processing" step: it returns the canonical
// encoding name declared by the meta attributes, applying the UTF-16 and
// x-user-defined rewrites.
func metaEncoding(attributes map[string]string) string {
	gotPragma := false
	if value, ok := attributes["http-equiv"]; ok && value == "content-type" {
		gotPragma = true
	}

	charset := ""
	charsetSet := false
	needPragma := false
	if value, ok := attributes["content"]; ok && !charsetSet {
		if label := charsetFromContent(value); label != "" {
			charset = label
			charsetSet = true
			needPragma = true
		}
	}
	if value, ok := attributes["charset"]; ok {
		charset = value
		charsetSet = true
		needPragma = false
	}
	if !charsetSet {
		return ""
	}
	if needPragma && !gotPragma {
		return ""
	}
	coding := lookup(charset)
	if coding == nil {
		return ""
	}
	name := canonicalName(coding)
	switch name {
	case "utf-16be", "utf-16le":
		name = "utf-8"
	case "x-user-defined":
		name = "windows-1252"
	}
	return name
}

// charsetFromContent extracts a character encoding label from a Content-Type
// value as described by the HTML "algorithm for extracting a character
// encoding from a meta element".
func charsetFromContent(content string) string {
	lower := strings.ToLower(content)
	position := 0
	for {
		index := strings.Index(lower[position:], "charset")
		if index < 0 {
			return ""
		}
		position += index + len("charset")
		if position < len(content) && content[position] == '=' {
			position++
			break
		}
	}
	if position >= len(content) {
		return ""
	}
	if content[position] == '"' || content[position] == '\'' {
		quote := content[position]
		end := strings.IndexByte(content[position+1:], quote)
		if end < 0 {
			return ""
		}
		return content[position+1 : position+1+end]
	}
	end := position
	for end < len(content) && !isASCIISpace(content[end]) && content[end] != ';' {
		end++
	}
	return content[position:end]
}

// parseTagAttributes parses attributes from start until a '>' or EOF. Names
// and values are ASCII-lowercased per the specification. It returns the
// attribute map, the position after the tag, and whether a closing '>' was
// found.
func parseTagAttributes(data []byte, start int) (map[string]string, int, bool) {
	attributes := make(map[string]string)
	position := start
	for position < len(data) {
		for position < len(data) && (isASCIISpace(data[position]) || data[position] == '/') {
			position++
		}
		if position >= len(data) {
			return attributes, position, false
		}
		if data[position] == '>' {
			return attributes, position + 1, true
		}
		nameStart := position
		for position < len(data) && !isASCIISpace(data[position]) &&
			data[position] != '=' && data[position] != '>' && data[position] != '/' {
			position++
		}
		if position == nameStart {
			position++
			continue
		}
		name := lowerBytes(data[nameStart:position])
		for position < len(data) && isASCIISpace(data[position]) {
			position++
		}
		value := ""
		if position < len(data) && data[position] == '=' {
			position++
			for position < len(data) && isASCIISpace(data[position]) {
				position++
			}
			if position < len(data) && (data[position] == '"' || data[position] == '\'') {
				quote := data[position]
				position++
				valueStart := position
				for position < len(data) && data[position] != quote {
					position++
				}
				value = lowerBytes(data[valueStart:position])
				if position < len(data) {
					position++
				}
			} else {
				valueStart := position
				for position < len(data) && !isASCIISpace(data[position]) && data[position] != '>' {
					position++
				}
				value = lowerBytes(data[valueStart:position])
			}
		}
		if _, exists := attributes[name]; !exists {
			attributes[name] = value
		}
	}
	return attributes, position, false
}

func lowerBytes(value []byte) string {
	buffer := make([]byte, len(value))
	for index, current := range value {
		buffer[index] = lowerASCII(current)
	}
	return string(buffer)
}

func hasPrefix(data []byte, prefix string) bool {
	return len(data) >= len(prefix) && string(data[:len(prefix)]) == prefix
}

func hasPrefixFold(data []byte, prefix string) bool {
	if len(data) < len(prefix) {
		return false
	}
	for index := 0; index < len(prefix); index++ {
		if lowerASCII(data[index]) != prefix[index] {
			return false
		}
	}
	return true
}

func lowerASCII(value byte) byte {
	if value >= 'A' && value <= 'Z' {
		return value + 32
	}
	return value
}

func isASCIISpace(value byte) bool {
	return value == '\t' || value == '\n' || value == '\f' || value == '\r' || value == ' '
}
