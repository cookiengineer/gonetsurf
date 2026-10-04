// Package cookies implements an RFC 6265 cookie jar with partitioning and a
// tracking-session model for privacy. See docs/privacy.md.
package cookies

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// SameSite is a cookie's SameSite policy.
type SameSite uint8

const (
	SameSiteDefault SameSite = iota
	SameSiteNone
	SameSiteLax
	SameSiteStrict
)

// Cookie is a stored cookie.
type Cookie struct {
	Name      string
	Value     string
	Domain    string
	Path      string
	Expires   time.Time
	MaxAge    int
	Secure    bool
	HttpOnly  bool
	SameSite  SameSite
	HostOnly  bool
	CreatedAt time.Time
}

// Key uniquely identifies a cookie within a partition.
type Key struct {
	Name   string
	Domain string
	Path   string
}

// Key returns the storage key for the cookie.
func (cookie Cookie) Key() Key {
	return Key{Name: cookie.Name, Domain: cookie.Domain, Path: cookie.Path}
}

// ParseSetCookie parses a Set-Cookie value in the context of the request URL.
// It returns the cookie and whether it is acceptable for the request.
func ParseSetCookie(requestHost, requestPath string, value string, now time.Time) (Cookie, bool) {
	parts := strings.Split(value, ";")
	pair := strings.TrimSpace(parts[0])
	equals := strings.IndexByte(pair, '=')
	if equals <= 0 {
		return Cookie{}, false
	}
	cookie := Cookie{
		Name:      strings.TrimSpace(pair[:equals]),
		Value:     strings.TrimSpace(pair[equals+1:]),
		CreatedAt: now,
	}
	if cookie.Name == "" || !isCookieName(cookie.Name) {
		return Cookie{}, false
	}

	domainSet := false
	pathSet := false
	expiresSet := false
	maxAgeSet := false
	for _, attribute := range parts[1:] {
		name, attributeValue, _ := strings.Cut(attribute, "=")
		name = strings.ToLower(strings.TrimSpace(name))
		attributeValue = strings.TrimSpace(attributeValue)
		switch name {
		case "expires":
			if parsed, err := http.ParseTime(attributeValue); err == nil {
				cookie.Expires = parsed
				expiresSet = true
			}
		case "max-age":
			if seconds, err := strconv.Atoi(attributeValue); err == nil {
				cookie.MaxAge = seconds
				maxAgeSet = true
			}
		case "domain":
			domain := strings.TrimPrefix(strings.ToLower(attributeValue), ".")
			if domain != "" {
				cookie.Domain = domain
				cookie.HostOnly = false
				domainSet = true
			}
		case "path":
			if strings.HasPrefix(attributeValue, "/") {
				cookie.Path = attributeValue
				pathSet = true
			}
		case "secure":
			cookie.Secure = true
		case "httponly":
			cookie.HttpOnly = true
		case "samesite":
			switch strings.ToLower(attributeValue) {
			case "none":
				cookie.SameSite = SameSiteNone
			case "lax":
				cookie.SameSite = SameSiteLax
			case "strict":
				cookie.SameSite = SameSiteStrict
			}
		}
	}

	if !domainSet {
		cookie.Domain = strings.ToLower(requestHost)
		cookie.HostOnly = true
	}
	if !pathSet {
		cookie.Path = defaultPath(requestPath)
	}
	if !domainMatches(requestHost, cookie.Domain) {
		return Cookie{}, false
	}
	if !cookie.HostOnly && isPublicSuffix(cookie.Domain) {
		return Cookie{}, false
	}
	if strings.HasPrefix(cookie.Name, "__Secure-") && !cookie.Secure {
		return Cookie{}, false
	}
	if strings.HasPrefix(cookie.Name, "__Host-") && (!cookie.Secure || !cookie.HostOnly || cookie.Path != "/") {
		return Cookie{}, false
	}
	if maxAgeSet && cookie.MaxAge <= 0 {
		cookie.Expires = now.Add(-time.Second)
	} else if !expiresSet && !maxAgeSet {
		// Session cookie: no expiry.
	}
	return cookie, true
}

// HasExpired reports whether the cookie is expired at the given time.
func (cookie Cookie) HasExpired(now time.Time) bool {
	if cookie.MaxAge < 0 {
		return true
	}
	if !cookie.Expires.IsZero() && !cookie.Expires.After(now) {
		return true
	}
	return false
}

// Matches reports whether the cookie applies to a request.
func (cookie Cookie) Matches(requestHost, requestPath string, isSecure bool) bool {
	if cookie.HasExpired(time.Now()) {
		return false
	}
	if cookie.HostOnly {
		if !strings.EqualFold(requestHost, cookie.Domain) {
			return false
		}
	} else if !domainMatches(requestHost, cookie.Domain) {
		return false
	}
	if !pathMatches(requestPath, cookie.Path) {
		return false
	}
	if cookie.Secure && !isSecure {
		return false
	}
	return true
}

func defaultPath(requestPath string) string {
	if requestPath == "" || requestPath[0] != '/' {
		return "/"
	}
	index := strings.LastIndexByte(requestPath, '/')
	if index == 0 {
		return "/"
	}
	if index < 0 {
		return "/"
	}
	return requestPath[:index]
}

func domainMatches(host, domain string) bool {
	host = strings.ToLower(host)
	domain = strings.ToLower(domain)
	if host == domain {
		return true
	}
	return strings.HasSuffix(host, "."+domain)
}

func pathMatches(requestPath, cookiePath string) bool {
	if requestPath == cookiePath {
		return true
	}
	if !strings.HasPrefix(requestPath, cookiePath) {
		return false
	}
	if strings.HasSuffix(cookiePath, "/") {
		return true
	}
	return requestPath[len(cookiePath)] == '/'
}

func isCookieName(name string) bool {
	for index := 0; index < len(name); index++ {
		character := name[index]
		if character <= 0x20 || character >= 0x7F ||
			strings.IndexByte("()<>@,;:\\\"/[]?={}", character) >= 0 {
			return false
		}
	}
	return true
}
