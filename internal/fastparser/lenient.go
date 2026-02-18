package fastparser

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
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

	// Normalize path: extract implicit Host from absolute-form URLs and bare
	// authority prefixes (e.g. "https://example.com/api" → "/api",
	// "example.com:8080/api" → "/api"). Returns the implied host authority or "".
	path, impliedHost := p.normalizePathLenient(path)

	req.Method = method
	req.Path = path
	req.Version = version

	// Parse headers
	req.Headers = p.parseHeadersLenient()

	// Inject the host extracted from the request-target if no Host header is
	// already present. If the user also supplied a bare host:port header line
	// (CR-3) that was converted to Host, we skip this to avoid duplicates.
	if impliedHost != "" {
		hasHost := false
		for _, h := range req.Headers {
			if eqFold(h.Key, "Host") {
				hasHost = true
				break
			}
		}
		if !hasHost {
			req.Headers = append([]Header{{Key: "Host", Value: impliedHost}}, req.Headers...)
		}
	}

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

// normalizePathLenient inspects the request-target for embedded host
// information and returns the normalized path plus the extracted host authority.
//
// Handled patterns:
//
//	Absolute-form:    "https://example.com:8080/api" → path="/api",   host="example.com:8080"
//	Absolute no path: "https://example.com"          → path="/",      host="example.com"
//	Bare host+port:   "example.com:8080/api"         → path="/api",   host="example.com:8080"
//	Bare hostname:    "example.com/api"              → path="/api",   host="example.com"
//
// Returns (path, "") when no host is embedded.
func (p *LenientParser) normalizePathLenient(path string) (normalizedPath, impliedHost string) {
	// Absolute-form: http:// or https://
	schemeLen := 0
	switch {
	case len(path) >= 8 && path[:8] == "https://":
		schemeLen = 8
	case len(path) >= 7 && path[:7] == "http://":
		schemeLen = 7
	}
	if schemeLen > 0 {
		rest := path[schemeLen:] // "authority/path" or just "authority"
		slashIdx := strings.IndexByte(rest, '/')
		var authority, urlPath string
		if slashIdx < 0 {
			authority = rest
			urlPath = "/"
		} else {
			authority = rest[:slashIdx]
			urlPath = rest[slashIdx:]
		}
		// Strip userinfo (user:pass@host) from the authority if present.
		if at := strings.LastIndexByte(authority, '@'); at >= 0 {
			authority = authority[at+1:]
		}
		if authority != "" {
			p.addWarning(1, fmt.Sprintf("absolute-form request-target: extracted Host %q, using path %q", authority, urlPath))
			return urlPath, authority
		}
		return urlPath, ""
	}

	// IPv6 literal prefix: "[::1]/api" or "[::1]:8080/api".
	if len(path) > 0 && path[0] == '[' {
		close := strings.IndexByte(path, ']')
		if close > 0 {
			bracket := path[:close+1] // "[::1]"
			rest := path[close+1:]    // "/api" or ":8080/api" or ""
			if strings.IndexByte(bracket, ':') > 0 {
				// Has at least one colon inside brackets — looks like IPv6.
				if len(rest) > 0 && rest[0] == '/' {
					// "[::1]/api"
					p.addWarning(1, fmt.Sprintf("request-target %q contains bare IPv6 host prefix, extracted Host %q, using path %q", path, bracket, rest))
					return rest, bracket
				}
				if len(rest) > 1 && rest[0] == ':' {
					// "[::1]:8080/api" — slashIdx is relative to rest[1:].
					slashIdx := strings.IndexByte(rest[1:], '/')
					if slashIdx >= 0 {
						portPart := rest[1 : 1+slashIdx] // "8080"
						urlPath := rest[1+slashIdx:]     // "/api/users" (includes leading /)
						if isPortStr(portPart) {
							authority := bracket + ":" + portPart
							p.addWarning(1, fmt.Sprintf("request-target %q contains bare IPv6 host prefix, extracted Host %q, using path %q", path, authority, urlPath))
							return urlPath, authority
						}
					}
				}
			}
		}
	}

	// Bare authority prefix: "host:port/path", "hostname.tld/path", or
	// "localhost:port/path" (bare word with port — no dot required when a
	// port is present, because the port itself disambiguates from method names).
	if len(path) > 0 {
		c0 := path[0]
		if (c0 >= 'a' && c0 <= 'z') || (c0 >= 'A' && c0 <= 'Z') || (c0 >= '0' && c0 <= '9') {
			slashIdx := strings.IndexByte(path, '/')
			if slashIdx > 0 {
				prefix := path[:slashIdx]
				rest := path[slashIdx:] // includes leading /
				if isHostnameLike([]byte(prefix)) {
					p.addWarning(1, fmt.Sprintf("request-target %q contains bare host prefix, extracted Host %q, using path %q", path, prefix, rest))
					return rest, prefix
				}
			}
		}
	}

	return path, ""
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

		// Parse "Key: Value" — lenient: accept whitespace before colon.

		// IPv6 literals start with '[' and contain colons inside the brackets,
		// which confuses the normal colon-splitting logic. Handle them first.
		if len(line) > 0 && line[0] == '[' {
			if h := parseIPv6HostLine(line); h != "" {
				p.addWarning(p.line-1, fmt.Sprintf("bare IPv6 address %q treated as implicit Host header", h))
				headers = append(headers, Header{Key: "Host", Value: h})
			} else {
				p.addWarning(p.line-1, fmt.Sprintf("malformed header (no colon), skipped: %s", string(line)))
			}
			continue
		}

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

		// CR-3: a bare "host:port" line split on its colon.
		//
		// Two cases are detected:
		//   a) key contains a dot (e.g. "example.com") — dot is the primary
		//      discriminator because HTTP header names never contain dots.
		//   b) key is a single-label name with no hyphen and all lowercase/digit
		//      chars (e.g. "localhost") — the all-digit port value provides
		//      sufficient disambiguation from real headers, which either contain
		//      hyphens (Content-Type) or start with an uppercase letter (Accept).
		if (isHostnameKeyStr(key) || isSingleLabelHost(key)) && isPortStr(value) {
			hostPort := key + ":" + value
			p.addWarning(p.line-1, fmt.Sprintf("bare host:port %q treated as implicit Host header", hostPort))
			headers = append(headers, Header{Key: "Host", Value: hostPort})
			continue
		}

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

// parseIPv6HostLine parses a raw header line that starts with '[' and returns
// the host authority string ("[::1]" or "[::1]:8080") if the line looks like
// a bare IPv6 address, or "" if it does not.
func parseIPv6HostLine(line []byte) string {
	if len(line) < 3 || line[0] != '[' {
		return ""
	}
	close := bytes.IndexByte(line, ']')
	if close < 0 {
		return ""
	}
	inner := line[1:close]
	// Must contain at least one colon to be IPv6 (not just "[word]").
	if bytes.IndexByte(inner, ':') < 0 {
		return ""
	}
	host := string(line[:close+1]) // "[::1]"
	rest := line[close+1:]
	if len(rest) == 0 {
		return host
	}
	// Optional port suffix ":digits"
	if len(rest) > 1 && rest[0] == ':' {
		port := rest[1:]
		if isPortStr(string(port)) {
			return host + ":" + string(port) // "[::1]:8080"
		}
	}
	return "" // unexpected suffix — not a bare host line
}

// isHostnameKeyStr reports whether s looks like a hostname suitable for use
// as the key part of a bare host:port header line (CR-3). Requirements:
// non-empty, at least one dot (dot is the discriminator since legitimate
// header field-names never contain dots in practice), and all characters
// are valid hostname chars [a-zA-Z0-9-.].
func isHostnameKeyStr(s string) bool {
	if s == "" {
		return false
	}
	hasDot := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-':
			// valid hostname char
		case c == '.':
			hasDot = true
		default:
			return false
		}
	}
	return hasDot
}

// isSingleLabelHost reports whether s looks like a single-label hostname
// (e.g. "localhost", "db", "myserver") — no dot, no hyphen, only lowercase
// letters and digits. Used in CR-3 when a key:value pair has an all-digit
// value: the numeric port disambiguates from legitimate HTTP headers, which
// either contain hyphens (Content-Type) or start with uppercase (Accept).
func isSingleLabelHost(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

// isPortStr reports whether s is a non-empty string of ASCII digits (a port number).
func isPortStr(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// isHostnameLike reports whether b looks like a bare hostname or host:port,
// e.g. "example.com", "api.example.com:8080", "192.168.1.1", "localhost:8080".
//
// To avoid false-positives on arbitrary bare words (method names, etc.) the
// value must contain at least one dot OR an explicit port suffix (:\d+).
// A bare word with a port (e.g. "localhost:8080") is accepted because the
// numeric port suffix unambiguously distinguishes it from a method name.
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
