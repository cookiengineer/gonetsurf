package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestLayoutCommand(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "page.html")
	source := `<!doctype html><html><body style="margin:0"><style>#target { width:100px; height:50px; padding:10px; border:2px solid }</style><div id="target"></div></body></html>`
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	original := os.Stdout
	os.Stdout = writer
	layoutErr := layout([]string{path, "400", "600"})
	writer.Close()
	os.Stdout = original
	if layoutErr != nil {
		t.Fatalf("layout: %v", layoutErr)
	}
	encoded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	var root layoutBoxJSON
	if err := json.Unmarshal(encoded, &root); err != nil {
		t.Fatalf("decode layout JSON: %v", err)
	}
	target := findLayoutBox(root, "target")
	if target == nil {
		t.Fatal("target box missing")
	}
	if target.Width != 124 || target.Height != 74 || target.ContentWidth != 100 || target.ContentHeight != 50 {
		t.Errorf("target geometry = %vx%v content %vx%v", target.Width, target.Height, target.ContentWidth, target.ContentHeight)
	}
}

func findLayoutBox(box layoutBoxJSON, id string) *layoutBoxJSON {
	if box.ID == id {
		return &box
	}
	for index := range box.Children {
		if found := findLayoutBox(box.Children[index], id); found != nil {
			return found
		}
	}
	return nil
}
