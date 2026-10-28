package values

import (
	"testing"

	"gonetsurf/parsers/css/parser"
)

// FuzzMath checks that arbitrary math functions and unicode ranges never
// panic.
func FuzzMath(f *testing.F) {
	seeds := []string{
		"calc(1px + 2px)",
		"min(1px, 2px)",
		"clamp(1px, 5px, 3px)",
		"round(up, 10px, 3px)",
		"hypot(3px, 4px)",
		"calc(1em + 2%)",
		"u+1-2, u+??",
		"calc()",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		for _, value := range parser.ParseComponentValueList(input) {
			function, ok := value.(*parser.Function)
			if !ok {
				continue
			}
			if expression, ok := ParseMath(function); ok {
				_, _ = expression.Simplify()
			}
		}
		_ = ParseURangeList(input)
	})
}
