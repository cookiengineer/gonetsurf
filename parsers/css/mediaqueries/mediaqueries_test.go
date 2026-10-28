package mediaqueries

import (
	"encoding/json"
	"os"
	"testing"
)

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
		{"(prefers-color-scheme: light)", true},
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

type environmentCase struct {
	Variant string `json:"variant"`
	Query   string `json:"query"`
	Match   bool   `json:"match"`
	Parse   *bool  `json:"parse"`
}

func TestMediaQueryEnvironmentMatrix(t *testing.T) {
	encoded, err := os.ReadFile("testdata/environments.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []environmentCase
	if err := json.Unmarshal(encoded, &cases); err != nil {
		t.Fatal(err)
	}
	environments := map[string]Environment{
		"default": DefaultEnvironment(),
		"narrow": func() Environment {
			environment := DefaultEnvironment()
			environment.Width = 400
			environment.Height = 800
			return environment
		}(),
		"wide": func() Environment {
			environment := DefaultEnvironment()
			environment.Width = 1400
			environment.Height = 800
			return environment
		}(),
		"print": func() Environment {
			environment := DefaultEnvironment()
			environment.MediaType = "print"
			environment.Update = "none"
			environment.OverflowBlock = "paged"
			environment.OverflowInline = "none"
			return environment
		}(),
		"highres": func() Environment {
			environment := DefaultEnvironment()
			environment.Resolution = 2
			environment.DynamicRange = "high"
			environment.VideoDynamicRange = "high"
			environment.ColorGamut = "p3"
			environment.VideoColorGamut = "p3"
			return environment
		}(),
		"dark": func() Environment {
			environment := DefaultEnvironment()
			environment.PrefersColorScheme = "dark"
			return environment
		}(),
		"reduce": func() Environment {
			environment := DefaultEnvironment()
			environment.PrefersReducedMotion = "reduce"
			environment.PrefersReducedData = "reduce"
			environment.PrefersReducedTransparency = "reduce"
			return environment
		}(),
		"contrast": func() Environment {
			environment := DefaultEnvironment()
			environment.PrefersContrast = "more"
			return environment
		}(),
		"touch": func() Environment {
			environment := DefaultEnvironment()
			environment.Pointer = "coarse"
			environment.AnyPointer = "coarse"
			environment.Hover = "none"
			environment.AnyHover = "none"
			return environment
		}(),
		"monochrome": func() Environment {
			environment := DefaultEnvironment()
			environment.Color = 0
			environment.Monochrome = 2
			environment.Grid = true
			return environment
		}(),
		"forced": func() Environment {
			environment := DefaultEnvironment()
			environment.ForcedColors = true
			environment.InvertedColors = true
			return environment
		}(),
		"standalone": func() Environment {
			environment := DefaultEnvironment()
			environment.DisplayMode = "standalone"
			return environment
		}(),
		"scripting": func() Environment {
			environment := DefaultEnvironment()
			environment.Scripting = "none"
			return environment
		}(),
	}
	for _, testCase := range cases {
		environment, ok := environments[testCase.Variant]
		if !ok {
			t.Fatalf("unknown environment %q", testCase.Variant)
		}
		query, err := Parse(testCase.Query)
		if testCase.Parse != nil && !*testCase.Parse {
			if err == nil {
				t.Errorf("Parse(%q) succeeded, want an error", testCase.Query)
			}
			continue
		}
		if err != nil {
			t.Errorf("Parse(%q): %v", testCase.Query, err)
			continue
		}
		if got := query.Matches(environment); got != testCase.Match {
			t.Errorf("Matches(%q, %s) = %v, want %v", testCase.Query, testCase.Variant, got, testCase.Match)
		}
	}
}
