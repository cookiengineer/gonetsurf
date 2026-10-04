// Package mime implements MIME header parsing needed by the browser, with a
// focus on safe filename handling.
package mime

import (
	"mime"
	"path"
	"strings"
)

// reservedNames lists Windows device names that must never be used as a file
// name.
var reservedNames = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true,
	"COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true,
	"LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// ParseContentDisposition parses a Content-Disposition header and returns the
// lowercase disposition type and a sanitized filename.
//
// See RFC 6266 and RFC 5987.
func ParseContentDisposition(value string) (disposition string, filename string, err error) {
	mediaType, parameters, err := mime.ParseMediaType(value)
	if err != nil {
		return "", "", err
	}
	disposition = strings.ToLower(strings.TrimSpace(mediaType))
	filename = parameters["filename*"]
	if filename == "" {
		filename = parameters["filename"]
	}
	return disposition, SanitizeFilename(filename), nil
}

// SanitizeFilename extracts the final path segment and removes characters that
// are unsafe in a filesystem name. It returns the empty string when no safe
// name remains.
func SanitizeFilename(input string) string {
	if input == "" {
		return ""
	}
	input = strings.ReplaceAll(input, "\\", "/")
	name := path.Base(input)
	name = strings.Map(func(character rune) rune {
		if character < 0x20 || character == 0x7F {
			return -1
		}
		return character
	}, name)
	name = strings.Trim(name, " .")
	if name == "" || name == "." || name == ".." {
		return ""
	}
	upper := strings.ToUpper(name)
	if dot := strings.IndexByte(upper, '.'); dot >= 0 {
		upper = upper[:dot]
	}
	if reservedNames[upper] {
		return ""
	}
	return name
}
