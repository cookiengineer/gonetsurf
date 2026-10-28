package encodings

import "testing"

func FuzzSniff(f *testing.F) {
	for _, seed := range []string{
		"",
		"<meta charset=\"utf-8\">",
		"<!doctype html><meta http-equiv=content-type content=\"text/html; charset=shift_jis\">",
		"\xef\xbb\xbfhello",
	} {
		f.Add(seed, "text/html")
	}
	f.Fuzz(func(t *testing.T, data, contentType string) {
		result := Sniff(contentType, []byte(data))
		_, _ = result.Decode([]byte(data))
		_, _ = result.ApplyMetaChange("utf-16le")
		_ = Prescan([]byte(data))
	})
}
