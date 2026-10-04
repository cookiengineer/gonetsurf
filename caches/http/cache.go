// Package httpcache implements HTTP caching per RFC 7234 plus the project's
// aggressive archive policy. See docs/cache.md.
package httpcache

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Header is a case-preserving multimap of HTTP header fields.
type Header map[string][]string

// Get returns the first value of a header field, case-insensitively.
func (header Header) Get(name string) string {
	for key, values := range header {
		if strings.EqualFold(key, name) && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

// Values returns all values of a header field, case-insensitively.
func (header Header) Values(name string) []string {
	for key, values := range header {
		if strings.EqualFold(key, name) {
			return values
		}
	}
	return nil
}

// CacheControl parses all Cache-Control directives into a lowercase map.
func CacheControl(header Header) map[string]string {
	directives := make(map[string]string)
	for _, value := range header.Values("Cache-Control") {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			name, argument, _ := strings.Cut(part, "=")
			name = strings.ToLower(strings.TrimSpace(name))
			argument = strings.Trim(strings.TrimSpace(argument), `"`)
			directives[name] = argument
		}
	}
	return directives
}

// Entry is a stored HTTP response.
type Entry struct {
	Method       string
	URL          string
	Status       int
	Header       Header
	Body         []byte
	StoredAt     time.Time
	RequestTime  time.Time
	ResponseTime time.Time
	Partition    string
	VaryKey      string
}

// Policy controls caching behavior.
type Policy struct {
	// Aggressive retains and serves responses past server freshness and
	// ignores no-cache/must-revalidate on reads. See docs/cache.md.
	Aggressive bool
	// Now allows tests to inject a clock.
	Now func() time.Time
}

func (policy Policy) now() time.Time {
	if policy.Now != nil {
		return policy.Now()
	}
	return time.Now()
}

// cacheableByDefault lists status codes that are cacheable by default.
var cacheableByDefault = map[int]bool{
	200: true, 203: true, 204: true, 206: true, 300: true, 301: true,
	308: true, 404: true, 405: true, 410: true, 414: true, 501: true,
}

// CanStore reports whether a response may be stored. It is intentionally
// permissive about private/authorization responses because the archive is
// local and single-user.
func CanStore(method string, status int, requestHeader, responseHeader Header) bool {
	if method != http.MethodGet && method != http.MethodHead {
		return false
	}
	requestDirectives := CacheControl(requestHeader)
	if _, ok := requestDirectives["no-store"]; ok {
		return false
	}
	responseDirectives := CacheControl(responseHeader)
	if _, ok := responseDirectives["no-store"]; ok {
		return false
	}
	if responseHeader.Get("Vary") == "*" {
		return false
	}
	if _, ok := responseDirectives["max-age"]; ok {
		return true
	}
	if _, ok := responseDirectives["s-maxage"]; ok {
		return true
	}
	if responseHeader.Get("Expires") != "" {
		return true
	}
	if _, ok := responseDirectives["public"]; ok {
		return true
	}
	return cacheableByDefault[status]
}

// FreshnessLifetime computes the freshness lifetime per RFC 7234 §4.2.
func FreshnessLifetime(entry *Entry) time.Duration {
	directives := CacheControl(entry.Header)
	if value, ok := directives["max-age"]; ok {
		return parseSeconds(value)
	}
	if value, ok := directives["s-maxage"]; ok {
		return parseSeconds(value)
	}
	if expires := parseDate(entry.Header.Get("Expires")); !expires.IsZero() {
		if date := parseDate(entry.Header.Get("Date")); !date.IsZero() {
			if lifetime := expires.Sub(date); lifetime > 0 {
				return lifetime
			}
			return 0
		}
	}
	if lastModified := parseDate(entry.Header.Get("Last-Modified")); !lastModified.IsZero() {
		if date := parseDate(entry.Header.Get("Date")); !date.IsZero() {
			if heuristic := date.Sub(lastModified) / 10; heuristic > 0 {
				return heuristic
			}
		}
	}
	return 0
}

// CurrentAge computes the current age of a stored response per RFC 7234 §4.2.3.
func CurrentAge(entry *Entry, now time.Time) time.Duration {
	dateValue := parseDate(entry.Header.Get("Date"))
	apparentAge := time.Duration(0)
	if !dateValue.IsZero() {
		apparentAge = entry.ResponseTime.Sub(dateValue)
		if apparentAge < 0 {
			apparentAge = 0
		}
	}
	ageValue := parseSeconds(entry.Header.Get("Age"))
	responseDelay := entry.ResponseTime.Sub(entry.RequestTime)
	if responseDelay < 0 {
		responseDelay = 0
	}
	correctedAgeValue := ageValue + responseDelay
	correctedInitialAge := apparentAge
	if correctedAgeValue > correctedInitialAge {
		correctedInitialAge = correctedAgeValue
	}
	residentTime := now.Sub(entry.ResponseTime)
	if residentTime < 0 {
		residentTime = 0
	}
	return correctedInitialAge + residentTime
}

// IsFresh reports whether the entry can be served without revalidation.
func IsFresh(entry *Entry, policy Policy) bool {
	if policy.Aggressive {
		return true
	}
	directives := CacheControl(entry.Header)
	if _, ok := directives["no-cache"]; ok {
		return false
	}
	if _, ok := directives["must-revalidate"]; ok {
		return FreshnessLifetime(entry) > CurrentAge(entry, policy.now())
	}
	return FreshnessLifetime(entry) > CurrentAge(entry, policy.now())
}

// RequiresRevalidation reports whether a stale entry must be revalidated.
func RequiresRevalidation(entry *Entry, policy Policy) bool {
	if policy.Aggressive {
		return false
	}
	directives := CacheControl(entry.Header)
	if _, ok := directives["no-cache"]; ok {
		return true
	}
	if _, ok := directives["must-revalidate"]; ok {
		return !IsFresh(entry, policy)
	}
	return !IsFresh(entry, policy)
}

// VaryKey returns the secondary cache key for a response's Vary header and a
// request's headers. An empty string means the response does not vary.
func VaryKey(vary string, requestHeader Header) string {
	if vary == "" {
		return ""
	}
	var builder strings.Builder
	for _, name := range strings.Split(vary, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		builder.WriteString(strings.ToLower(name))
		builder.WriteByte('=')
		builder.WriteString(strings.Join(requestHeader.Values(name), ","))
		builder.WriteByte('\n')
	}
	return builder.String()
}

// ValidatorsMatch reports whether the entry's validators match a 304 or 200
// response's validators.
func ValidatorsMatch(entry *Entry, responseHeader Header) bool {
	entryETag := entry.Header.Get("ETag")
	responseETag := responseHeader.Get("ETag")
	if entryETag != "" && responseETag != "" {
		return weakETagEqual(entryETag, responseETag)
	}
	entryModified := parseDate(entry.Header.Get("Last-Modified"))
	responseModified := parseDate(responseHeader.Get("Last-Modified"))
	if !entryModified.IsZero() && !responseModified.IsZero() {
		return entryModified.Equal(responseModified)
	}
	return false
}

func weakETagEqual(first, second string) bool {
	return strings.TrimPrefix(first, "W/") == strings.TrimPrefix(second, "W/")
}

func parseSeconds(value string) time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || seconds < 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func parseDate(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	parsed, err := http.ParseTime(value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}
