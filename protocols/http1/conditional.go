package http1

import "strconv"

// SetIfNoneMatch sets the If-None-Match request header.
func SetIfNoneMatch(request *Request, etag string) {
	if etag != "" {
		request.Header.Set("If-None-Match", etag)
	}
}

// SetIfModifiedSince sets the If-Modified-Since request header.
func SetIfModifiedSince(request *Request, httpDate string) {
	if httpDate != "" {
		request.Header.Set("If-Modified-Since", httpDate)
	}
}

// SetRange sets a byte-range request header.
func SetRange(request *Request, start, end int64) {
	if start < 0 {
		return
	}
	if end < start {
		request.Header.Set("Range", "bytes="+strconv.FormatInt(start, 10)+"-")
		return
	}
	request.Header.Set("Range", "bytes="+strconv.FormatInt(start, 10)+"-"+strconv.FormatInt(end, 10))
}

// SetIfRange sets the If-Range request header.
func SetIfRange(request *Request, validator string) {
	if validator != "" {
		request.Header.Set("If-Range", validator)
	}
}

// IsNotModified reports whether the response is a 304 Not Modified.
func (response *Response) IsNotModified() bool {
	return response.StatusCode == 304
}

// IsPartialContent reports whether the response is a 206 Partial Content.
func (response *Response) IsPartialContent() bool {
	return response.StatusCode == 206
}
