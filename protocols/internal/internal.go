// Package internal handles browser-internal URL schemes: about:, data:, and
// netsurf-cache:.
package internal

import (
	"context"
	"errors"
	"strings"

	"gonetsurf/urls"
)

// ErrNotFound is returned when an internal resource does not exist.
var ErrNotFound = errors.New("internal: resource not found")

// Resource is the result of handling an internal URL.
type Resource struct {
	MediaType  string
	Body       []byte
	StatusCode int
	Header     map[string]string
}

// Handler resolves an internal URL to a resource.
type Handler interface {
	Handle(ctx context.Context, target *urls.URL) (*Resource, error)
}

// Registry dispatches internal URLs by scheme.
type Registry struct {
	handlers map[string]Handler
}

// NewRegistry returns a registry with the default handlers registered.
func NewRegistry() *Registry {
	registry := &Registry{handlers: make(map[string]Handler)}
	registry.Register("about", AboutHandler{})
	registry.Register("data", DataHandler{})
	return registry
}

// Register adds a handler for a scheme.
func (registry *Registry) Register(scheme string, handler Handler) {
	registry.handlers[strings.ToLower(scheme)] = handler
}

// Handle resolves an internal URL.
func (registry *Registry) Handle(ctx context.Context, target *urls.URL) (*Resource, error) {
	handler, ok := registry.handlers[strings.ToLower(target.Scheme)]
	if !ok {
		return nil, ErrNotFound
	}
	return handler.Handle(ctx, target)
}

// AboutHandler handles about: URLs.
type AboutHandler struct{}

// Handle implements Handler.
func (AboutHandler) Handle(_ context.Context, target *urls.URL) (*Resource, error) {
	switch target.OpaquePath {
	case "blank":
		return &Resource{
			MediaType:  "text/html;charset=utf-8",
			Body:       []byte("<!doctype html><title></title>"),
			StatusCode: 200,
		}, nil
	default:
		return nil, ErrNotFound
	}
}

// DataHandler handles data: URLs (RFC 2397).
type DataHandler struct{}

// Handle implements Handler.
func (DataHandler) Handle(_ context.Context, target *urls.URL) (*Resource, error) {
	mediaType, body, ok := target.DataURL()
	if !ok {
		return nil, ErrNotFound
	}
	return &Resource{MediaType: mediaType, Body: body, StatusCode: 200}, nil
}

// CacheLoader resolves a cache URL to stored content.
type CacheLoader func(target *urls.URL) (mediaType string, body []byte, ok bool)

// CacheHandler serves content from the on-disk cache.
type CacheHandler struct {
	Load CacheLoader
}

// Handle implements Handler.
func (handler CacheHandler) Handle(_ context.Context, target *urls.URL) (*Resource, error) {
	if handler.Load == nil {
		return nil, ErrNotFound
	}
	// Reconstruct the original URL from the netsurf-cache path.
	original := *target
	original.Scheme = "https"
	if original.HasHost {
		mediaType, body, ok := handler.Load(&original)
		if !ok {
			return nil, ErrNotFound
		}
		return &Resource{MediaType: mediaType, Body: body, StatusCode: 200}, nil
	}
	return nil, ErrNotFound
}
