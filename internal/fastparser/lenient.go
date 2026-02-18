package fastparser

import (
	"bytes"
	"fmt"
	"strconv"
)

// ParseResult holds the result of lenient parsing.
type ParseResult struct {
	Request  *Request
	Response *Response
	Warnings []string
	Partial  bool
}

// LenientParser provides best-effort HTTP message parsing that never fails
// on malformed input. It extracts what it can and reports issues as warnings.
type LenientParser struct {
	data     []byte
	pos      int
	length   int
	line     int
	warnings []string
}

// NewLenientParser creates a new lenient parser for the given data.
func NewLenientParser(data []byte) *LenientParser {
	return &LenientParser{
		data:   data,
		pos:    0,
		length: len(data),
		line:   1,
	}
}

// Parse auto-detects and parses the message with best-effort extraction.
func (p *LenientParser) Parse() *ParseResult {
	result := &ParseResult{}

	if p.length == 0 {
		p.addWarning(1, "empty input")
		result.Partial = true
		return result
	}

	if bytes.HasPrefix(p.data, []byte("HTTP/")) {
		resp := p.parseResponseLenient()
		result.Response = resp
	} else {
		req := p.parseRequestLenient()
		result.Request = req
	}

	result.Warnings = p.warnings
	return result
}

func (p *LenientParser) parseRequestLenient() *Request {
	req := &Request{}

	// Parse request line
	line := p.readLineLenient()
	if line == nil {
		p.addWarning(1, "empty request, no start line found")
		return req
	}

	method, path, version := p.parseRequestLineLenient(line)
	req.Method = method
	req.Path = path
	req.Version = version

	// Parse headers
	req.Headers = p.parseHeadersLenient()

	// Parse body
	body, partial := p.parseBodyLenient(req.Headers)
	req.Body = body
	if partial {
		// Set partial on the result via a secondary mechanism — caller checks warnings
		p.addWarning(0, "message body is incomplete")
	}

	return req
}

func (p *LenientParser) parseResponseLenient() *Response {
	resp := &Response{}

	// Parse status line
	line := p.readLineLenient()
	if line == nil {
		p.addWarning(1, "empty response, no start line found")
		return resp
	}

	version, statusCode, reason := p.parseStatusLineLenient(line)
	resp.Version = version
	resp.StatusCode = statusCode
	resp.Reason = reason

	// Parse headers
	resp.Headers = p.parseHeadersLenient()

	// Parse body
	body, partial := p.parseBodyLenient(resp.Headers)
	resp.Body = body
	if partial {
		p.addWarning(0, "message body is incomplete")
	}

	return resp
}

func (p *LenientParser) parseRequestLineLenient(line []byte) (method, path, version string) {
	// Try to split "METHOD SP PATH SP VERSION"
	parts := bytes.Fields(line)

	switch len(parts) {
	case 0:
		p.addWarning(p.line-1, "empty request line")
		return "", "", "HTTP/1.1"
	case 1:
		// Just method, no path or version
		p.addWarning(p.line-1, "request line has only method, no path or version")
		return string(parts[0]), "/", "HTTP/1.1"
	case 2:
		// Method + path, missing version
		p.addWarning(p.line-1, "missing HTTP version in request-line, defaulting to HTTP/1.1")
		return string(parts[0]), string(parts[1]), "HTTP/1.1"
	default:
		// Normal: method path version (extra parts joined into path? No — version is last)
		return string(parts[0]), string(parts[1]), string(parts[2])
	}
}

func (p *LenientParser) parseStatusLineLenient(line []byte) (version string, statusCode int, reason string) {
	parts := bytes.Fields(line)

	switch len(parts) {
	case 0:
		p.addWarning(p.line-1, "empty status line")
		return "HTTP/1.1", 0, ""
	case 1:
		// Just version
		p.addWarning(p.line-1, "status line has only version, no status code")
		return string(parts[0]), 0, ""
	case 2:
		// Version + status code, no reason
		code, err := strconv.Atoi(string(parts[1]))
		if err != nil {
			p.addWarning(p.line-1, fmt.Sprintf("invalid status code %q, setting to 0", string(parts[1])))
			code = 0
		}
		return string(parts[0]), code, ""
	default:
		// Version + status code + reason (reason may contain spaces)
		code, err := strconv.Atoi(string(parts[1]))
		if err != nil {
			p.addWarning(p.line-1, fmt.Sprintf("invalid status code %q, setting to 0", string(parts[1])))
			code = 0
		}
		// Reconstruct reason from remaining parts
		reasonStart := bytes.Index(line, parts[1]) + len(parts[1])
		reason = string(bytes.TrimSpace(line[reasonStart:]))
		return string(parts[0]), code, reason
	}
}

func (p *LenientParser) parseHeadersLenient() []Header {
	var headers []Header

	for {
		if p.pos >= p.length {
			return headers
		}

		// Check for empty line (end of headers)
		if p.data[p.pos] == '\r' && p.pos+1 < p.length && p.data[p.pos+1] == '\n' {
			p.pos += 2
			p.line++
			return headers
		}
		if p.data[p.pos] == '\n' {
			p.pos++
			p.line++
			return headers
		}
		if p.data[p.pos] == '\r' {
			// Bare CR — treat as end of headers
			p.pos++
			p.line++
			return headers
		}

		line := p.readLineLenient()
		if line == nil {
			return headers
		}

		// Handle obs-fold (continuation lines)
		for p.pos < p.length && (p.data[p.pos] == ' ' || p.data[p.pos] == '\t') {
			cont := p.readLineLenient()
			if cont == nil {
				break
			}
			line = append(line, ' ')
			line = append(line, bytes.TrimLeft(cont, " \t")...)
		}

		// Parse "Key: Value" — lenient: accept whitespace before colon
		colon := bytes.IndexByte(line, ':')
		if colon < 0 {
			// No colon — check if the line looks like a bare hostname/host:port.
			// A common editor pattern is to write the host on its own line without
			// the "Host:" prefix (e.g. "example.com" or "api.example.com:8080").
			if isHostnameLike(line) {
				p.addWarning(p.line-1, fmt.Sprintf("bare hostname %q treated as implicit Host header", string(line)))
				headers = append(headers, Header{Key: "Host", Value: string(bytes.TrimSpace(line))})
			} else {
				p.addWarning(p.line-1, fmt.Sprintf("malformed header (no colon), skipped: %s", string(line)))
			}
			continue
		}

		key := string(bytes.TrimRight(line[:colon], " \t"))
		if key != string(line[:colon]) {
			p.addWarning(p.line-1, fmt.Sprintf("whitespace before colon in header name %q, accepted leniently", string(line[:colon])))
		}

		value := string(trimOWSBytes(line[colon+1:]))
		headers = append(headers, Header{Key: key, Value: value})
	}
}

func (p *LenientParser) parseBodyLenient(headers []Header) (body []byte, partial bool) {
	if p.pos >= p.length {
		return nil, false
	}

	// Check for chunked
	if isChunked(headers) {
		decoded, err := Dechunk(p.data[p.pos:])
		if err != nil {
			// Partial chunked decode — return what we can
			p.addWarning(0, fmt.Sprintf("chunked encoding error: %v, returning available data", err))
			// Try to extract whatever we got before the error
			remaining := p.data[p.pos:]
			return remaining, true
		}
		return decoded, false
	}

	// Read all available body bytes — Content-Length is treated as advisory
	// in lenient mode. A wrong Content-Length is a formatting inconsistency;
	// the body data that follows is still valid and must not be discarded.
	available := p.length - p.pos
	body = make([]byte, available)
	copy(body, p.data[p.pos:])
	p.pos = p.length

	cl := getContentLength(headers)
	if cl >= 0 && int64(available) != cl {
		p.addWarning(0, fmt.Sprintf("Content-Length declared %d, actual body is %d bytes", cl, available))
		// If actual is less than declared the message may have been truncated
		// in transit; signal that to the caller.
		if int64(available) < cl {
			return body, true
		}
	}

	return body, false
}

func (p *LenientParser) readLineLenient() []byte {
	if p.pos >= p.length {
		return nil
	}

	start := p.pos
	for p.pos < p.length {
		if p.data[p.pos] == '\r' && p.pos+1 < p.length && p.data[p.pos+1] == '\n' {
			line := p.data[start:p.pos]
			p.pos += 2
			p.line++
			return line
		}
		if p.data[p.pos] == '\n' {
			line := p.data[start:p.pos]
			p.pos++
			p.line++
			return line
		}
		if p.data[p.pos] == '\r' {
			// Bare CR — treat as line ending
			line := p.data[start:p.pos]
			p.pos++
			p.line++
			return line
		}
		p.pos++
	}

	// No line ending — return remaining data
	line := p.data[start:p.pos]
	return line
}

func (p *LenientParser) addWarning(line int, msg string) {
	if line > 0 {
		p.warnings = append(p.warnings, fmt.Sprintf("line %d: %s", line, msg))
	} else {
		p.warnings = append(p.warnings, msg)
	}
}

// trimOWSBytes trims SP/HTAB from both ends.
func trimOWSBytes(b []byte) []byte {
	for len(b) > 0 && (b[0] == ' ' || b[0] == '\t') {
		b = b[1:]
	}
	for len(b) > 0 && (b[len(b)-1] == ' ' || b[len(b)-1] == '\t') {
		b = b[:len(b)-1]
	}
	return b
}

// isHostnameLike reports whether b looks like a bare hostname or host:port,
// e.g. "example.com", "api.example.com:8080", "192.168.1.1".
//
// To avoid false-positives on arbitrary bare words (method names, etc.) the
// value must contain at least one dot OR an explicit port suffix (:\d+).
// Pattern: [a-zA-Z0-9][a-zA-Z0-9\-.]*(:\d+)?
func isHostnameLike(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	c0 := b[0]
	if !((c0 >= 'a' && c0 <= 'z') || (c0 >= 'A' && c0 <= 'Z') || (c0 >= '0' && c0 <= '9')) {
		return false
	}

	// Split off optional port suffix (colon followed entirely by digits).
	host := b
	hasPort := false
	if idx := bytes.LastIndexByte(b, ':'); idx > 0 {
		port := b[idx+1:]
		allDigits := len(port) > 0
		for _, ch := range port {
			if ch < '0' || ch > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			hasPort = true
			host = b[:idx]
		}
	}

	hasDot := false
	for _, ch := range host {
		switch {
		case (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-':
			// valid hostname char
		case ch == '.':
			hasDot = true
		default:
			return false // space, underscore, etc. — not a hostname
		}
	}

	return hasDot || hasPort
}
