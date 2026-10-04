package mediaqueries

import "testing"

func TestParseAndMatch(t *testing.T) {
	environment := DefaultEnvironment()
	environment.Width = 800
	environment.Height = 600

	tests := []struct {
		input string
		want  bool
	}{
		{"screen", true},
		{"print", false},
		{"not print", true},
		{"screen and (min-width: 600px)", true},
		{"screen and (min-width: 900px)", false},
		{"(max-width: 800px)", true},
		{"(width >= 800px)", true},
		{"(width < 800px)", false},
		{"(orientation: landscape)", true},
		{"(orientation: portrait)", false},
		{"(min-resolution: 1dppx)", true},
		{"screen, print", true},
		{"print, (min-width: 900px)", false},
		{"(prefers-color-scheme: light)", false},
	}
	for _, test := range tests {
		list, err := Parse(test.input)
		if err != nil {
			t.Errorf("Parse(%q): %v", test.input, err)
			continue
		}
		if got := list.Matches(environment); got != test.want {
			t.Errorf("Matches(%q) = %v, want %v", test.input, got, test.want)
		}
	}
}

func TestParseErrors(t *testing.T) {
	if _, err := Parse("screen 5"); err == nil {
		t.Error("expected an error for a non-feature token")
	}
}
