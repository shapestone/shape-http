package fastparser

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
)

// ParseCurl parses a curl command string and returns a ParseResult with
// best-effort extraction, matching the output format of Parse().
//
// It never errors on malformed input — issues are reported as Warnings.
func ParseCurl(cmd string) *ParseResult {
	cp := &curlParser{}
	result := cp.parse(cmd)
	result.Warnings = cp.warnings
	return result
}

type curlParser struct {
	warnings []string
}

func (cp *curlParser) warn(msg string) {
	cp.warnings = append(cp.warnings, msg)
}

func (cp *curlParser) parse(cmd string) *ParseResult {
	result := &ParseResult{}

	if strings.TrimSpace(cmd) == "" {
		cp.warn("empty curl command")
		result.Partial = true
		return result
	}

	// Normalize backslash line continuations before tokenizing.
	cmd = strings.ReplaceAll(cmd, "\\\r\n", " ")
	cmd = strings.ReplaceAll(cmd, "\\\n", " ")

	tokens, err := shellSplit(cmd)
	if err != nil {
		cp.warn(fmt.Sprintf("malformed curl command: %v", err))
		result.Partial = true
		return result
	}

	if len(tokens) == 0 {
		cp.warn("empty curl command")
		result.Partial = true
		return result
	}

	// Tolerate an optional leading "curl" token.
	if strings.EqualFold(tokens[0], "curl") {
		tokens = tokens[1:]
	}

	// Expand compound short flags like -sS → [-s, -S] before the main loop.
	tokens = expandShortFlags(tokens)

	var (
		method         string
		rawURL         string
		version        = "HTTP/1.1"
		headers        []Header
		dataParts      []string
		formFields     []string
		urlEncFields   []string
		explicitMethod bool
	)

	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]

		// Helper: consume the next token as an argument.
		next := func() (string, bool) {
			if i+1 < len(tokens) {
				i++
				return tokens[i], true
			}
			return "", false
		}

		switch tok {
		// Method
		case "-X", "--request":
			if v, ok := next(); ok {
				method = strings.ToUpper(v)
				explicitMethod = true
			}

		// Headers
		case "-H", "--header":
			if v, ok := next(); ok {
				headers = append(headers, parseCurlHeader(v))
			}

		// Body data — multiple -d flags are joined with "&" (curl behaviour).
		case "-d", "--data", "--data-raw", "--data-binary", "--data-ascii":
			if v, ok := next(); ok {
				if strings.HasPrefix(v, "@") {
					cp.warn(fmt.Sprintf("file upload %q is not supported, body skipped", v))
				} else {
					dataParts = append(dataParts, v)
				}
			}

		// Multipart form data
		case "-F", "--form":
			if v, ok := next(); ok {
				formFields = append(formFields, v)
			}

		// URL-encoded form data
		case "--data-urlencode":
			if v, ok := next(); ok {
				urlEncFields = append(urlEncFields, v)
			}

		// Cookie header.
		case "-b", "--cookie":
			if v, ok := next(); ok {
				headers = append(headers, Header{Key: "Cookie", Value: v})
			}

		// Basic auth — convert to Authorization: Basic header.
		case "-u", "--user":
			if v, ok := next(); ok {
				if !strings.ContainsRune(v, ':') {
					cp.warn(fmt.Sprintf("-u %q: no colon found; encoding username only (password was not provided)", v))
				}
				encoded := base64.StdEncoding.EncodeToString([]byte(v))
				headers = append(headers, Header{Key: "Authorization", Value: "Basic " + encoded})
			}

		// HTTP version
		case "--http2", "--http2-prior-knowledge":
			version = "HTTP/2"
		case "--http3":
			version = "HTTP/3"
		case "--http1.0":
			version = "HTTP/1.0"
		case "--http1.1":
			version = "HTTP/1.1"

		// -I / --head implies HEAD method.
		case "-I", "--head":
			if !explicitMethod {
				method = "HEAD"
			}

		// Flags that are silently ignored (no argument).
		case "-v", "--verbose",
			"-s", "--silent",
			"-S", "--show-error",
			"-L", "--location",
			"--compressed",
			"-k", "--insecure",
			"-i", "--include",
			"-O", // write to file named by remote
			"-g", "--globoff",
			"--no-keepalive",
			"--fail", "-f",
			"--no-progress-meter",
			"-#", "--progress-bar":
			// silently skip

		// Flags that are silently ignored (consume one argument).
		case "-o", "--output",
			"-m", "--max-time",
			"--connect-timeout",
			"-A", "--user-agent",
			"--proxy", "-x",
			"--cert", "--key", "--cacert",
			"--resolve",
			"-e", "--referer",
			"--limit-rate",
			"-w", "--write-out",
			"--retry",
			"--dns-servers",
			"--interface",
			"--local-port",
			"--max-redirs":
			next() // consume and discard

		default:
			if strings.HasPrefix(tok, "-") {
				cp.warn(fmt.Sprintf("unknown curl flag %q, skipping", tok))
			} else {
				// Positional argument — the URL.
				if rawURL == "" {
					rawURL = tok
				} else {
					cp.warn(fmt.Sprintf("unexpected positional argument %q, skipping", tok))
				}
			}
		}
	}

	if rawURL == "" {
		cp.warn("no URL found in curl command")
		result.Partial = true
		return result
	}

	// Build body from the first non-empty body source.
	var body []byte
	var autoContentType string

	switch {
	case len(formFields) > 0:
		b, boundary := buildMultipartForm(formFields, cp)
		body = b
		autoContentType = "multipart/form-data; boundary=" + boundary
	case len(urlEncFields) > 0:
		body = []byte(buildURLEncoded(urlEncFields))
		autoContentType = "application/x-www-form-urlencoded"
	case len(dataParts) > 0:
		body = []byte(strings.Join(dataParts, "&"))
	}

	// Default method: GET, or POST when a body is present.
	if method == "" {
		if len(body) > 0 {
			method = "POST"
		} else {
			method = "GET"
		}
	}

	// Parse the URL into scheme, host, path components.
	scheme, host, path := parseCurlURL(rawURL)

	// Inject Host header (prepend so it appears first, matching lenient behaviour).
	if host != "" && !curlHeadersHas(headers, "Host") {
		headers = append([]Header{{Key: "Host", Value: host}}, headers...)
	}

	// Auto Content-Type for form bodies (only when not explicitly set).
	if autoContentType != "" && !curlHeadersHas(headers, "Content-Type") {
		headers = append(headers, Header{Key: "Content-Type", Value: autoContentType})
	}

	// Auto Content-Length when a body is present and the header is absent.
	if len(body) > 0 && !curlHeadersHas(headers, "Content-Length") {
		headers = append(headers, Header{Key: "Content-Length", Value: fmt.Sprintf("%d", len(body))})
	}

	result.Request = &Request{
		Method:  method,
		Path:    path,
		Version: version,
		Scheme:  scheme,
		Headers: headers,
		Body:    body,
	}
	return result
}

// parseCurlHeader splits "Key: Value" on the first colon.
func parseCurlHeader(s string) Header {
	colon := strings.IndexByte(s, ':')
	if colon < 0 {
		return Header{Key: s}
	}
	return Header{
		Key:   strings.TrimRight(s[:colon], " \t"),
		Value: strings.TrimLeft(s[colon+1:], " \t"),
	}
}

// parseCurlURL extracts (scheme, host, path) from a raw URL string.
// If no scheme is found the entire string is returned as the path.
func parseCurlURL(rawURL string) (scheme, host, path string) {
	// Strip URL fragment (#...) — fragments are never sent in HTTP requests.
	if i := strings.IndexByte(rawURL, '#'); i >= 0 {
		rawURL = rawURL[:i]
	}

	switch {
	case strings.HasPrefix(rawURL, "https://"):
		scheme = "https"
		rawURL = rawURL[8:]
	case strings.HasPrefix(rawURL, "http://"):
		scheme = "http"
		rawURL = rawURL[7:]
	default:
		// No scheme — return as-is (could be "/path" or "host/path").
		if !strings.HasPrefix(rawURL, "/") {
			rawURL = "/" + rawURL
		}
		return "", "", rawURL
	}

	slashIdx := strings.IndexByte(rawURL, '/')
	if slashIdx < 0 {
		// e.g. "https://example.com" — no path component.
		return scheme, rawURL, "/"
	}
	return scheme, rawURL[:slashIdx], rawURL[slashIdx:]
}

// curlHeadersHas reports whether headers contains a header with key (case-insensitive).
func curlHeadersHas(headers []Header, key string) bool {
	for _, h := range headers {
		if eqFold(h.Key, key) {
			return true
		}
	}
	return false
}

// shellSplit tokenizes a shell command string respecting single and double quotes.
// It returns an error only for unclosed quotes.
func shellSplit(s string) ([]string, error) {
	var tokens []string
	var cur bytes.Buffer
	inSingle := false
	inDouble := false
	hasContent := false

	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inSingle:
			if c == '\'' {
				inSingle = false
			} else {
				cur.WriteByte(c)
				hasContent = true
			}
		case inDouble:
			if c == '"' {
				inDouble = false
			} else if c == '\\' && i+1 < len(s) {
				// Inside double quotes only a few chars are escapable.
				next := s[i+1]
				switch next {
				case '"', '\\', '$', '`', '\n':
					cur.WriteByte(next)
					i++
				default:
					cur.WriteByte(c) // literal backslash
				}
				hasContent = true
			} else {
				cur.WriteByte(c)
				hasContent = true
			}
		case c == '\'':
			inSingle = true
			hasContent = true // empty quotes still yield an empty token
		case c == '"':
			inDouble = true
			hasContent = true
		case c == '\\':
			if i+1 < len(s) {
				cur.WriteByte(s[i+1])
				i++
				hasContent = true
			}
		case c == ' ' || c == '\t':
			if hasContent {
				tokens = append(tokens, cur.String())
				cur.Reset()
				hasContent = false
			}
		default:
			cur.WriteByte(c)
			hasContent = true
		}
	}

	if inSingle {
		return nil, fmt.Errorf("unclosed single quote")
	}
	if inDouble {
		return nil, fmt.Errorf("unclosed double quote")
	}
	if hasContent {
		tokens = append(tokens, cur.String())
	}
	return tokens, nil
}

// buildMultipartForm encodes form fields as multipart/form-data with a fixed
// boundary. File upload references (@filename) are skipped with a warning.
func buildMultipartForm(fields []string, cp *curlParser) (body []byte, boundary string) {
	boundary = "ShapeHttpFormBoundary"
	var buf bytes.Buffer
	for _, field := range fields {
		eq := strings.IndexByte(field, '=')
		if eq < 0 {
			cp.warn(fmt.Sprintf("-F value %q has no '=', skipped", field))
			continue
		}
		name := field[:eq]
		value := field[eq+1:]
		if strings.HasPrefix(value, "@") {
			cp.warn(fmt.Sprintf("-F file upload %q is not supported, skipped", field))
			continue
		}
		buf.WriteString("--" + boundary + "\r\n")
		buf.WriteString("Content-Disposition: form-data; name=\"" + name + "\"\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(value)
		buf.WriteString("\r\n")
	}
	buf.WriteString("--" + boundary + "--\r\n")
	return buf.Bytes(), boundary
}

// buildURLEncoded builds an application/x-www-form-urlencoded body from
// --data-urlencode fields. Supported formats per curl(1):
//
//	"name=value"   → name=percentEncode(value)
//	"=value"       → percentEncode(value)
//	"name"         → percentEncode(name) (treated as value only)
func buildURLEncoded(fields []string) string {
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		eq := strings.IndexByte(field, '=')
		if eq < 0 {
			parts = append(parts, percentEncode(field))
		} else {
			name := field[:eq]
			value := field[eq+1:]
			if name == "" {
				parts = append(parts, percentEncode(value))
			} else {
				parts = append(parts, name+"="+percentEncode(value))
			}
		}
	}
	return strings.Join(parts, "&")
}

// expandShortFlags expands compound short flags (e.g. -sS → -s -S, -vk → -v -k).
// If a flag that takes an argument appears inside a compound (e.g. -XPOST),
// the remaining characters become that flag's argument, matching curl behaviour.
// Flags that already start with "--" or that are a single char are left unchanged.
func expandShortFlags(tokens []string) []string {
	// Single-char flags that consume the next token as their argument.
	shortArgFlags := map[byte]bool{
		'X': true, 'H': true, 'd': true, 'F': true,
		'u': true, 'o': true, 'A': true, 'e': true,
		'm': true, 'w': true, 'x': true, 'b': true,
	}
	out := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		// Only expand tokens of the form -(two or more letters/digits).
		if len(tok) > 2 && tok[0] == '-' && tok[1] != '-' && tok[1] != '#' {
			chars := tok[1:]
			expanded := false
			for i := 0; i < len(chars); i++ {
				c := chars[i]
				out = append(out, "-"+string(c))
				if shortArgFlags[c] {
					// Rest of the compound is the inline argument (e.g. -XPOST → -X POST).
					if i+1 < len(chars) {
						out = append(out, chars[i+1:])
					}
					expanded = true
					break
				}
				expanded = true
			}
			if !expanded {
				out = append(out, tok)
			}
		} else {
			out = append(out, tok)
		}
	}
	return out
}

// percentEncode percent-encodes a string per RFC 3986 unreserved characters.
func percentEncode(s string) string {
	var buf bytes.Buffer
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~' {
			buf.WriteByte(c)
		} else {
			fmt.Fprintf(&buf, "%%%02X", c)
		}
	}
	return buf.String()
}
