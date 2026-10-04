// Package http1 implements an HTTP/1.1 client with strict, privacy-aware
// framing. See specifications/http/rfc7230.txt through rfc7234.txt.
package http1

import "strings"

// Field is a single header field with its original casing and order.
type Field struct {
	Name  string
	Value string
}

// Header is an ordered list of header fields. Order and casing are preserved
// so the transport can control fingerprinting.
type Header []Field

// Get returns the first value of the named field, case-insensitively.
func (header Header) Get(name string) string {
	for _, field := range header {
		if strings.EqualFold(field.Name, name) {
			return field.Value
		}
	}
	return ""
}

// Values returns all values of the named field, case-insensitively.
func (header Header) Values(name string) []string {
	var values []string
	for _, field := range header {
		if strings.EqualFold(field.Name, name) {
			values = append(values, field.Value)
		}
	}
	return values
}

// Has reports whether the named field is present.
func (header Header) Has(name string) bool {
	for _, field := range header {
		if strings.EqualFold(field.Name, name) {
			return true
		}
	}
	return false
}

// Add appends a field.
func (header *Header) Add(name, value string) {
	*header = append(*header, Field{Name: name, Value: value})
}

// Set replaces all occurrences of name with a single field, or appends it.
func (header *Header) Set(name, value string) {
	replaced := false
	result := (*header)[:0]
	for _, field := range *header {
		if strings.EqualFold(field.Name, name) {
			if !replaced {
				result = append(result, Field{Name: name, Value: value})
				replaced = true
			}
			continue
		}
		result = append(result, field)
	}
	if !replaced {
		result = append(result, Field{Name: name, Value: value})
	}
	*header = result
}

// Del removes all occurrences of name.
func (header *Header) Del(name string) {
	result := (*header)[:0]
	for _, field := range *header {
		if !strings.EqualFold(field.Name, name) {
			result = append(result, field)
		}
	}
	*header = result
}

// Clone returns a deep copy of the header.
func (header Header) Clone() Header {
	clone := make(Header, len(header))
	copy(clone, header)
	return clone
}

func isToken(value string) bool {
	if value == "" {
		return false
	}
	for index := 0; index < len(value); index++ {
		if !isTokenCharacter(value[index]) {
			return false
		}
	}
	return true
}

func isTokenCharacter(value byte) bool {
	switch {
	case value >= 'a' && value <= 'z',
		value >= 'A' && value <= 'Z',
		value >= '0' && value <= '9':
		return true
	}
	switch value {
	case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
		return true
	}
	return false
}
