package http1

import (
	"context"
	"errors"
	"strings"

	"gonetsurf/urls"
)

// ErrTooManyRedirects is returned when the redirect limit is exceeded.
var ErrTooManyRedirects = errors.New("http1: too many redirects")

// Client performs requests and follows redirects.
type Client struct {
	Transport     *Transport
	CheckRedirect func(request *Request, via []*Request) error
	MaxRedirects  int
}

// Do sends a request and follows redirects.
func (client *Client) Do(ctx context.Context, request *Request) (*Response, error) {
	transport := client.Transport
	if transport == nil {
		transport = &Transport{}
	}
	maximum := client.MaxRedirects
	if maximum == 0 {
		maximum = 10
	}
	var via []*Request
	current := request
	for redirects := 0; ; redirects++ {
		response, err := transport.RoundTrip(ctx, current)
		if err != nil {
			return nil, err
		}
		location := response.Header.Get("Location")
		if !isRedirectStatus(response.StatusCode) || location == "" {
			return response, nil
		}
		if client.CheckRedirect != nil {
			if err := client.CheckRedirect(current, via); err != nil {
				return response, err
			}
		}
		if redirects >= maximum {
			response.Close()
			return nil, ErrTooManyRedirects
		}
		next, err := redirectURL(current.URL, location)
		if err != nil {
			response.Close()
			return nil, err
		}
		method := redirectedMethod(current.Method, response.StatusCode)
		via = append(via, current)
		response.Close()
		current = &Request{
			Method: method,
			URL:    next,
			Proto:  "HTTP/1.1",
			Header: redirectedHeader(current, next, method),
			Body:   nil,
		}
	}
}

func isRedirectStatus(status int) bool {
	switch status {
	case 301, 302, 303, 307, 308:
		return true
	default:
		return false
	}
}

func redirectedMethod(method string, status int) string {
	switch status {
	case 303:
		if method != "HEAD" {
			return "GET"
		}
	case 301, 302:
		if method == "POST" {
			return "GET"
		}
	}
	return method
}

func redirectedHeader(previous *Request, next *urls.URL, method string) Header {
	header := previous.Header.Clone()
	header.Del("Host")
	if method == "GET" {
		header.Del("Content-Length")
		header.Del("Content-Type")
		header.Del("Transfer-Encoding")
	}
	if previous.URL != nil && !previous.URL.SameOrigin(next) {
		header.Del("Authorization")
		header.Del("Cookie")
		header.Del("Proxy-Authorization")
	}
	// Reject HTTPS to HTTP downgrades for security-sensitive headers.
	return header
}

func redirectURL(base *urls.URL, location string) (*urls.URL, error) {
	trimmed := strings.TrimSpace(location)
	return urls.ParseReference(base, trimmed)
}
