package main

import "testing"

func TestIsAllowedModule(t *testing.T) {
	tests := []struct {
		module string
		allow  bool
	}{
		{"golang.org/x/net", true},
		{"golang.org/x/text", true},
		{"golang.org/x/image", true},
		{"gonetsurf", true},
		{"github.com/grafana/sobek", false},
		{"github.com/dlclark/regexp2", false},
		{"example.com/third/party", false},
	}
	for _, test := range tests {
		if got := isAllowedModule(test.module); got != test.allow {
			t.Errorf("isAllowedModule(%q) = %v, want %v", test.module, got, test.allow)
		}
	}
}
