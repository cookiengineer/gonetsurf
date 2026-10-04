package css

import (
	"testing"

	"gonetsurf/parsers/css/animations"
	"gonetsurf/parsers/css/cascade"
	"gonetsurf/parsers/css/colors"
	"gonetsurf/parsers/css/conditions"
	"gonetsurf/parsers/css/cssom"
	"gonetsurf/parsers/css/grammar"
	"gonetsurf/parsers/css/mediaqueries"
	"gonetsurf/parsers/css/nesting"
	"gonetsurf/parsers/css/parser"
	"gonetsurf/parsers/css/selectors"
	"gonetsurf/parsers/css/tokenizer"
	"gonetsurf/parsers/css/values"
)

func FuzzCSS(f *testing.F) {
	seeds := []string{
		"",
		"a { color: red; }",
		"@media (min-width: 10px) { a { b: c } }",
		"@keyframes x { from { opacity: 0 } to { opacity: 1 } }",
		"a, b:hover > c::after { width: calc(100% - 2px) }",
		"@supports (display: grid) { a {} }",
		"#abc #abcd #abcdef color(srgb 1 0 0 / 50%)",
		"u+1-2, u+??, U+10FFFF",
		"a { & .b { c: d } }",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		tokens := tokenizer.Tokenize(input)
		if len(tokens) == 0 || tokens[len(tokens)-1].Type != tokenizer.TokenEOF {
			t.Fatalf("tokenizer did not end with EOF")
		}
		componentValues := parser.ParseComponentValueList(input)
		_ = parser.SerializeValues(componentValues)
		sheet := cssom.Parse(input)
		_ = sheet.Serialize()
		_, _ = parser.ParseDeclaration(input)
		_, _ = selectors.Parse(input)
		_, _ = colors.Parse(input)
		_ = values.ParseURangeList(input)
		_, _ = conditions.ParseSupports(input)
		_, _ = conditions.ParseImport(input)
		_, _ = mediaqueries.Parse(input)
		_, _ = animations.ParseKeyframesRule(input)
		_ = nesting.Flatten(nesting.Parse(input))
		_ = cascade.Cascade([]cascade.Declaration{{
			Property: "color",
			Value:    componentValues,
			Origin:   cascade.OriginAuthor,
		}})
		if definition := "a | <length>"; definition != "" {
			_, _ = grammar.MatchDefinition(definition, componentValues, grammar.NewContext())
		}
	})
}

var benchmarkStylesheet = []byte(`
a, b:hover > c::after { color: #abcdef; width: calc(100% - 2px); margin: 0 auto; }
@media (min-width: 600px) and (orientation: landscape) { .grid { display: grid; gap: 1rem 2rem; } }
@supports (display: flex) { .flex { display: flex; flex-wrap: wrap; } }
@keyframes spin { from { transform: rotate(0) } to { transform: rotate(360deg) } }
`)

func BenchmarkTokenize(b *testing.B) {
	for iteration := 0; iteration < b.N; iteration++ {
		tokenizer.Tokenize(string(benchmarkStylesheet))
	}
}

func BenchmarkParseStylesheet(b *testing.B) {
	input := string(benchmarkStylesheet)
	for iteration := 0; iteration < b.N; iteration++ {
		parser.ParseStylesheet(input)
	}
}

func BenchmarkParseSelectors(b *testing.B) {
	for iteration := 0; iteration < b.N; iteration++ {
		selectors.Parse("a, b:hover > c::after, :is(.x, #y) .z")
	}
}

func BenchmarkParseColors(b *testing.B) {
	for iteration := 0; iteration < b.N; iteration++ {
		colors.Parse("rgb(10% 20% 30% / 50%)")
	}
}
