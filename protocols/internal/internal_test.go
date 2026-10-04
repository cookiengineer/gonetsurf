package internal

import (
	"context"
	"testing"

	"gonetsurf/urls"
)

func TestAboutBlank(t *testing.T) {
	registry := NewRegistry()
	target, _ := urls.Parse("about:blank")
	resource, err := registry.Handle(context.Background(), target)
	if err != nil || resource.StatusCode != 200 {
		t.Fatalf("about:blank = (%+v, %v)", resource, err)
	}
}

func TestDataURL(t *testing.T) {
	registry := NewRegistry()
	target, _ := urls.Parse("data:text/plain;base64,aGVsbG8=")
	resource, err := registry.Handle(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	if string(resource.Body) != "hello" || resource.MediaType != "text/plain" {
		t.Fatalf("data resource = %+v", resource)
	}
}

func TestCacheHandler(t *testing.T) {
	handler := CacheHandler{Load: func(target *urls.URL) (string, []byte, bool) {
		if target.Host == "example.com" {
			return "text/html", []byte("<h1>cached</h1>"), true
		}
		return "", nil, false
	}}
	target, _ := urls.Parse("netsurf-cache://example.com/page")
	resource, err := handler.Handle(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	if string(resource.Body) != "<h1>cached</h1>" {
		t.Errorf("body = %q", string(resource.Body))
	}
}
