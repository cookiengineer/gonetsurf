package selectors

import "testing"

func mustParse(t *testing.T, input string) *SelectorList {
	t.Helper()
	list, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse(%q): %v", input, err)
	}
	return list
}

func TestParseSimpleSelectors(t *testing.T) {
	tests := []struct {
		input string
		parts int
	}{
		{"div", 1},
		{"*", 1},
		{"#id", 1},
		{".class", 1},
		{"div.class#id", 3},
		{"div:hover", 2},
		{"a::before", 2},
		{"a:before", 2},
		{"a[href]", 2},
		{"a[href=x]", 2},
		{"a[href^='x' i]", 2},
		{"&", 1},
	}
	for _, test := range tests {
		list := mustParse(t, test.input)
		if len(list.Selectors) != 1 {
			t.Errorf("Parse(%q): got %d selectors, want 1", test.input, len(list.Selectors))
			continue
		}
		if got := len(list.Selectors[0].Compounds[0].Parts); got != test.parts {
			t.Errorf("Parse(%q): got %d parts, want %d", test.input, got, test.parts)
		}
	}
}

func TestParseCombinators(t *testing.T) {
	tests := []struct {
		input      string
		combinator Combinator
	}{
		{"a b", Descendant},
		{"a > b", Child},
		{"a + b", NextSibling},
		{"a ~ b", SubsequentSibling},
		{"a || b", Column},
	}
	for _, test := range tests {
		list := mustParse(t, test.input)
		selectors := list.Selectors[0]
		if len(selectors.Compounds) != 2 {
			t.Errorf("Parse(%q): got %d compounds", test.input, len(selectors.Compounds))
			continue
		}
		if selectors.Combinators[0] != test.combinator {
			t.Errorf("Parse(%q): combinator = %v, want %v", test.input, selectors.Combinators[0], test.combinator)
		}
	}
}

func TestParseSelectorList(t *testing.T) {
	list := mustParse(t, "a, b, c")
	if len(list.Selectors) != 3 {
		t.Errorf("got %d selectors, want 3", len(list.Selectors))
	}
}

func TestFunctionalPseudoClasses(t *testing.T) {
	list := mustParse(t, ":not(.a, #b)")
	pseudo := list.Selectors[0].Compounds[0].Parts[0].(*PseudoClass)
	if pseudo.Name != "not" || pseudo.Selectors == nil || len(pseudo.Selectors.Selectors) != 2 {
		t.Errorf("pseudo = %#v", pseudo)
	}

	list = mustParse(t, ":nth-child(2n+1)")
	pseudo = list.Selectors[0].Compounds[0].Parts[0].(*PseudoClass)
	if pseudo.AnB == nil || pseudo.AnB.A != 2 || pseudo.AnB.B != 1 {
		t.Errorf("AnB = %#v", pseudo.AnB)
	}

	list = mustParse(t, ":nth-child(odd of .item)")
	pseudo = list.Selectors[0].Compounds[0].Parts[0].(*PseudoClass)
	if pseudo.AnB == nil || pseudo.AnB.A != 2 || pseudo.AnB.B != 1 {
		t.Errorf("AnB = %#v", pseudo.AnB)
	}
	if pseudo.OfSelector == nil || len(pseudo.OfSelector.Selectors) != 1 {
		t.Errorf("OfSelector = %#v", pseudo.OfSelector)
	}
}

func TestSpecificity(t *testing.T) {
	tests := []struct {
		input string
		want  Specificity
	}{
		{"*", Specificity{}},
		{"div", Specificity{C: 1}},
		{"#id", Specificity{A: 1}},
		{".class", Specificity{B: 1}},
		{"a.b#c", Specificity{A: 1, B: 1, C: 1}},
		{"a > b", Specificity{C: 2}},
		{"::before", Specificity{C: 1}},
		{":hover", Specificity{B: 1}},
		{":where(.a, #b)", Specificity{}},
		{":is(.a, #b)", Specificity{A: 1}},
		{":not(.a)", Specificity{B: 1}},
		{":nth-child(2n+1)", Specificity{B: 1}},
		{":nth-child(2n of .item)", Specificity{B: 2}},
	}
	for _, test := range tests {
		list := mustParse(t, test.input)
		if got := list.MaxSpecificity(); got != test.want {
			t.Errorf("Specificity(%q) = %+v, want %+v", test.input, got, test.want)
		}
	}
}

func TestParseErrors(t *testing.T) {
	inputs := []string{".", "a >", ":not(", "#12"}
	for _, input := range inputs {
		if _, err := Parse(input); err == nil {
			t.Errorf("Parse(%q): expected an error", input)
		}
	}
}
