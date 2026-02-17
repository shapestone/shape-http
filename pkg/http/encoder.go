package http

import "strconv"

// appendRequest serializes a Request to HTTP/1.1 wire format.
// It appends "METHOD PATH VERSION\r\n" followed by headers and body.
func appendRequest(buf []byte, req *Request) ([]byte, error) {
	if req.Method == "" {
		return nil, &ParseError{Message: "request method is empty"}
	}
	if req.Path == "" {
		return nil, &ParseError{Message: "request path is empty"}
	}

	version := req.Version
	if version == "" {
		version = "HTTP/1.1"
	}

	buf = appendRequestLine(buf, req.Method, req.Path, version)
	buf = appendHeaders(buf, req.Headers)

	// Auto-set Content-Length if body present and header absent
	if len(req.Body) > 0 && req.Headers.Get("Content-Length") == "" && !req.Headers.IsChunked() {
		buf = append(buf, "Content-Length: "...)
		buf = strconv.AppendInt(buf, int64(len(req.Body)), 10)
		buf = appendCRLF(buf)
	}

	buf = appendCRLF(buf) // empty line before body
	if len(req.Body) > 0 {
		buf = append(buf, req.Body...)
	}

	return buf, nil
}

// appendResponse serializes a Response to HTTP/1.1 wire format.
// It appends "VERSION STATUS REASON\r\n" followed by headers and body.
func appendResponse(buf []byte, resp *Response) []byte {
	version := resp.Version
	if version == "" {
		version = "HTTP/1.1"
	}

	buf = appendStatusLine(buf, version, resp.StatusCode, resp.Reason)
	buf = appendHeaders(buf, resp.Headers)

	// Auto-set Content-Length if body present and header absent
	if len(resp.Body) > 0 && resp.Headers.Get("Content-Length") == "" && !resp.Headers.IsChunked() {
		buf = append(buf, "Content-Length: "...)
		buf = strconv.AppendInt(buf, int64(len(resp.Body)), 10)
		buf = appendCRLF(buf)
	}

	buf = appendCRLF(buf) // empty line before body
	if len(resp.Body) > 0 {
		buf = append(buf, resp.Body...)
	}

	return buf
}

// appendHeaders appends all headers in "Key: Value\r\n" format.
func appendHeaders(buf []byte, headers Headers) []byte {
	for _, h := range headers {
		buf = append(buf, h.Key...)
		buf = append(buf, ':', ' ')
		buf = append(buf, h.Value...)
		buf = appendCRLF(buf)
	}
	return buf
}
