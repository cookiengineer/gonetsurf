package urls

import "testing"

// FuzzParse ensures the parser never panics on arbitrary input, with and
// without a base URL.
func FuzzParse(f *testing.F) {
	seeds := []string{
		"https://example.com/a/b?c=d#e",
		"http://[::1]:8080/",
		"mailto:user@example.com",
		"data:text/plain,hello",
		"file:///etc/hosts",
		"//example.com/path",
		"../relative",
		"http://0x7f.1/",
		"blob:https://example.com/id",
	}
	for _, seed := range seeds {
		f.Add(seed, "https://base.example/dir/page")
		f.Add(seed, "")
	}
	base, _ := Parse("https://base.example/dir/page")
	f.Fuzz(func(t *testing.T, input string, _ string) {
		parsed, err := Parse(input)
		if err == nil && parsed != nil {
			_ = parsed.String()
		}
		_, _ = ParseReference(base, input)
	})
}
