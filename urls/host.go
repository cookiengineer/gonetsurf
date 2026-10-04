package urls

import (
	"fmt"
	"net/netip"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/idna"
)

// HostKind classifies a parsed host.
type HostKind uint8

const (
	HostNone HostKind = iota
	HostOpaque
	HostDomain
	HostIPv4
	HostIPv6
)

// whatwgIDNA mirrors the URL Standard's domain-to-ASCII profile: UTS #46
// mapping, non-transitional processing, UseSTD3ASCIIRules disabled, and no
// DNS-length verification.
var whatwgIDNA = idna.New(
	idna.MapForLookup(),
	idna.StrictDomainName(false),
	idna.Transitional(false),
	idna.CheckHyphens(false),
	idna.BidiRule(),
)

func isForbiddenHostCodePoint(character rune) bool {
	switch character {
	case 0x00, 0x09, 0x0A, 0x0D, 0x20, '#', '/', ':', '<', '>', '?', '@',
		'[', '\\', ']', '^', '|':
		return true
	default:
		return false
	}
}

func isForbiddenDomainCodePoint(character rune) bool {
	return isForbiddenHostCodePoint(character) || character <= 0x1F ||
		character == '%' || character == 0x7F
}

// parseHost implements the URL Standard host parser.
//
// isOpaque selects between an opaque host (non-special URLs) and a domain host.
func parseHost(input string, isOpaque bool) (string, HostKind, bool) {
	if strings.HasPrefix(input, "[") {
		if !strings.HasSuffix(input, "]") {
			return "", HostNone, false
		}
		address, ok := parseIPv6(input[1 : len(input)-1])
		if !ok {
			return "", HostNone, false
		}
		return "[" + address + "]", HostIPv6, true
	}

	if isOpaque {
		for _, character := range input {
			if isForbiddenHostCodePoint(character) {
				return "", HostNone, false
			}
		}
		return percentEncode(input, inC0ControlSet), HostOpaque, true
	}

	if input == "" {
		return "", HostNone, false
	}

	decoded := percentDecode(input)
	if !utf8.Valid(decoded) {
		decoded = []byte(strings.ToValidUTF8(string(decoded), "\uFFFD"))
	}

	asciiDomain, ok := domainToASCII(string(decoded))
	if !ok || asciiDomain == "" {
		return "", HostNone, false
	}

	if endsInNumber(asciiDomain) {
		address, ok := parseIPv4(asciiDomain)
		if !ok {
			return "", HostNone, false
		}
		return address, HostIPv4, true
	}

	for _, character := range asciiDomain {
		if isForbiddenDomainCodePoint(character) {
			return "", HostNone, false
		}
	}
	return asciiDomain, HostDomain, true
}

// domainToASCII implements the URL Standard domain-to-ASCII with beStrict
// false. Pure ASCII input is lowercased without IDNA processing, matching the
// behavior browsers exhibit for already-ASCII labels such as invalid punycode.
func domainToASCII(input string) (string, bool) {
	if isASCIIString(input) {
		return strings.ToLower(input), true
	}
	ascii, err := whatwgIDNA.ToASCII(input)
	if err != nil {
		return "", false
	}
	return ascii, true
}

func isASCIIString(input string) bool {
	for index := 0; index < len(input); index++ {
		if input[index] >= 0x80 {
			return false
		}
	}
	return true
}

// endsInNumber implements the URL Standard ends-in-a-number checker.
func endsInNumber(domain string) bool {
	parts := strings.Split(domain, ".")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	if len(parts) == 0 {
		return false
	}
	last := parts[len(parts)-1]
	if last != "" && isAllASCIIDigits(last) {
		return true
	}
	_, _, ok := parseIPv4Number(last)
	return ok
}

func isAllASCIIDigits(input string) bool {
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

func digitValue(character rune) (int, bool) {
	switch {
	case character >= '0' && character <= '9':
		return int(character - '0'), true
	case character >= 'a' && character <= 'f':
		return int(character-'a') + 10, true
	case character >= 'A' && character <= 'F':
		return int(character-'A') + 10, true
	default:
		return 0, false
	}
}

// parseIPv4Number implements the URL Standard IPv4 number parser. It returns
// the parsed value, whether a non-decimal prefix was used, and whether parsing
// succeeded. The value saturates at the maximum uint64 so that a syntactically
// valid but out-of-range number is rejected later by the IPv4 parser's range
// check rather than by the number parser.
func parseIPv4Number(input string) (uint64, bool, bool) {
	if input == "" {
		return 0, false, false
	}
	radix := uint64(10)
	nonDecimal := false
	remaining := input
	switch {
	case len(input) >= 2 && (strings.HasPrefix(input, "0x") || strings.HasPrefix(input, "0X")):
		radix = 16
		nonDecimal = true
		remaining = input[2:]
	case len(input) >= 2 && input[0] == '0':
		radix = 8
		nonDecimal = true
		remaining = input[1:]
	}
	if remaining == "" {
		return 0, true, true
	}
	var value uint64
	overflow := false
	for _, character := range remaining {
		digit, ok := digitValue(character)
		if !ok || uint64(digit) >= radix {
			return 0, false, false
		}
		if !overflow {
			if value > (^uint64(0)-uint64(digit))/radix {
				overflow = true
			} else {
				value = value*radix + uint64(digit)
			}
		}
	}
	if overflow {
		value = ^uint64(0)
	}
	return value, nonDecimal, true
}

// parseIPv4 implements the URL Standard IPv4 parser, returning the canonical
// dotted-decimal form.
func parseIPv4(input string) (string, bool) {
	parts := strings.Split(input, ".")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		if len(parts) == 1 {
			return "", false
		}
		parts = parts[:len(parts)-1]
	}
	if len(parts) > 4 {
		return "", false
	}

	numbers := make([]uint64, 0, len(parts))
	for _, part := range parts {
		value, _, ok := parseIPv4Number(part)
		if !ok {
			return "", false
		}
		numbers = append(numbers, value)
	}

	for index := 0; index < len(numbers)-1; index++ {
		if numbers[index] > 255 {
			return "", false
		}
	}
	limit := uint64(1)
	for shift := 0; shift < 5-len(numbers); shift++ {
		limit *= 256
	}
	if numbers[len(numbers)-1] >= limit {
		return "", false
	}

	address := numbers[len(numbers)-1]
	for index := 0; index < len(numbers)-1; index++ {
		address += numbers[index] * shiftFactor(3-index)
	}
	return fmt.Sprintf("%d.%d.%d.%d",
		byte(address>>24), byte(address>>16), byte(address>>8), byte(address)), true
}

func shiftFactor(exponent int) uint64 {
	factor := uint64(1)
	for count := 0; count < exponent; count++ {
		factor *= 256
	}
	return factor
}

// parseIPv6 implements the URL Standard IPv6 parser using net/netip for the
// address grammar and canonical serialization.
func parseIPv6(input string) (string, bool) {
	if strings.ContainsRune(input, '%') {
		return "", false
	}
	address, err := netip.ParseAddr(input)
	if err != nil || !address.Is6() {
		return "", false
	}
	return address.String(), true
}
