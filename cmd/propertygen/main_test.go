package main

import "testing"

func TestNormalizeGrammar(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"  a   |  b ", "a | b"},
		{"<'width'>", "<'width'>"},
		{"<palette-mix()>", "<palette-mix>"},
		{"‘red’", "'red'"},
	}
	for _, test := range tests {
		if got := normalizeGrammar(test.input); got != test.want {
			t.Errorf("normalizeGrammar(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestAnimationConstant(t *testing.T) {
	tests := map[string]string{
		"not animatable":         "NotAnimatable",
		"discrete":               "Discrete",
		"by computed value":      "ByComputedValue",
		"by computed value type": "ByComputedValue",
	}
	for input, want := range tests {
		if got := animationConstant(input); got != want {
			t.Errorf("animationConstant(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestMergeManualOverridesWin(t *testing.T) {
	specs := map[string]propertySpec{
		"display": {Name: "display", Grammar: "old", GrammarSet: true, Initial: "inline", InitialSet: true},
	}
	manual := map[string]propertySpec{
		"display": {Name: "display", Grammar: "new", GrammarSet: true},
	}
	merged := merge(specs, nil, manual)
	if merged["display"].Grammar != "new" {
		t.Errorf("grammar = %q, want new", merged["display"].Grammar)
	}
	if merged["display"].Initial != "inline" {
		t.Errorf("initial = %q, want inline", merged["display"].Initial)
	}
}
