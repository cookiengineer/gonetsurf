package urls

import "golang.org/x/net/publicsuffix"

// Origin is a URL origin: either a tuple origin or an opaque origin.
type Origin struct {
	Scheme string
	Host   string
	Port   string
	Opaque bool
}

// Origin returns the origin of the URL.
func (u *URL) Origin() Origin {
	if u.Scheme == "blob" {
		// A blob URL's origin is the origin of its path when that path is an
		// http, https, or file URL; otherwise it is opaque.
		inner, err := Parse(u.OpaquePath)
		if err != nil {
			return Origin{Opaque: true}
		}
		switch inner.Scheme {
		case "http", "https":
			return inner.Origin()
		default:
			return Origin{Opaque: true}
		}
	}
	switch u.Scheme {
	case "http", "https", "ws", "wss", "ftp":
		port := u.Port
		if port == "" {
			port = defaultPort(u.Scheme)
		}
		return Origin{Scheme: u.Scheme, Host: u.Host, Port: port}
	default:
		return Origin{Opaque: true}
	}
}

// IsOpaque reports whether the origin is opaque.
func (o Origin) IsOpaque() bool {
	return o.Opaque
}

// String serializes the origin, returning "null" for opaque origins.
func (o Origin) String() string {
	if o.Opaque {
		return "null"
	}
	if o.Port != "" && o.Port != defaultPort(o.Scheme) {
		return o.Scheme + "://" + o.Host + ":" + o.Port
	}
	return o.Scheme + "://" + o.Host
}

// OriginString returns the serialized origin of the URL.
func (u *URL) OriginString() string {
	return u.Origin().String()
}

// SameOrigin reports whether two URLs share the same origin.
func (u *URL) SameOrigin(other *URL) bool {
	first := u.Origin()
	second := other.Origin()
	if first.Opaque || second.Opaque {
		return false
	}
	return first == second
}

// Site returns the registrable domain of the URL's host, or the host itself
// when it is not a registrable domain (for example an IP address). Opaque
// origins, including file and data URLs, have no site.
func (u *URL) Site() string {
	origin := u.Origin()
	if origin.Opaque || origin.Host == "" {
		return ""
	}
	domain, err := publicsuffix.EffectiveTLDPlusOne(origin.Host)
	if err != nil {
		return origin.Host
	}
	return domain
}

// SameSite reports whether two URLs are schemelessly same site.
func (u *URL) SameSite(other *URL) bool {
	if u.Origin().Opaque || other.Origin().Opaque {
		return false
	}
	return u.Site() == other.Site()
}
