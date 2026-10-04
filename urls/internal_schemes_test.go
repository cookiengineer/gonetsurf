package urls

import "testing"

func TestInternalSchemes(t *testing.T) {
	tests := []struct {
		input    string
		internal InternalScheme
		href     string
	}{
		{"about:blank", InternalAbout, "about:blank"},
		{"data:text/plain,hello", InternalData, "data:text/plain,hello"},
		{"data:text/plain;base64,aGVsbG8=", InternalData, "data:text/plain;base64,aGVsbG8="},
		{"blob:https://example.com/id", InternalBlob, "blob:https://example.com/id"},
		{"file:///etc/hosts", InternalFile, "file:///etc/hosts"},
		{"netsurf-cache://example.com/a", InternalCache, "netsurf-cache://example.com/a"},
		{"https://example.com/", SchemeUnknown, "https://example.com/"},
	}
	for _, test := range tests {
		parsed, err := Parse(test.input)
		if err != nil {
			t.Fatalf("Parse(%q): %v", test.input, err)
		}
		if got := parsed.Internal(); got != test.internal {
			t.Errorf("Parse(%q).Internal() = %d, want %d", test.input, got, test.internal)
		}
		if got := parsed.String(); got != test.href {
			t.Errorf("Parse(%q).String() = %q, want %q", test.input, got, test.href)
		}
	}
}

func TestDataURL(t *testing.T) {
	parsed, err := Parse("data:text/plain;base64,aGVsbG8=")
	if err != nil {
		t.Fatal(err)
	}
	mediaType, body, ok := parsed.DataURL()
	if !ok {
		t.Fatal("expected a data URL")
	}
	if mediaType != "text/plain" {
		t.Errorf("media type = %q, want %q", mediaType, "text/plain")
	}
	if string(body) != "hello" {
		t.Errorf("body = %q, want %q", string(body), "hello")
	}

	plain, _ := Parse("data:,hello%20world")
	_, plainBody, ok := plain.DataURL()
	if !ok || string(plainBody) != "hello world" {
		t.Errorf("plain data body = %q (ok=%v), want %q", string(plainBody), ok, "hello world")
	}
}
