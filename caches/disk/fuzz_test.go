package disk

import (
	"testing"

	"gonetsurf/urls"
)

func FuzzPathMapping(f *testing.F) {
	for _, seed := range []string{
		"https://example.com/",
		"https://example.com/a/b?x=1#fragment",
		"http://example.com:8080/~q~/index.html",
		"data:text/plain,hello",
		"https://example.com/%2e%2e/%00",
	} {
		f.Add(seed)
	}
	layout := NewLayout("/tmp/gonetsurf-fuzz-cache")
	f.Fuzz(func(t *testing.T, input string) {
		url, err := urls.Parse(input)
		if err != nil {
			return
		}
		relative := layout.RelativePath(url)
		_, _ = layout.URLFor(relative)
		_ = layout.Path(url)
		_ = layout.Directory(url)
	})
}
