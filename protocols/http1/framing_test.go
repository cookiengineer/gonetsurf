package http1

import (
	"bufio"
	"io"
	nethttp "net/http"
	"strings"
	"testing"
)

// TestFramingCorpus checks the response framing decisions that are relevant to
// request smuggling and reply splitting.
func TestFramingCorpus(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		accept  bool
		status  int
		body    string
		chunked bool
	}{
		{
			name:   "content-length",
			raw:    "HTTP/1.1 200 OK\r\nContent-Length: 5\r\n\r\nhello",
			accept: true, status: 200, body: "hello",
		},
		{
			name:   "chunked",
			raw:    "HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nhello\r\n0\r\n\r\n",
			accept: true, status: 200, body: "hello", chunked: true,
		},
		{
			name:   "transfer-encoding-wins",
			raw:    "HTTP/1.1 200 OK\r\nContent-Length: 999\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nhello\r\n0\r\n\r\n",
			accept: true, status: 200, body: "hello", chunked: true,
		},
		{
			name:   "duplicate-content-length-same",
			raw:    "HTTP/1.1 200 OK\r\nContent-Length: 5\r\nContent-Length: 5\r\n\r\nhello",
			accept: true, status: 200, body: "hello",
		},
		{
			name:   "no-body-on-204",
			raw:    "HTTP/1.1 204 No Content\r\nContent-Length: 5\r\n\r\n",
			accept: true, status: 204, body: "",
		},
		{
			name:   "no-body-on-304",
			raw:    "HTTP/1.1 304 Not Modified\r\nContent-Length: 5\r\n\r\n",
			accept: true, status: 304, body: "",
		},
		{
			name:   "duplicate-content-length-conflict",
			raw:    "HTTP/1.1 200 OK\r\nContent-Length: 5\r\nContent-Length: 6\r\n\r\nhello",
			accept: false,
		},
		{
			name:   "negative-content-length",
			raw:    "HTTP/1.1 200 OK\r\nContent-Length: -5\r\n\r\n",
			accept: false,
		},
		{
			name:   "non-numeric-content-length",
			raw:    "HTTP/1.1 200 OK\r\nContent-Length: abc\r\n\r\n",
			accept: false,
		},
		{
			name:   "whitespace-before-colon",
			raw:    "HTTP/1.1 200 OK\r\nBad Header : value\r\nContent-Length: 0\r\n\r\n",
			accept: false,
		},
		{
			name:   "invalid-header-name",
			raw:    "HTTP/1.1 200 OK\r\nBad@Name: value\r\nContent-Length: 0\r\n\r\n",
			accept: false,
		},
		{
			name:   "invalid-status",
			raw:    "HTTP/1.1 999 Nope\r\nContent-Length: 0\r\n\r\n",
			accept: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, err := ReadResponse(bufio.NewReader(strings.NewReader(test.raw)), nil)
			if test.accept {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				defer response.Close()
				if response.StatusCode != test.status {
					t.Errorf("status = %d, want %d", response.StatusCode, test.status)
				}
				if response.Chunked != test.chunked {
					t.Errorf("chunked = %v, want %v", response.Chunked, test.chunked)
				}
				body, err := io.ReadAll(response.Body)
				if err != nil {
					t.Fatalf("read body: %v", err)
				}
				if string(body) != test.body {
					t.Errorf("body = %q, want %q", string(body), test.body)
				}
				return
			}
			if err == nil {
				response.Close()
				t.Fatalf("expected an error for %q", test.raw)
			}
		})
	}
}

// TestReadResponseDifferential compares our parser with net/http for
// well-formed responses.
func TestReadResponseDifferential(t *testing.T) {
	raws := []string{
		"HTTP/1.1 200 OK\r\nContent-Length: 5\r\n\r\nhello",
		"HTTP/1.1 404 Not Found\r\nContent-Length: 0\r\n\r\n",
		"HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nhello\r\n0\r\n\r\n",
		"HTTP/1.1 200 OK\r\nContent-Length: 11\r\nX-Test: a\r\nX-Test: b\r\n\r\nhello world",
		"HTTP/1.1 302 Found\r\nLocation: /next\r\nContent-Length: 0\r\n\r\n",
		"HTTP/1.1 200 OK\r\nContent-Length: 3\r\n\r\nabc",
	}
	for _, raw := range raws {
		ours, ourErr := ReadResponse(bufio.NewReader(strings.NewReader(raw)), nil)
		theirs, theirErr := nethttp.ReadResponse(bufio.NewReader(strings.NewReader(raw)), nil)
		if (ourErr == nil) != (theirErr == nil) {
			t.Errorf("%q: our error = %v, net/http error = %v", raw, ourErr, theirErr)
			continue
		}
		if ourErr != nil {
			continue
		}
		if ours.StatusCode != theirs.StatusCode {
			t.Errorf("%q: our status = %d, net/http status = %d", raw, ours.StatusCode, theirs.StatusCode)
		}
		ourBody, _ := io.ReadAll(ours.Body)
		theirBody, _ := io.ReadAll(theirs.Body)
		ours.Close()
		theirs.Body.Close()
		if string(ourBody) != string(theirBody) {
			t.Errorf("%q: our body = %q, net/http body = %q", raw, string(ourBody), string(theirBody))
		}
	}
}
