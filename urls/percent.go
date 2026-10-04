package urls

import (
	"strings"
	"unicode/utf8"
)

const upperHexDigits = "0123456789ABCDEF"

// percentEncodeSet reports whether a code point must be percent-encoded for a
// particular URL component.
type percentEncodeSet func(character rune) bool

func inC0ControlSet(character rune) bool {
	return character <= 0x1F || character > 0x7E
}

func inFragmentSet(character rune) bool {
	return inC0ControlSet(character) || character == ' ' || character == '"' ||
		character == '<' || character == '>' || character == '`'
}

func inQuerySet(character rune) bool {
	return inC0ControlSet(character) || character == ' ' || character == '"' ||
		character == '#' || character == '<' || character == '>'
}

func inSpecialQuerySet(character rune) bool {
	return inQuerySet(character) || character == '\''
}

func inPathSet(character rune) bool {
	return inQuerySet(character) || character == '?' || character == '^' ||
		character == '`' || character == '{' || character == '}'
}

func inUserinfoSet(character rune) bool {
	return inPathSet(character) || character == '/' || character == ':' ||
		character == ';' || character == '=' || character == '@' ||
		character == '[' || character == '\\' || character == ']' ||
		character == '^' || character == '|'
}

func inComponentSet(character rune) bool {
	return inUserinfoSet(character) || character == '$' || character == '%' ||
		character == '&' || character == '+' || character == ','
}

// percentEncode UTF-8 percent-encodes every rune matched by the set.
func percentEncode(input string, set percentEncodeSet) string {
	var builder strings.Builder
	var encoded [utf8.UTFMax]byte
	for _, character := range input {
		if !set(character) {
			builder.WriteRune(character)
			continue
		}
		count := utf8.EncodeRune(encoded[:], character)
		for _, octet := range encoded[:count] {
			builder.WriteByte('%')
			builder.WriteByte(upperHexDigits[octet>>4])
			builder.WriteByte(upperHexDigits[octet&0x0F])
		}
	}
	return builder.String()
}

// percentDecode decodes %XX sequences, leaving malformed sequences untouched.
func percentDecode(input string) []byte {
	output := make([]byte, 0, len(input))
	for index := 0; index < len(input); index++ {
		if input[index] == '%' && index+2 < len(input) &&
			isASCIIHexDigit(rune(input[index+1])) && isASCIIHexDigit(rune(input[index+2])) {
			output = append(output, byte(hexValue(rune(input[index+1]))<<4|hexValue(rune(input[index+2]))))
			index += 2
			continue
		}
		output = append(output, input[index])
	}
	return output
}

func isASCIIHexDigit(character rune) bool {
	return isASCIIDigit(character) ||
		(character >= 'a' && character <= 'f') ||
		(character >= 'A' && character <= 'F')
}

func hexValue(character rune) int {
	switch {
	case character >= '0' && character <= '9':
		return int(character - '0')
	case character >= 'a' && character <= 'f':
		return int(character-'a') + 10
	case character >= 'A' && character <= 'F':
		return int(character-'A') + 10
	default:
		return 0
	}
}
