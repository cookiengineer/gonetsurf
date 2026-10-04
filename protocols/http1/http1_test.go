package http1

import (
	"bufio"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gonetsurf/urls"
)

func TestReadResponseChunked(t *testing.T) {
	raw := "HTTP/1.1 200 OK\r\n" +
		"Transfer-Encoding: chunked\r\n" +
		"Trailer: X-Sum\r\n" +
		"\r\n" +
		"5\r\nhello\r\n" +
		"6\r\n world\r\n" +
		"0\r\n" +
		"X-Sum: 6\r\n" +
		"\r\n"
	response, err := ReadResponse(bufio.NewReader(strings.NewReader(raw)), nil)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "hello world" {
		t.Errorf("body = %q", string(body))
	}
	if !response.Chunked {
		t.Error("expected chunked")
	}
	if response.Trailers().Get("X-Sum") != "6" {
		t.Errorf("trailer X-Sum = %q", response.Trailers().Get("X-Sum"))
	}
}

func TestReadResponseContentLength(t *testing.T) {
	raw := "HTTP/1.1 200 OK\r\nContent-Length: 5\r\n\r\nhello"
	response, err := ReadResponse(bufio.NewReader(strings.NewReader(raw)), nil)
	if err != nil {
		t.Fatal(err)
	}
	if response.ContentLength != 5 {
		t.Fatalf("ContentLength = %d", response.ContentLength)
	}
	body, _ := io.ReadAll(response.Body)
	if string(body) != "hello" {
		t.Errorf("body = %q", string(body))
	}
}

func TestRejectWhitespaceBeforeColon(t *testing.T) {
	raw := "HTTP/1.1 200 OK\r\nBad Header : value\r\n\r\n"
	if _, err := ReadResponse(bufio.NewReader(strings.NewReader(raw)), nil); err == nil {
		t.Fatal("expected an error for whitespace before colon")
	}
}

func TestContentLengthAndTransferEncoding(t *testing.T) {
	raw := "HTTP/1.1 200 OK\r\nContent-Length: 5\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nhello\r\n0\r\n\r\n"
	response, err := ReadResponse(bufio.NewReader(strings.NewReader(raw)), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !response.Chunked {
		t.Error("Transfer-Encoding must take precedence over Content-Length")
	}
}

func TestRequestRoundTrip(t *testing.T) {
	parsed, _ := urls.Parse("https://example.com/a/b?c=d")
	request := &Request{Method: "GET", URL: parsed, Proto: "HTTP/1.1", Header: Header{}}
	var builder strings.Builder
	writer := bufio.NewWriter(&builder)
	if err := WriteRequest(writer, request); err != nil {
		t.Fatal(err)
	}
	read, err := ReadRequest(bufio.NewReader(strings.NewReader(builder.String())))
	if err != nil {
		t.Fatal(err)
	}
	if read.Method != "GET" || read.Header.Get("Host") != "example.com" {
		t.Fatalf("read request = %+v", read)
	}
}

func TestClientRedirects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/start" {
			http.Redirect(writer, request, "/final", http.StatusFound)
			return
		}
		writer.Header().Set("Content-Type", "text/plain")
		io.WriteString(writer, "done")
	}))
	defer server.Close()

	start, _ := urls.Parse(server.URL + "/start")
	client := &Client{Transport: &Transport{}}
	response, err := client.Do(t.Context(), &Request{Method: "GET", URL: start, Header: Header{}})
	if err != nil {
		t.Fatal(err)
	}
	defer response.Close()
	body, _ := io.ReadAll(response.Body)
	if string(body) != "done" {
		t.Errorf("body = %q", string(body))
	}
	if response.Request.URL.Pathname() != "/final" {
		t.Errorf("final path = %q", response.Request.URL.Pathname())
	}
}

func TestConnectionReuse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		io.WriteString(writer, "ok")
	}))
	defer server.Close()

	transport := &Transport{}
	client := &Client{Transport: transport}
	target, _ := urls.Parse(server.URL + "/")
	for count := 0; count < 3; count++ {
		response, err := client.Do(t.Context(), &Request{Method: "GET", URL: target, Header: Header{}})
		if err != nil {
			t.Fatal(err)
		}
		io.Copy(io.Discard, response.Body)
		response.Close()
	}
	if len(transport.idle) == 0 {
		t.Error("expected at least one pooled connection")
	}
	transport.CloseIdle()
}

func TestSetRange(t *testing.T) {
	request := &Request{Header: Header{}}
	SetRange(request, 10, 20)
	if got := request.Header.Get("Range"); got != "bytes=10-20" {
		t.Errorf("Range = %q", got)
	}
	SetIfNoneMatch(request, `"abc"`)
	if got := request.Header.Get("If-None-Match"); got != `"abc"` {
		t.Errorf("If-None-Match = %q", got)
	}
	_ = time.Second
}
