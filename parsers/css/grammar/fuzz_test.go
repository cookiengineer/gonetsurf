package grammar

import (
	"testing"

	"gonetsurf/parsers/css/parser"
)

// FuzzMatchDefinition checks that arbitrary grammar definitions and values
// never panic. Parse errors are expected for malformed definitions.
func FuzzMatchDefinition(f *testing.F) {
	seeds := [][2]string{
		{"<length> | auto", "10px"},
		{"[ <length> | <percentage> ]{1,4}", "10px 20% 30px"},
		{"<length>#", "10px, 20px"},
		{"<length> || auto", "auto 10px"},
		{"<length> && <percentage>", "50% 10px"},
		{"<'width'>", "auto"},
		{"[ a | b ]!", "a"},
		{"<length>{2,1}", "10px"},
	}
	for _, seed := range seeds {
		f.Add(seed[0], seed[1])
	}
	f.Fuzz(func(t *testing.T, definition, input string) {
		context := testContext()
		_, _ = MatchDefinition(definition, parser.ParseComponentValueList(input), context)
	})
}
