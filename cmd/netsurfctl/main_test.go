package main

import (
	"encoding/json"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestPackagesHaveDocComments(t *testing.T) {
	output, err := exec.Command("go", "list", "-f", "{{.Dir}}", "gonetsurf/...").Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	for _, directory := range strings.Fields(string(output)) {
		entries, err := os.ReadDir(directory)
		if err != nil {
			t.Fatalf("read %s: %v", directory, err)
		}
		found := false
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			source, err := os.ReadFile(filepath.Join(directory, name))
			if err != nil {
				t.Fatalf("read %s: %v", name, err)
			}
			trimmed := strings.TrimSpace(string(source))
			if strings.HasPrefix(trimmed, "// Package ") || strings.Contains(trimmed, "\n// Package ") ||
				strings.HasPrefix(trimmed, "// Command ") || strings.Contains(trimmed, "\n// Command ") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("package in %s has no package doc comment", directory)
		}
	}
}

func TestRenderCommand(t *testing.T) {
	directory := t.TempDir()
	htmlPath := filepath.Join(directory, "page.html")
	source := `<!doctype html><html><head><style>html, body { margin: 0 }</style></head>` +
		`<body><div style="width:200px;height:100px;background-color:#ff0000"></div></body></html>`
	if err := os.WriteFile(htmlPath, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	outputPath := filepath.Join(directory, "page.png")
	if err := render([]string{htmlPath, outputPath, "200", "100"}); err != nil {
		t.Fatalf("render: %v", err)
	}
	file, err := os.Open(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	decoded, err := png.Decode(file)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.Bounds().Dx() != 200 || decoded.Bounds().Dy() != 100 {
		t.Errorf("image bounds = %v", decoded.Bounds())
	}
	red, green, blue, alpha := decoded.At(10, 10).RGBA()
	if red>>8 != 255 || green>>8 != 0 || blue>>8 != 0 || alpha>>8 != 255 {
		t.Errorf("painted pixel = (%d, %d, %d, %d), want red", red>>8, green>>8, blue>>8, alpha>>8)
	}
}

func TestReftestCommand(t *testing.T) {
	directory := t.TempDir()
	firstPath := filepath.Join(directory, "first.html")
	secondPath := filepath.Join(directory, "second.html")
	source := `<!doctype html><html><body style="margin:0"><div style="width:20px;height:20px;background:#ff0000"></div></body></html>`
	if err := os.WriteFile(firstPath, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secondPath, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := reftestCommand([]string{firstPath, secondPath, "40", "40"}); err != nil {
		t.Fatalf("identical reftest: %v", err)
	}
	other := `<!doctype html><html><body style="margin:0"><div style="width:20px;height:20px;background:#0000ff"></div></body></html>`
	if err := os.WriteFile(secondPath, []byte(other), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := reftestCommand([]string{firstPath, secondPath, "40", "40"}); err == nil {
		t.Fatal("expected a reftest mismatch")
	}
}
