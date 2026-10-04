package http1

import (
	"bufio"
	"io"
	"strings"
	"testing"
)

func FuzzReadResponse(f *testing.F) {
	seeds := []string{
		"",
		"HTTP/1.1 200 OK\r\nContent-Length: 5\r\n\r\nhello",
		"HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nhello\r\n0\r\n\r\n",
		"HTTP/1.1 200 OK\r\nContent-Length: 5\r\nTransfer-Encoding: chunked\r\n\r\n0\r\n\r\n",
		"HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\n\r\n",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		response, err := ReadResponse(bufio.NewReader(strings.NewReader(input)), nil)
		if err == nil {
			io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
			response.Close()
		}
		request, err := ReadRequest(bufio.NewReader(strings.NewReader(input)))
		if err == nil && request.Body != nil {
			io.Copy(io.Discard, io.LimitReader(request.Body, 1<<20))
			request.Body.Close()
		}
	})
}
