package urls

import "testing"

func TestOrigin(t *testing.T) {
	tests := []struct {
		input          string
		origin         string
		site           string
		opaqueExpected bool
	}{
		{"https://example.com/a", "https://example.com", "example.com", false},
		{"https://www.example.com/a", "https://www.example.com", "example.com", false},
		{"https://example.com:8443/a", "https://example.com:8443", "example.com", false},
		{"http://example.com/", "http://example.com", "example.com", false},
		{"data:text/plain,hello", "null", "", true},
		{"blob:https://example.com/id", "https://example.com", "example.com", false},
		{"blob:ftp://host/path", "null", "", true},
		{"ftp://files.example.com/x", "ftp://files.example.com", "example.com", false},
		{"https://192.168.1.1/", "https://192.168.1.1", "192.168.1.1", false},
	}
	for _, test := range tests {
		parsed, err := Parse(test.input)
		if err != nil {
			t.Fatalf("Parse(%q): %v", test.input, err)
		}
		if got := parsed.OriginString(); got != test.origin {
			t.Errorf("Parse(%q).OriginString() = %q, want %q", test.input, got, test.origin)
		}
		if parsed.Origin().IsOpaque() != test.opaqueExpected {
			t.Errorf("Parse(%q).Origin().IsOpaque() = %v, want %v",
				test.input, parsed.Origin().IsOpaque(), test.opaqueExpected)
		}
		if got := parsed.Site(); got != test.site {
			t.Errorf("Parse(%q).Site() = %q, want %q", test.input, got, test.site)
		}
	}
}

func TestSameSite(t *testing.T) {
	first, _ := Parse("https://www.example.com/a")
	second, _ := Parse("http://shop.example.com/b")
	if !first.SameSite(second) {
		t.Errorf("expected www.example.com and shop.example.com to be same site")
	}
	third, _ := Parse("https://example.org/")
	if first.SameSite(third) {
		t.Errorf("expected example.com and example.org to be different sites")
	}
	if first.SameOrigin(second) {
		t.Errorf("expected different origins across schemes/hosts")
	}
}
