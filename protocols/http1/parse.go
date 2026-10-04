package http1

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"gonetsurf/urls"
)

// Framing errors.
var (
	ErrInvalidMessage = errors.New("http1: invalid message")
	ErrHeaderTooLarge = errors.New("http1: header section too large")
	ErrLineTooLong    = errors.New("http1: line too long")
	ErrInvalidFraming = errors.New("http1: invalid body framing")
)

const (
	maxLineLength   = 8192
	maxHeaderFields = 200
)

// ReadRequest reads and parses an HTTP/1.1 request.
func ReadRequest(reader *bufio.Reader) (*Request, error) {
	line, err := readLine(reader, maxLineLength)
	if err != nil {
		return nil, err
	}
	parts := strings.SplitN(line, " ", 3)
	if len(parts) != 3 {
		return nil, ErrInvalidMessage
	}
	request := &Request{Method: parts[0], Proto: parts[2]}
	target := parts[1]
	switch {
	case target == "*":
		// server-wide OPTIONS; leave the URL nil.
	case strings.HasPrefix(target, "/"):
		request.URL, err = urls.Parse("http://placeholder" + target)
		if err != nil {
			return nil, ErrInvalidMessage
		}
	default:
		request.URL, err = urls.Parse(target)
		if err != nil {
			return nil, ErrInvalidMessage
		}
	}
	if request.Header, err = readHeader(reader); err != nil {
		return nil, err
	}
	body, length, chunked, err := framing(reader, request.Header, request.Method, 0, request.Proto)
	if err != nil {
		return nil, err
	}
	request.Body = body
	_ = length
	_ = chunked
	return request, nil
}

// ReadResponse reads and parses an HTTP/1.1 response for the given request.
func ReadResponse(reader *bufio.Reader, request *Request) (*Response, error) {
	line, err := readLine(reader, maxLineLength)
	if err != nil {
		return nil, err
	}
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 2 {
		return nil, ErrInvalidMessage
	}
	statusCode, err := strconv.Atoi(parts[1])
	if err != nil || statusCode < 100 || statusCode > 599 {
		return nil, ErrInvalidMessage
	}
	response := &Response{Proto: parts[0], StatusCode: statusCode, Request: request}
	if len(parts) == 3 {
		response.Reason = parts[2]
	}
	if response.Header, err = readHeader(reader); err != nil {
		return nil, err
	}
	method := ""
	if request != nil {
		method = request.Method
	}
	body, length, chunked, err := framing(reader, response.Header, method, statusCode, response.Proto)
	if err != nil {
		return nil, err
	}
	response.Body = body
	response.ContentLength = length
	response.Chunked = chunked
	if chunked {
		if chunkedBody, ok := body.(*chunkedReader); ok {
			response.chunked = chunkedBody
		}
	}
	return response, nil
}

func readLine(reader *bufio.Reader, limit int) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF && line == "" {
			return "", io.EOF
		}
		if len(line) > limit {
			return "", ErrLineTooLong
		}
		if len(line) > 0 {
			return trimLineEnding(line), nil
		}
		return "", err
	}
	if len(line) > limit {
		return "", ErrLineTooLong
	}
	return trimLineEnding(line), nil
}

func trimLineEnding(line string) string {
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	return line
}

func readHeader(reader *bufio.Reader) (Header, error) {
	var header Header
	for count := 0; ; count++ {
		if count > maxHeaderFields {
			return nil, ErrHeaderTooLarge
		}
		line, err := readLine(reader, maxLineLength)
		if err != nil {
			return nil, err
		}
		if line == "" {
			return header, nil
		}
		if line[0] == ' ' || line[0] == '\t' {
			// obs-fold: append to the previous field value with a space.
			if len(header) == 0 {
				return nil, ErrInvalidMessage
			}
			header[len(header)-1].Value += " " + strings.Trim(line, " \t")
			continue
		}
		colon := strings.IndexByte(line, ':')
		if colon <= 0 {
			return nil, ErrInvalidMessage
		}
		name := line[:colon]
		if !isToken(name) {
			return nil, ErrInvalidMessage
		}
		value := strings.Trim(line[colon+1:], " \t")
		if strings.ContainsAny(value, "\r\n") {
			return nil, ErrInvalidMessage
		}
		header = append(header, Field{Name: name, Value: value})
	}
}

func framing(reader *bufio.Reader, header Header, method string, status int, proto string) (io.ReadCloser, int64, bool, error) {
	if method == "HEAD" || (status >= 100 && status < 200) || status == 204 || status == 304 {
		return io.NopCloser(strings.NewReader("")), 0, false, nil
	}

	transferEncoding := strings.Join(header.Values("Transfer-Encoding"), ",")
	if transferEncoding != "" {
		codings := splitList(transferEncoding)
		if len(codings) > 0 && strings.EqualFold(codings[len(codings)-1], "chunked") {
			return newChunkedReader(reader), -1, true, nil
		}
		if method == "" {
			// A response without chunked is delimited by connection close.
			return io.NopCloser(reader), -1, false, nil
		}
		return nil, 0, false, ErrInvalidFraming
	}

	contentLengths := header.Values("Content-Length")
	if len(contentLengths) > 0 {
		var length int64 = -1
		for _, value := range contentLengths {
			parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
			if err != nil || parsed < 0 {
				return nil, 0, false, ErrInvalidFraming
			}
			if length == -1 {
				length = parsed
			} else if length != parsed {
				return nil, 0, false, ErrInvalidFraming
			}
		}
		return io.NopCloser(io.LimitReader(reader, length)), length, false, nil
	}

	if method != "" {
		return io.NopCloser(strings.NewReader("")), 0, false, nil
	}
	return io.NopCloser(reader), -1, false, nil
}

func splitList(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// ParseChunkSize parses a chunk-size line (hex size and optional extensions).
func ParseChunkSize(line string) (int64, error) {
	size := line
	if semicolon := strings.IndexByte(line, ';'); semicolon >= 0 {
		size = line[:semicolon]
	}
	size = strings.TrimSpace(size)
	if size == "" {
		return 0, fmt.Errorf("http1: empty chunk size")
	}
	value, err := strconv.ParseInt(size, 16, 64)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("http1: invalid chunk size %q", size)
	}
	return value, nil
}
