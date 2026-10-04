package tokenizer

import "testing"

func FuzzTokenize(f *testing.F) {
	seeds := []string{
		"",
		"a { color: red; }",
		"@media (min-width: 10px) {}",
		"url(foo) url('bar') url( baz )",
		"\\31 23 .009 9e-3 #id #123",
		"/* comment */ <!-- -->",
		"\"unterminated\nnext",
		"1e+\uFFFD\U0001F600",
		"a{b:c}",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		tokens := Tokenize(input)
		if len(tokens) == 0 {
			t.Fatal("Tokenize returned no tokens")
		}
		if tokens[len(tokens)-1].Type != TokenEOF {
			t.Fatalf("last token = %s, want EOF", tokens[len(tokens)-1].Type)
		}
		preprocessed := Preprocess([]rune(input))
		for index, token := range tokens {
			if token.End < token.Start {
				t.Fatalf("token %d has inverted offsets [%d,%d)", index, token.Start, token.End)
			}
			if token.End > len(preprocessed) {
				t.Fatalf("token %d ends at %d beyond input length %d", index, token.End, len(preprocessed))
			}
			if got := string(preprocessed[token.Start:token.End]); token.Type != TokenEOF && got != token.Representation {
				t.Fatalf("token %d representation %q does not match source slice %q", index, token.Representation, got)
			}
		}
	})
}
