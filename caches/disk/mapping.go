// Package disk maps URLs to a human-readable, invertible filesystem mirror and
// maintains a journal of URL metadata. It implements the layout described in
// docs/cache.md.
package disk

import (
	"fmt"
	"strings"

	"gonetsurf/urls"
)

const (
	querySentinel    = "~q~"
	emptySegment     = "~empty~"
	schemeSentinel   = "~"
	indexName        = "index.html"
	maxSegmentBytes  = 240
	hashSuffixLength = 8
)

// Layout maps URLs into a root directory.
type Layout struct {
	Root string
}

// NewLayout returns a Layout rooted at root.
func NewLayout(root string) Layout {
	return Layout{Root: root}
}

// RelativePath returns the path of the URL relative to the layout root.
func (layout Layout) RelativePath(url *urls.URL) string {
	segments := []string{authoritySegment(url)}
	if url.CannotBeABase {
		segments = append(segments, escapeSegment("~opaque~"), escapeSegment(url.OpaquePath))
	} else {
		pathSegments := url.Path
		if len(pathSegments) > 0 && pathSegments[len(pathSegments)-1] == "" {
			pathSegments = pathSegments[:len(pathSegments)-1]
		}
		for _, segment := range pathSegments {
			segments = append(segments, escapeSegment(segment))
		}
	}

	if url.HasQuery {
		segments = append(segments, querySentinel)
		for _, part := range strings.Split(url.Query, "/") {
			segments = append(segments, escapeSegment(part))
		}
		segments = append(segments, indexName)
	} else if isDirectoryPath(url) {
		segments = append(segments, indexName)
	}
	return strings.Join(segments, "/")
}

// Path returns the absolute path of the URL within the layout root.
func (layout Layout) Path(url *urls.URL) string {
	return strings.TrimRight(layout.Root, "/") + "/" + layout.RelativePath(url)
}

// Directory returns the directory that contains the URL's mapped file.
func (layout Layout) Directory(url *urls.URL) string {
	relative := layout.RelativePath(url)
	if index := strings.LastIndexByte(relative, '/'); index >= 0 {
		return strings.TrimRight(layout.Root, "/") + "/" + relative[:index]
	}
	return strings.TrimRight(layout.Root, "/")
}

// URLFor reconstructs a URL from a relative path. It is the inverse of
// RelativePath for paths that do not contain a literal index.html disambiguated
// by the journal.
func (layout Layout) URLFor(relativePath string) (*urls.URL, error) {
	segments := strings.Split(strings.Trim(relativePath, "/"), "/")
	if len(segments) == 0 || segments[0] == "" {
		return nil, fmt.Errorf("disk: empty relative path")
	}
	scheme, host, port, err := parseAuthoritySegment(segments[0])
	if err != nil {
		return nil, err
	}
	rest := segments[1:]

	queryIndex := -1
	for index, segment := range rest {
		if segment == querySentinel {
			queryIndex = index
			break
		}
	}

	var pathSegments []string
	hasQuery := false
	query := ""
	isDirectory := false
	if queryIndex >= 0 {
		pathSegments = rest[:queryIndex]
		querySegments := rest[queryIndex+1:]
		if len(querySegments) > 0 && querySegments[len(querySegments)-1] == indexName {
			querySegments = querySegments[:len(querySegments)-1]
		}
		decoded := make([]string, 0, len(querySegments))
		for _, segment := range querySegments {
			decoded = append(decoded, unescapeSegment(segment))
		}
		query = strings.Join(decoded, "/")
		hasQuery = true
	} else {
		pathSegments = rest
		if len(pathSegments) > 0 && pathSegments[len(pathSegments)-1] == indexName {
			pathSegments = pathSegments[:len(pathSegments)-1]
			isDirectory = true
		}
	}

	decodedPath := make([]string, 0, len(pathSegments))
	for _, segment := range pathSegments {
		decodedPath = append(decodedPath, unescapeSegment(segment))
	}

	var builder strings.Builder
	builder.WriteString(scheme)
	builder.WriteString("://")
	if port != "" {
		builder.WriteString(host)
		builder.WriteString(":")
		builder.WriteString(port)
	} else {
		builder.WriteString(host)
	}
	builder.WriteByte('/')
	if len(decodedPath) > 0 {
		builder.WriteString(strings.Join(decodedPath, "/"))
		if isDirectory {
			builder.WriteByte('/')
		}
	}
	if hasQuery {
		builder.WriteByte('?')
		builder.WriteString(query)
	}
	return urls.Parse(builder.String())
}

func authoritySegment(url *urls.URL) string {
	host := url.Host
	if url.Port != "" {
		host += "~" + url.Port
	}
	switch url.Scheme {
	case "https":
		return host
	case "http":
		return schemeSentinel + "http" + schemeSentinel + host
	default:
		return schemeSentinel + url.Scheme + schemeSentinel + host
	}
}

func parseAuthoritySegment(segment string) (scheme, host, port string, err error) {
	scheme = "https"
	remainder := segment
	if strings.HasPrefix(segment, schemeSentinel) {
		rest := segment[1:]
		end := strings.Index(rest, schemeSentinel)
		if end < 0 {
			return "", "", "", fmt.Errorf("disk: malformed authority segment %q", segment)
		}
		scheme = rest[:end]
		remainder = rest[end+1:]
	}
	host = remainder
	if index := strings.LastIndexByte(remainder, '~'); index >= 0 {
		candidate := remainder[index+1:]
		if candidate != "" && isAllDigits(candidate) {
			host = remainder[:index]
			port = candidate
		}
	}
	if host == "" {
		return "", "", "", fmt.Errorf("disk: empty host in %q", segment)
	}
	return scheme, host, port, nil
}

func isDirectoryPath(url *urls.URL) bool {
	if url.CannotBeABase {
		return false
	}
	if len(url.Path) == 0 {
		return true
	}
	return url.Path[len(url.Path)-1] == ""
}

func escapeSegment(segment string) string {
	if segment == "" {
		return emptySegment
	}
	if strings.HasPrefix(segment, "~") {
		segment = "~" + segment
	}
	if len(segment) > maxSegmentBytes {
		hash := shortHash(segment)
		segment = segment[:maxSegmentBytes-len(hash)-2] + "~" + hash
	}
	return segment
}

func unescapeSegment(segment string) string {
	if segment == emptySegment {
		return ""
	}
	if strings.HasPrefix(segment, "~~") {
		return segment[1:]
	}
	return segment
}

func shortHash(input string) string {
	var hash uint64 = 1469598103934665603
	for index := 0; index < len(input); index++ {
		hash ^= uint64(input[index])
		hash *= 1099511628211
	}
	const digits = "0123456789abcdef"
	buffer := make([]byte, hashSuffixLength)
	for index := range buffer {
		buffer[index] = digits[hash&0xF]
		hash >>= 4
	}
	return string(buffer)
}

func isAllDigits(input string) bool {
	if input == "" {
		return false
	}
	for _, character := range input {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}
