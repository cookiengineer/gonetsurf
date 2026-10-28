package dom

import "testing"

// FuzzSelectors checks that arbitrary selector text never panics during
// parsing or matching.
func FuzzSelectors(f *testing.F) {
	seeds := []string{
		"div",
		"div > p.first",
		"li:nth-child(2n+1)",
		"[data-role=main]",
		"div:has(> p)",
		":is(.a, #b) .c",
		"a[href^='x' i]",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	document, _, _, _, _ := selectorFixture()
	f.Fuzz(func(t *testing.T, selector string) {
		_ = document.QuerySelectorAll(selector)
	})
}
