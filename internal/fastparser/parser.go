// Package fastparser implements a high-performance HTTP/1.1 message parser
// without AST construction. It scans bytes directly into Request/Response types.
package fastparser

import (
	"bytes"
	"fmt"
	"strconv"
)

// Request represents a parsed HTTP request.
type Request struct {
	Method  string
	Path    string
	Version string
	Scheme  string // "https", "http", or "" — populated from absolute-form targets
	Headers []Header
	Body    []byte
}

// Response represents a parsed HTTP response.
type Response struct {
	Version    string
	StatusCode int
	Reason     string
	Headers    []Header
	Body       []byte
}

// Header is a key-value pair.
type Header struct {
	Key   string
	Value string
}

// Parser implements a zero-allocation HTTP/1.1 parser that scans bytes directly.
type Parser struct {
	data   []byte
	pos    int
	length int
	line   int // 1-indexed line number for error reporting
}

// NewParser creates a new fast parser for the given data.
func NewParser(data []byte) *Parser {
	return &Parser{
		data:   data,
		pos:    0,
		length: len(data),
		line:   1,
	}
}

// initParser initializes a parser in-place (stack-friendly, avoids heap alloc).
func initParser(p *Parser, data []byte) {
	p.data = data
	p.pos = 0
	p.length = len(data)
	p.line = 1
}

// ParseRequest parses an HTTP request message.
func (p *Parser) ParseRequest() (*Request, error) {
	method, path, version, err := p.parseRequestLine()
	if err != nil {
		return nil, err
	}

	headers, err := p.parseHeaders()
	if err != nil {
		return nil, err
	}

	wasChunked := isChunked(headers)
	body, err := p.parseBody(headers)
	if err != nil {
		return nil, err
	}
	if wasChunked {
		headers = normalizeChunkedHeaders(headers, len(body))
	}

	return &Request{
		Method:  method,
		Path:    path,
		Version: version,
		Headers: headers,
		Body:    body,
	}, nil
}

// ParseResponse parses an HTTP response message.
func (p *Parser) ParseResponse() (*Response, error) {
	version, statusCode, reason, err := p.parseStatusLine()
	if err != nil {
		return nil, err
	}

	headers, err := p.parseHeaders()
	if err != nil {
		return nil, err
	}

	wasChunked := isChunked(headers)
	body, err := p.parseBody(headers)
	if err != nil {
		return nil, err
	}
	if wasChunked {
		headers = normalizeChunkedHeaders(headers, len(body))
	}

	return &Response{
		Version:    version,
		StatusCode: statusCode,
		Reason:     reason,
		Headers:    headers,
		Body:       body,
	}, nil
}

// parseRequestLine parses "METHOD SP PATH SP VERSION CRLF".
func (p *Parser) parseRequestLine() (method, path, version string, err error) {
	line, err := p.readLine()
	if err != nil {
		return "", "", "", p.errorf("missing request line")
	}

	// Find first SP
	sp1 := bytes.IndexByte(line, ' ')
	if sp1 < 0 {
		return "", "", "", p.errorf("malformed request line: no method separator")
	}
	method = internMethod(line[:sp1])

	rest := line[sp1+1:]

	// Find second SP (separating path from version)
	sp2 := bytes.IndexByte(rest, ' ')
	if sp2 < 0 {
		return "", "", "", p.errorf("malformed request line: no version separator")
	}
	path = string(rest[:sp2])
	version = internVersion(rest[sp2+1:])

	if method == "" {
		return "", "", "", p.errorf("empty request method")
	}
	if path == "" {
		return "", "", "", p.errorf("empty request path")
	}

	return method, path, version, nil
}

// parseStatusLine parses "VERSION SP STATUS SP REASON CRLF".
func (p *Parser) parseStatusLine() (version string, statusCode int, reason string, err error) {
	line, err := p.readLine()
	if err != nil {
		return "", 0, "", p.errorf("missing status line")
	}

	// Find first SP
	sp1 := bytes.IndexByte(line, ' ')
	if sp1 < 0 {
		return "", 0, "", p.errorf("malformed status line: no version separator")
	}
	version = internVersion(line[:sp1])

	rest := line[sp1+1:]

	// Find second SP (separating status code from reason)
	sp2 := bytes.IndexByte(rest, ' ')
	if sp2 < 0 {
		// Allow status line with no reason phrase: "HTTP/1.1 200"
		code, convErr := strconv.Atoi(string(rest))
		if convErr != nil {
			return "", 0, "", p.errorf("invalid status code: %s", string(rest))
		}
		return version, code, "", nil
	}

	code, convErr := strconv.Atoi(string(rest[:sp2]))
	if convErr != nil {
		return "", 0, "", p.errorf("invalid status code: %s", string(rest[:sp2]))
	}
	reason = internReason(rest[sp2+1:])

	return version, code, reason, nil
}

// parseHeaders parses header lines until empty line (CRLF CRLF).
// Pre-allocates the headers slice to avoid growth allocations.
func (p *Parser) parseHeaders() ([]Header, error) {
	headers := make([]Header, 0, 8)

	for {
		if p.pos >= p.length {
			// End of data without empty line — headers section is complete
			return headers, nil
		}

		// Check for empty line (end of headers)
		if p.pos < p.length && p.data[p.pos] == '\r' && p.pos+1 < p.length && p.data[p.pos+1] == '\n' {
			p.pos += 2
			p.line++
			return headers, nil
		}
		if p.pos < p.length && p.data[p.pos] == '\n' {
			p.pos++
			p.line++
			return headers, nil
		}

		line, err := p.readLine()
		if err != nil {
			return headers, nil
		}

		// Handle obs-fold (continuation line starting with SP/HTAB)
		for p.pos < p.length && (p.data[p.pos] == ' ' || p.data[p.pos] == '\t') {
			cont, contErr := p.readLine()
			if contErr != nil {
				break
			}
			// Replace obs-fold with single SP
			line = append(line, ' ')
			line = append(line, bytes.TrimLeft(cont, " \t")...)
		}

		// Parse "Key: Value"
		colon := bytes.IndexByte(line, ':')
		if colon < 0 {
			return nil, p.errorf("malformed header line (no colon): %s", string(line))
		}

		keyBytes := line[:colon]

		// RFC 9112: no whitespace between field-name and colon
		if colon > 0 && (line[colon-1] == ' ' || line[colon-1] == '\t') {
			return nil, p.errorf("whitespace before colon in header name: %s", string(keyBytes))
		}

		key := internHeaderName(keyBytes)
		value := string(trimOWS(line[colon+1:]))
		headers = append(headers, Header{Key: key, Value: value})
	}
}

// parseBody determines and reads the message body.
// Body length determination per RFC 9112:
// 1. Transfer-Encoding: chunked → parse chunk frames
// 2. Content-Length → read exactly N bytes
// 3. Neither → remaining bytes (connection-close semantics)
func (p *Parser) parseBody(headers []Header) ([]byte, error) {
	// Check for chunked transfer encoding
	if isChunked(headers) {
		return Dechunk(p.data[p.pos:])
	}

	// Check for Content-Length
	cl := getContentLength(headers)
	if cl >= 0 {
		if p.pos+int(cl) > p.length {
			return nil, p.errorf("body truncated: expected %d bytes but only %d available", cl, p.length-p.pos)
		}
		body := make([]byte, cl)
		copy(body, p.data[p.pos:p.pos+int(cl)])
		p.pos += int(cl)
		return body, nil
	}

	// No Content-Length, no chunked: remaining bytes are body
	if p.pos >= p.length {
		return nil, nil
	}
	body := make([]byte, p.length-p.pos)
	copy(body, p.data[p.pos:])
	p.pos = p.length
	return body, nil
}

// readLine reads bytes until CRLF or LF, advancing pos.
// Returns the line content (without line ending).
func (p *Parser) readLine() ([]byte, error) {
	if p.pos >= p.length {
		return nil, fmt.Errorf("unexpected end of input at line %d", p.line)
	}

	start := p.pos
	for p.pos < p.length {
		if p.data[p.pos] == '\r' && p.pos+1 < p.length && p.data[p.pos+1] == '\n' {
			line := p.data[start:p.pos]
			p.pos += 2
			p.line++
			return line, nil
		}
		if p.data[p.pos] == '\n' {
			line := p.data[start:p.pos]
			p.pos++
			p.line++
			return line, nil
		}
		p.pos++
	}

	// No line ending — return remaining data
	line := p.data[start:p.pos]
	return line, nil
}

// trimOWS trims optional whitespace (SP and HTAB) from both ends of b.
func trimOWS(b []byte) []byte {
	for len(b) > 0 && (b[0] == ' ' || b[0] == '\t') {
		b = b[1:]
	}
	for len(b) > 0 && (b[len(b)-1] == ' ' || b[len(b)-1] == '\t') {
		b = b[:len(b)-1]
	}
	return b
}

// isChunked checks if headers contain Transfer-Encoding: chunked.
func isChunked(headers []Header) bool {
	for _, h := range headers {
		if eqFold(h.Key, "Transfer-Encoding") {
			if containsFold(h.Value, "chunked") {
				return true
			}
		}
	}
	return false
}

// normalizeChunkedHeaders removes Transfer-Encoding after chunked decoding and
// sets Content-Length to the decoded body length.
//
// Chunked encoding is a transport-level framing mechanism. Once the body is
// fully decoded, the struct should reflect the logical message: a known-length
// body with Content-Length, not a chunked stream. This makes the struct
// self-consistent for re-marshaling (e.g., in HTTP client/server use).
func normalizeChunkedHeaders(headers []Header, bodyLen int) []Header {
	out := headers[:0:len(headers)]
	out = out[:0]
	hasContentLength := false
	for _, h := range headers {
		if eqFold(h.Key, "Transfer-Encoding") {
			// Strip "chunked" from the value; drop the header if nothing remains.
			stripped := stripChunked(h.Value)
			if stripped != "" {
				out = append(out, Header{Key: h.Key, Value: stripped})
			}
			continue
		}
		if eqFold(h.Key, "Content-Length") {
			// Replace with the decoded body length.
			out = append(out, Header{Key: h.Key, Value: strconv.Itoa(bodyLen)})
			hasContentLength = true
			continue
		}
		out = append(out, h)
	}
	if !hasContentLength {
		out = append(out, Header{Key: "Content-Length", Value: strconv.Itoa(bodyLen)})
	}
	return out
}

// stripChunked removes "chunked" from a Transfer-Encoding value and returns
// the remainder (trimmed). Returns "" if chunked was the only encoding.
func stripChunked(value string) string {
	// Fast path: value is exactly "chunked"
	if eqFold(value, "chunked") {
		return ""
	}
	// Multiple encodings: remove the "chunked" token.
	// Encodings are comma-separated; chunked must be the last per RFC 9112 §6.1.
	result := ""
	for _, part := range splitComma(value) {
		part = trimString(part)
		if !eqFold(part, "chunked") {
			if result != "" {
				result += ", "
			}
			result += part
		}
	}
	return result
}

// splitComma splits a comma-separated string into parts.
func splitComma(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// trimString trims leading and trailing ASCII whitespace.
func trimString(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

// getContentLength returns the Content-Length value, or -1 if absent/invalid.
func getContentLength(headers []Header) int64 {
	for _, h := range headers {
		if eqFold(h.Key, "Content-Length") {
			v := bytes.TrimSpace([]byte(h.Value))
			n, err := strconv.ParseInt(string(v), 10, 64)
			if err != nil {
				return -1
			}
			return n
		}
	}
	return -1
}

// eqFold is a fast ASCII case-insensitive string comparison.
func eqFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

// containsFold checks if haystack contains needle (case-insensitive).
func containsFold(haystack, needle string) bool {
	hl, nl := len(haystack), len(needle)
	if nl > hl {
		return false
	}
	for i := 0; i <= hl-nl; i++ {
		if eqFold(haystack[i:i+nl], needle) {
			return true
		}
	}
	return false
}

func (p *Parser) errorf(format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	return fmt.Errorf("http: parse error at line %d: %s", p.line, msg)
}
