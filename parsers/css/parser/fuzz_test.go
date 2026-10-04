package parser

import "testing"

func FuzzParseStylesheet(f *testing.F) {
	seeds := []string{
		"",
		"a { color: red; }",
		"@media (min-width: 10px) { a { b: c } }",
		"a{b:url(x);c:var(--x)}",
		"<!-- --> /**/",
		"a{b:!important}",
		"@import 'x';",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		_ = ParseStylesheet(input)
		_ = ParseListOfRules(input)
		_ = ParseListOfDeclarations(input)
		_ = ParseComponentValueList(input)
		if _, err := ParseDeclaration(input); err != nil {
			_ = err
		}
	})
}
