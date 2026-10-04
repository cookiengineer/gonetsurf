package http1

import (
	"io"

	"gonetsurf/urls"
)

// Request is an HTTP/1.1 request.
type Request struct {
	Method string
	URL    *urls.URL
	Proto  string
	Header Header
	Body   io.ReadCloser
}

// Host returns the authority used for the Host header.
func (request *Request) Host() string {
	if value := request.Header.Get("Host"); value != "" {
		return value
	}
	if request.URL != nil {
		return request.URL.HostWithPort()
	}
	return ""
}

// Response is an HTTP/1.1 response.
type Response struct {
	StatusCode    int
	Reason        string
	Proto         string
	Header        Header
	Body          io.ReadCloser
	ContentLength int64
	Chunked       bool
	Trailer       Header
	Request       *Request

	chunked *chunkedReader
}

// Trailers returns the trailer fields. For chunked responses the trailers are
// only available after the body has been fully read.
func (response *Response) Trailers() Header {
	if response.chunked != nil {
		return response.chunked.trailer
	}
	return response.Trailer
}

// Close closes the response body.
func (response *Response) Close() error {
	if response.Body == nil {
		return nil
	}
	return response.Body.Close()
}
