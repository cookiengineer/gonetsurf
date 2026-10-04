package urls

// InternalScheme identifies schemes that the browser handles itself rather
// than fetching over the network.
type InternalScheme uint8

const (
	SchemeUnknown InternalScheme = iota
	InternalAbout
	InternalData
	InternalBlob
	InternalFile
	InternalCache
)

// internalSchemeNames maps scheme names to their internal classification.
var internalSchemeNames = map[string]InternalScheme{
	"about":         InternalAbout,
	"data":          InternalData,
	"blob":          InternalBlob,
	"file":          InternalFile,
	"netsurf-cache": InternalCache,
}

// IsInternalSchemeName reports whether a scheme is handled internally.
func IsInternalSchemeName(name string) bool {
	_, ok := internalSchemeNames[name]
	return ok
}

// Internal returns the internal classification of the URL's scheme, or
// SchemeUnknown when the URL is fetched over the network.
func (u *URL) Internal() InternalScheme {
	return internalSchemeNames[u.Scheme]
}

// DataURL returns the raw payload of a data: URL: the media type (without
// parameters) and the decoded body. See RFC 2397.
func (u *URL) DataURL() (mediaType string, body []byte, ok bool) {
	if u.Scheme != "data" {
		return "", nil, false
	}
	payload := u.OpaquePath
	comma := indexByte(payload, ',')
	if comma < 0 {
		return "", nil, false
	}
	metadata := payload[:comma]
	encoded := payload[comma+1:]
	mediaType = "text/plain;charset=US-ASCII"
	isBase64 := false
	if metadata != "" {
		mediaType = metadata
		if len(metadata) >= 7 && metadata[len(metadata)-7:] == ";base64" {
			isBase64 = true
			mediaType = metadata[:len(metadata)-7]
		}
	}
	if isBase64 {
		decoded, err := decodeBase64(encoded)
		if err != nil {
			return "", nil, false
		}
		return mediaType, decoded, true
	}
	return mediaType, []byte(percentDecodeToBytes(encoded)), true
}

func indexByte(input string, target byte) int {
	for index := 0; index < len(input); index++ {
		if input[index] == target {
			return index
		}
	}
	return -1
}

func percentDecodeToBytes(input string) string {
	return string(percentDecode(input))
}

func decodeBase64(input string) ([]byte, error) {
	return decodeBase64Std(input)
}
