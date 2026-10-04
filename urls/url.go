package urls

import (
	"errors"
	"strconv"
	"strings"
)

// ErrInvalidURL is returned when a URL cannot be parsed.
var ErrInvalidURL = errors.New("urls: invalid URL")

const eofCodePoint rune = -1

// URL is a parsed URL record.
//
// Host is stored in serialized form (including brackets for IPv6). Port is the
// serialized decimal port, or the empty string when the port is null. Path is
// the list of path segments for hierarchical URLs; OpaquePath is used instead
// when CannotBeABase is true.
type URL struct {
	Scheme        string
	SchemeKind    SchemeKind
	Username      string
	Password      string
	HasHost       bool
	Host          string
	HostKind      HostKind
	Port          string
	Path          []string
	OpaquePath    string
	CannotBeABase bool
	Query         string
	HasQuery      bool
	Fragment      string
	HasFragment   bool
}

// Parse parses input as an absolute URL.
func Parse(input string) (*URL, error) {
	return parse(input, nil)
}

// ParseReference parses input relative to base.
func ParseReference(base *URL, input string) (*URL, error) {
	if base == nil {
		return nil, ErrInvalidURL
	}
	return parse(input, base)
}

type parserState uint8

const (
	stateSchemeStart parserState = iota
	stateScheme
	stateNoScheme
	stateSpecialRelativeOrAuthority
	statePathOrAuthority
	stateRelative
	stateRelativeSlash
	stateSpecialAuthoritySlashes
	stateSpecialAuthorityIgnoreSlashes
	stateAuthority
	stateHost
	statePort
	stateFile
	stateFileSlash
	stateFileHost
	statePathStart
	statePath
	stateOpaquePath
	stateQuery
	stateFragment
)

func parse(input string, base *URL) (*URL, error) {
	input = removeASCIITabOrNewline(trimC0ControlOrSpace(input))
	characters := []rune(input)

	url := &URL{}
	state := stateSchemeStart
	var buffer []rune
	atSignSeen := false
	insideBrackets := false
	passwordTokenSeen := false
	index := 0

	remainingStartsWith := func(prefix string) bool {
		prefixRunes := []rune(prefix)
		start := index + 1
		if start+len(prefixRunes) > len(characters) {
			return false
		}
		for offset, expected := range prefixRunes {
			if characters[start+offset] != expected {
				return false
			}
		}
		return true
	}

	for {
		hasInput := index >= 0 && index < len(characters)
		character := eofCodePoint
		if hasInput {
			character = characters[index]
		}
		delta := 0

		switch state {
		case stateSchemeStart:
			if isASCIIAlpha(character) {
				buffer = append(buffer, toASCIILower(character))
				state = stateScheme
			} else {
				state = stateNoScheme
				delta = -1
			}

		case stateScheme:
			switch {
			case isASCIIAlphanumeric(character) || character == '+' || character == '-' || character == '.':
				buffer = append(buffer, toASCIILower(character))
			case character == ':':
				url.Scheme = string(buffer)
				url.SchemeKind = classifyScheme(url.Scheme)
				buffer = buffer[:0]
				switch {
				case url.Scheme == "file":
					state = stateFile
				case url.SchemeKind.isSpecial() && base != nil && base.Scheme == url.Scheme:
					state = stateSpecialRelativeOrAuthority
				case url.SchemeKind.isSpecial():
					state = stateSpecialAuthoritySlashes
				case remainingStartsWith("/"):
					state = statePathOrAuthority
					delta = 1
				default:
					url.CannotBeABase = true
					url.OpaquePath = ""
					state = stateOpaquePath
				}
			default:
				buffer = buffer[:0]
				state = stateNoScheme
				delta = -1 - index
			}

		case stateNoScheme:
			switch {
			case base == nil || (base.CannotBeABase && character != '#'):
				return nil, ErrInvalidURL
			case base.CannotBeABase && character == '#':
				url.Scheme = base.Scheme
				url.SchemeKind = base.SchemeKind
				url.CannotBeABase = true
				url.OpaquePath = base.OpaquePath
				url.Query = base.Query
				url.HasQuery = base.HasQuery
				url.Fragment = ""
				url.HasFragment = true
				state = stateFragment
			case base.Scheme != "file":
				state = stateRelative
				delta = -1
			default:
				state = stateFile
				delta = -1
			}

		case stateSpecialRelativeOrAuthority:
			if character == '/' && remainingStartsWith("/") {
				state = stateSpecialAuthorityIgnoreSlashes
				delta = 1
			} else {
				state = stateRelative
				delta = -1
			}

		case statePathOrAuthority:
			if character == '/' {
				state = stateAuthority
			} else {
				state = statePath
				delta = -1
			}

		case stateRelative:
			url.Scheme = base.Scheme
			url.SchemeKind = base.SchemeKind
			switch {
			case character == '/':
				state = stateRelativeSlash
			case url.SchemeKind.isSpecial() && character == '\\':
				state = stateRelativeSlash
			default:
				url.Username = base.Username
				url.Password = base.Password
				url.HasHost = base.HasHost
				url.Host = base.Host
				url.HostKind = base.HostKind
				url.Port = base.Port
				url.Path = append([]string(nil), base.Path...)
				url.Query = base.Query
				url.HasQuery = base.HasQuery
				switch character {
				case '?':
					url.Query = ""
					url.HasQuery = true
					state = stateQuery
				case '#':
					url.Fragment = ""
					url.HasFragment = true
					state = stateFragment
				case eofCodePoint:
					// The URL is a clone of the base with the fragment removed.
				default:
					url.Query = ""
					url.HasQuery = false
					shortenPath(url)
					state = statePath
					delta = -1
				}
			}

		case stateRelativeSlash:
			switch {
			case url.SchemeKind.isSpecial() && (character == '/' || character == '\\'):
				state = stateSpecialAuthorityIgnoreSlashes
			case character == '/':
				state = stateAuthority
			default:
				url.Username = base.Username
				url.Password = base.Password
				url.HasHost = base.HasHost
				url.Host = base.Host
				url.HostKind = base.HostKind
				url.Port = base.Port
				state = statePath
				delta = -1
			}

		case stateSpecialAuthoritySlashes:
			if character == '/' && remainingStartsWith("/") {
				state = stateSpecialAuthorityIgnoreSlashes
				delta = 1
			} else {
				state = stateSpecialAuthorityIgnoreSlashes
				delta = -1
			}

		case stateSpecialAuthorityIgnoreSlashes:
			if character != '/' && character != '\\' {
				state = stateAuthority
				delta = -1
			}

		case stateAuthority:
			switch {
			case character == '@':
				if atSignSeen {
					buffer = append([]rune{'%', '4', '0'}, buffer...)
				}
				atSignSeen = true
				for _, codePoint := range buffer {
					if codePoint == ':' && !passwordTokenSeen {
						passwordTokenSeen = true
						continue
					}
					encoded := percentEncode(string(codePoint), inUserinfoSet)
					if passwordTokenSeen {
						url.Password += encoded
					} else {
						url.Username += encoded
					}
				}
				buffer = buffer[:0]
			case character == eofCodePoint || character == '/' || character == '?' || character == '#' ||
				(url.SchemeKind.isSpecial() && character == '\\'):
				if atSignSeen && len(buffer) == 0 {
					return nil, ErrInvalidURL
				}
				bufferLength := len(buffer)
				buffer = buffer[:0]
				state = stateHost
				delta = -(bufferLength + 1)
			default:
				buffer = append(buffer, character)
			}

		case stateHost:
			switch {
			case character == ':' && !insideBrackets:
				if len(buffer) == 0 {
					return nil, ErrInvalidURL
				}
				host, hostKind, ok := parseHost(string(buffer), !url.SchemeKind.isSpecial())
				if !ok {
					return nil, ErrInvalidURL
				}
				url.HasHost = true
				url.Host = host
				url.HostKind = hostKind
				buffer = buffer[:0]
				state = statePort
			case character == eofCodePoint || character == '/' || character == '?' || character == '#' ||
				(url.SchemeKind.isSpecial() && character == '\\'):
				if url.SchemeKind.isSpecial() && len(buffer) == 0 {
					return nil, ErrInvalidURL
				}
				host, hostKind, ok := parseHost(string(buffer), !url.SchemeKind.isSpecial())
				if !ok {
					return nil, ErrInvalidURL
				}
				url.HasHost = true
				url.Host = host
				url.HostKind = hostKind
				buffer = buffer[:0]
				state = statePathStart
				delta = -1
			default:
				if character == '[' {
					insideBrackets = true
				}
				if character == ']' {
					insideBrackets = false
				}
				buffer = append(buffer, character)
			}

		case statePort:
			switch {
			case isASCIIDigit(character):
				buffer = append(buffer, character)
			case character == eofCodePoint || character == '/' || character == '?' || character == '#' ||
				(url.SchemeKind.isSpecial() && character == '\\'):
				if len(buffer) > 0 {
					port, valid := parsePort(string(buffer), url.Scheme)
					if !valid {
						return nil, ErrInvalidURL
					}
					url.Port = port
					buffer = buffer[:0]
				}
				state = statePathStart
				delta = -1
			default:
				return nil, ErrInvalidURL
			}

		case stateFile:
			url.Scheme = "file"
			url.SchemeKind = SchemeFile
			url.HasHost = true
			url.Host = ""
			url.HostKind = HostDomain
			switch {
			case character == '/' || character == '\\':
				state = stateFileSlash
			case base != nil && base.Scheme == "file":
				url.Host = base.Host
				url.HostKind = base.HostKind
				url.Path = append([]string(nil), base.Path...)
				url.Query = base.Query
				url.HasQuery = base.HasQuery
				switch character {
				case '?':
					url.Query = ""
					url.HasQuery = true
					state = stateQuery
				case '#':
					url.Fragment = ""
					url.HasFragment = true
					state = stateFragment
				case eofCodePoint:
					// The URL is a clone of the base; fall through to return.
				default:
					url.Query = ""
					url.HasQuery = false
					if !startsWithWindowsDriveLetter(characters[index:]) {
						shortenPath(url)
					} else {
						url.Path = nil
					}
					state = statePath
					delta = -1
				}
			default:
				state = statePath
				delta = -1
			}

		case stateFileSlash:
			if character == '/' || character == '\\' {
				state = stateFileHost
			} else {
				if base != nil && base.Scheme == "file" {
					url.Host = base.Host
					url.HostKind = base.HostKind
					if !startsWithWindowsDriveLetter(characters[index:]) &&
						len(base.Path) > 0 && isNormalizedWindowsDriveLetter(base.Path[0]) {
						url.Path = append(url.Path, base.Path[0])
					}
				}
				state = statePath
				delta = -1
			}

		case stateFileHost:
			switch {
			case character == eofCodePoint || character == '/' || character == '\\' ||
				character == '?' || character == '#':
				delta = -1
				switch {
				case isWindowsDriveLetter(string(buffer)):
					state = statePath
				case len(buffer) == 0:
					url.Host = ""
					state = statePathStart
				default:
					host, _, ok := parseHost(string(buffer), false)
					if !ok {
						return nil, ErrInvalidURL
					}
					if host == "localhost" {
						host = ""
					}
					url.Host = host
					url.HostKind = HostDomain
					buffer = buffer[:0]
					state = statePathStart
				}
			default:
				buffer = append(buffer, character)
			}

		case statePathStart:
			switch {
			case url.SchemeKind.isSpecial():
				state = statePath
				if character != '/' && character != '\\' {
					delta = -1
				}
			case character == '?':
				url.Query = ""
				url.HasQuery = true
				state = stateQuery
			case character == '#':
				url.Fragment = ""
				url.HasFragment = true
				state = stateFragment
			case character != eofCodePoint:
				state = statePath
				if character != '/' {
					delta = -1
				}
			}

		case statePath:
			isDelimiter := character == eofCodePoint || character == '/' ||
				(url.SchemeKind.isSpecial() && character == '\\') ||
				character == '?' || character == '#'
			if !isDelimiter {
				buffer = append(buffer, []rune(percentEncode(string(character), inPathSet))...)
				break
			}
			switch {
			case isDoubleDotSegment(string(buffer)):
				shortenPath(url)
				if character != '/' && !(url.SchemeKind.isSpecial() && character == '\\') {
					url.Path = append(url.Path, "")
				}
			case isSingleDotSegment(string(buffer)):
				if character != '/' && !(url.SchemeKind.isSpecial() && character == '\\') {
					url.Path = append(url.Path, "")
				}
			default:
				if url.Scheme == "file" && len(url.Path) == 0 && isWindowsDriveLetter(string(buffer)) {
					buffer[1] = ':'
				}
				url.Path = append(url.Path, string(buffer))
			}
			buffer = buffer[:0]
			if character == '?' {
				url.Query = ""
				url.HasQuery = true
				state = stateQuery
			}
			if character == '#' {
				url.Fragment = ""
				url.HasFragment = true
				state = stateFragment
			}

		case stateOpaquePath:
			switch {
			case character == '?':
				url.Query = ""
				url.HasQuery = true
				state = stateQuery
			case character == '#':
				url.Fragment = ""
				url.HasFragment = true
				state = stateFragment
			case character == ' ':
				if remainingStartsWith("?") || remainingStartsWith("#") {
					url.OpaquePath += "%20"
				} else {
					url.OpaquePath += " "
				}
			case character != eofCodePoint:
				url.OpaquePath += percentEncode(string(character), inC0ControlSet)
			}

		case stateQuery:
			switch {
			case character == '#' || character == eofCodePoint:
				encodeSet := inQuerySet
				if url.SchemeKind.isSpecial() {
					encodeSet = inSpecialQuerySet
				}
				url.Query += percentEncode(string(buffer), encodeSet)
				buffer = buffer[:0]
				if character == '#' {
					url.Fragment = ""
					url.HasFragment = true
					state = stateFragment
				}
			case character != eofCodePoint:
				buffer = append(buffer, character)
			}

		case stateFragment:
			if character != eofCodePoint {
				url.Fragment += percentEncode(string(character), inFragmentSet)
			}
		}

		next := index + delta
		if next >= len(characters) {
			return url, nil
		}
		index = next + 1
	}
}

func parsePort(input string, scheme string) (string, bool) {
	port := 0
	for _, character := range input {
		port = port*10 + int(character-'0')
		if port > 65535 {
			return "", false
		}
	}
	serialized := strconv.Itoa(port)
	if serialized == defaultPort(scheme) {
		return "", true
	}
	return serialized, true
}

func shortenPath(url *URL) {
	if url.Scheme == "file" && len(url.Path) == 1 && isNormalizedWindowsDriveLetter(url.Path[0]) {
		return
	}
	if len(url.Path) > 0 {
		url.Path = url.Path[:len(url.Path)-1]
	}
}

func isWindowsDriveLetter(segment string) bool {
	return len(segment) == 2 && isASCIIAlpha(rune(segment[0])) &&
		(segment[1] == ':' || segment[1] == '|')
}

func isNormalizedWindowsDriveLetter(segment string) bool {
	return len(segment) == 2 && isASCIIAlpha(rune(segment[0])) && segment[1] == ':'
}

func startsWithWindowsDriveLetter(characters []rune) bool {
	if len(characters) < 2 {
		return false
	}
	if !isASCIIAlpha(characters[0]) || (characters[1] != ':' && characters[1] != '|') {
		return false
	}
	return len(characters) == 2 || characters[2] == '/' || characters[2] == '\\' ||
		characters[2] == '?' || characters[2] == '#'
}

func isSingleDotSegment(segment string) bool {
	return segment == "." || strings.EqualFold(segment, "%2e")
}

func isDoubleDotSegment(segment string) bool {
	if segment == ".." {
		return true
	}
	return strings.EqualFold(segment, "%2e.") || strings.EqualFold(segment, ".%2e") ||
		strings.EqualFold(segment, "%2e%2e")
}

func isASCIIAlpha(character rune) bool {
	return (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z')
}

func isASCIIDigit(character rune) bool {
	return character >= '0' && character <= '9'
}

func isASCIIAlphanumeric(character rune) bool {
	return isASCIIAlpha(character) || isASCIIDigit(character)
}

func toASCIILower(character rune) rune {
	if character >= 'A' && character <= 'Z' {
		return character + 32
	}
	return character
}

func trimC0ControlOrSpace(input string) string {
	return strings.TrimFunc(input, func(character rune) bool {
		return character <= 0x20
	})
}

func removeASCIITabOrNewline(input string) string {
	return strings.Map(func(character rune) rune {
		if character == '\t' || character == '\n' || character == '\r' {
			return -1
		}
		return character
	}, input)
}
