package selectors

import "testing"

// FuzzParse checks that arbitrary selector text never panics.
func FuzzParse(f *testing.F) {
	seeds := []string{
		"",
		"a",
		"a > b + c ~ d",
		"a, b:hover > c::after",
		":is(.a, #b) .c",
		":has(> .item)",
		":nth-child(2n+1 of .item)",
		"[href^='x' i]",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		list, err := Parse(input)
		if err != nil {
			return
		}
		_ = list.MaxSpecificity()
	})
}
