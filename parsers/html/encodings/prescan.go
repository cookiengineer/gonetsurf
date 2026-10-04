package encodings

import (
	"bytes"
	"strings"
)

// Prescan implements the HTML Standard's prescan a byte stream to determine its
// encoding. It inspects the first 1024 bytes for a <meta> charset declaration.
func Prescan(data []byte) string {
	if len(data) > sniffByteCount {
		data = data[:sniffByteCount]
	}
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
		if hasPrefixFold(data[position:], "<meta") {
			attributes, next, ok := parseTagAttributes(data, position+5)
			if ok {
				if label := charsetFromMetaAttributes(attributes); label != "" {
					return label
				}
				position = next
				continue
			}
		}
		closing := bytes.IndexByte(data[position:], '>')
		if closing < 0 {
			return ""
		}
		position += closing + 1
	}
	return ""
}

func charsetFromMetaAttributes(attributes map[string]string) string {
	if charset, ok := attributes["charset"]; ok && charset != "" {
		return charset
	}
	httpEquiv := strings.ToLower(strings.TrimSpace(attributes["http-equiv"]))
	if httpEquiv == "content-type" {
		if content, ok := attributes["content"]; ok {
			return charsetFromContent(content)
		}
	}
	if content, ok := attributes["content"]; ok {
		if label := charsetFromContent(content); label != "" {
			return label
		}
	}
	return ""
}

func charsetFromContent(content string) string {
	lower := strings.ToLower(content)
	index := strings.Index(lower, "charset")
	if index < 0 {
		return ""
	}
	remainder := content[index+len("charset"):]
	remainder = strings.TrimLeft(remainder, " \t\n\r\f")
	if remainder == "" || remainder[0] != '=' {
		return ""
	}
	remainder = strings.TrimLeft(remainder[1:], " \t\n\r\f")
	if remainder == "" {
		return ""
	}
	if remainder[0] == '"' || remainder[0] == '\'' {
		quote := remainder[0]
		end := strings.IndexByte(remainder[1:], quote)
		if end < 0 {
			return ""
		}
		return remainder[1 : 1+end]
	}
	end := strings.IndexAny(remainder, " \t\n\r\f;")
	if end < 0 {
		return remainder
	}
	return remainder[:end]
}

// parseTagAttributes parses attributes from start until a '>' or EOF. It
// returns the attribute map, the position after the tag, and whether a closing
// '>' was found.
func parseTagAttributes(data []byte, start int) (map[string]string, int, bool) {
	attributes := make(map[string]string)
	position := start
	for position < len(data) {
		// Skip whitespace and stray slashes.
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
		name := strings.ToLower(string(data[nameStart:position]))
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
				value = string(data[valueStart:position])
				if position < len(data) {
					position++
				}
			} else {
				valueStart := position
				for position < len(data) && !isASCIISpace(data[position]) && data[position] != '>' {
					position++
				}
				value = string(data[valueStart:position])
			}
		}
		if _, exists := attributes[name]; !exists {
			attributes[name] = value
		}
	}
	return attributes, position, false
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
