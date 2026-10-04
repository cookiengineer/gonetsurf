package properties

import "testing"

func TestValidateSmoke(t *testing.T) {
	tests := []struct {
		property string
		value    string
		valid    bool
	}{
		{"color", "red", true},
		{"color", "#fff", true},
		{"color", "rgb(1 2 3)", true},
		{"color", "10px", false},
		{"width", "10px", true},
		{"width", "auto", true},
		{"width", "50%", true},
		{"width", "bogus", false},
		{"z-index", "3", true},
		{"z-index", "auto", true},
		{"z-index", "1.5", false},
		{"margin-top", "0 auto", false},
		{"margin-top", "auto", true},
		{"display", "grid", true},
		{"display", "bogus", false},
		{"opacity", "0.5", true},
		{"opacity", "150%", true},
		{"font-size", "medium", true},
		{"font-size", "12px", true},
		{"line-height", "normal", true},
		{"line-height", "1.5", true},
		{"animation-duration", "2s, 3s", true},
		{"animation-name", "none", true},
	}
	for _, test := range tests {
		valid, known := Validate(test.property, test.value)
		if !known {
			t.Errorf("%s not registered", test.property)
			continue
		}
		if valid != test.valid {
			t.Errorf("Validate(%q, %q) = %v, want %v", test.property, test.value, valid, test.valid)
		}
	}
}
