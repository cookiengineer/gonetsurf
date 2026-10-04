package http1

import (
	"bufio"
	"io"
	"strings"
)

type chunkedReader struct {
	reader    *bufio.Reader
	remaining int64
	done      bool
	started   bool
	needCRLF  bool
	trailer   Header
}

func newChunkedReader(reader *bufio.Reader) *chunkedReader {
	return &chunkedReader{reader: reader}
}

func (chunked *chunkedReader) Read(target []byte) (int, error) {
	if chunked.done {
		return 0, io.EOF
	}
	if chunked.remaining == 0 {
		if err := chunked.nextChunk(); err != nil {
			return 0, err
		}
		if chunked.done {
			return 0, io.EOF
		}
	}
	if int64(len(target)) > chunked.remaining {
		target = target[:chunked.remaining]
	}
	count, err := chunked.reader.Read(target)
	chunked.remaining -= int64(count)
	if chunked.remaining == 0 {
		chunked.needCRLF = true
	}
	return count, err
}

func (chunked *chunkedReader) nextChunk() error {
	if chunked.needCRLF {
		if _, err := readLine(chunked.reader, maxLineLength); err != nil {
			return err
		}
		chunked.needCRLF = false
	}
	if chunked.started {
		// A CRLF separates chunks and was already consumed above.
	}
	line, err := readLine(chunked.reader, maxLineLength)
	if err != nil {
		return err
	}
	size, err := ParseChunkSize(line)
	if err != nil {
		return err
	}
	chunked.started = true
	if size == 0 {
		return chunked.readTrailers()
	}
	chunked.remaining = size
	return nil
}

func (chunked *chunkedReader) readTrailers() error {
	for {
		line, err := readLine(chunked.reader, maxLineLength)
		if err != nil {
			return err
		}
		if line == "" {
			break
		}
		colon := strings.IndexByte(line, ':')
		if colon <= 0 {
			return ErrInvalidMessage
		}
		chunked.trailer = append(chunked.trailer, Field{
			Name:  line[:colon],
			Value: strings.Trim(line[colon+1:], " \t"),
		})
	}
	chunked.done = true
	return nil
}

func (chunked *chunkedReader) Close() error {
	return nil
}
