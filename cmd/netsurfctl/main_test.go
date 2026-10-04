package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		io.WriteString(writer, "hello fetch")
	}))
	defer server.Close()

	status, body, err := fetchBody(server.URL)
	if err != nil {
		t.Fatalf("fetchBody: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("status = %d, want 200", status)
	}
	if string(body) != "hello fetch" {
		t.Errorf("body = %q", string(body))
	}

	if _, _, err := fetchBody("ftp://example.com/"); err == nil {
		t.Error("expected an error for an unsupported scheme")
	}
}

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
