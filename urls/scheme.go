package urls

// SchemeKind classifies URL schemes that have special parsing behavior.
//
// See https://url.spec.whatwg.org/#special-scheme.
type SchemeKind uint8

const (
	SchemeOther SchemeKind = iota
	SchemeFTP
	SchemeFile
	SchemeHTTP
	SchemeHTTPS
	SchemeWS
	SchemeWSS
)

func classifyScheme(scheme string) SchemeKind {
	switch scheme {
	case "ftp":
		return SchemeFTP
	case "file":
		return SchemeFile
	case "http":
		return SchemeHTTP
	case "https":
		return SchemeHTTPS
	case "ws":
		return SchemeWS
	case "wss":
		return SchemeWSS
	default:
		return SchemeOther
	}
}

// isSpecial reports whether the scheme uses the special-URL parsing rules.
func (kind SchemeKind) isSpecial() bool {
	return kind != SchemeOther
}

// defaultPort returns the default port for a special scheme, or the empty
// string when the scheme has no default port.
func defaultPort(scheme string) string {
	switch scheme {
	case "ftp":
		return "21"
	case "http", "ws":
		return "80"
	case "https", "wss":
		return "443"
	default:
		return ""
	}
}
