package cssom

import "testing"

// FuzzParseSerialize checks that parsing arbitrary CSS and serializing the
// result never panics and always produces reparseable CSS.
func FuzzParseSerialize(f *testing.F) {
	seeds := []string{
		"",
		"a { color: red; }",
		"a, b:hover > c::after { margin: 0 auto !important }",
		"@media screen { a { color: red } }",
		"@import url(foo.css);",
		"div { --brand: green; width: calc(100% - 2px); }",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		sheet := Parse(input)
		serialized := sheet.Serialize()
		_ = Parse(serialized)
	})
}
