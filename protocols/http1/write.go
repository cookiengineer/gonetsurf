package http1

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

// WriteRequest serializes a request, preserving header order and casing. The
// Host header is placed first when absent.
func WriteRequest(writer *bufio.Writer, request *Request) error {
	target := "/"
	if request.URL != nil {
		target = request.URL.Pathname()
		if request.URL.Search() != "" {
			target += request.URL.Search()
		}
	}
	proto := request.Proto
	if proto == "" {
		proto = "HTTP/1.1"
	}
	if _, err := writer.WriteString(request.Method + " " + target + " " + proto + "\r\n"); err != nil {
		return err
	}
	header := request.Header.Clone()
	if !header.Has("Host") && request.URL != nil {
		header = append(Header{{Name: "Host", Value: request.URL.HostWithPort()}}, header...)
	}
	for _, field := range header {
		if !isToken(field.Name) {
			return ErrInvalidMessage
		}
		if strings.ContainsAny(field.Value, "\r\n") {
			return ErrInvalidMessage
		}
		if _, err := writer.WriteString(field.Name + ": " + field.Value + "\r\n"); err != nil {
			return err
		}
	}
	if _, err := writer.WriteString("\r\n"); err != nil {
		return err
	}
	if request.Body != nil {
		if _, err := io.Copy(writer, request.Body); err != nil {
			return err
		}
	}
	return writer.Flush()
}

// WriteResponse serializes a response. It is primarily used by tests.
func WriteResponse(writer *bufio.Writer, response *Response) error {
	proto := response.Proto
	if proto == "" {
		proto = "HTTP/1.1"
	}
	statusLine := proto + " " + strconv.Itoa(response.StatusCode)
	if response.Reason != "" {
		statusLine += " " + response.Reason
	}
	if _, err := writer.WriteString(statusLine + "\r\n"); err != nil {
		return err
	}
	for _, field := range response.Header {
		if _, err := writer.WriteString(field.Name + ": " + field.Value + "\r\n"); err != nil {
			return err
		}
	}
	if _, err := writer.WriteString("\r\n"); err != nil {
		return err
	}
	if response.Body != nil {
		if _, err := io.Copy(writer, response.Body); err != nil {
			return err
		}
	}
	return writer.Flush()
}
