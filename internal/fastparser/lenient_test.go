package fastparser

import (
	"strings"
	"testing"
)

func TestLenient_NormalRequest(t *testing.T) {
	data := []byte("GET /api HTTP/1.1\r\nHost: example.com\r\n\r\n")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request, got nil")
	}
	if result.Request.Method != "GET" {
		t.Errorf("Method = %q, want GET", result.Request.Method)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings, got %v", result.Warnings)
	}
}

func TestLenient_NormalResponse(t *testing.T) {
	data := []byte("HTTP/1.1 200 OK\r\nContent-Length: 5\r\n\r\nHello")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Response == nil {
		t.Fatal("expected response, got nil")
	}
	if result.Response.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", result.Response.StatusCode)
	}
	if string(result.Response.Body) != "Hello" {
		t.Errorf("Body = %q, want Hello", string(result.Response.Body))
	}
}

func TestLenient_MissingVersion(t *testing.T) {
	data := []byte("GET /path\r\nHost: example.com\r\n\r\n")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if result.Request.Version != "HTTP/1.1" {
		t.Errorf("Version = %q, want HTTP/1.1 (default)", result.Request.Version)
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warning for missing version")
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "missing HTTP version") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'missing HTTP version' warning, got %v", result.Warnings)
	}
}

func TestLenient_WhitespaceBeforeColon(t *testing.T) {
	data := []byte("GET / HTTP/1.1\r\nContent-Type : text/html\r\n\r\n")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	// Should accept and trim
	if result.Request.Headers[0].Key != "Content-Type" {
		t.Errorf("Header key = %q, want Content-Type", result.Request.Headers[0].Key)
	}
	if result.Request.Headers[0].Value != "text/html" {
		t.Errorf("Header value = %q, want text/html", result.Request.Headers[0].Value)
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warning for whitespace before colon")
	}
}

func TestLenient_MalformedHeaderSkipped(t *testing.T) {
	data := []byte("GET / HTTP/1.1\r\nHost: example.com\r\nX-Bad Header Value\r\nAccept: text/html\r\n\r\n")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	// Should have 2 headers (malformed one skipped)
	if len(result.Request.Headers) != 2 {
		t.Errorf("Headers count = %d, want 2", len(result.Request.Headers))
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "malformed header") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'malformed header' warning, got %v", result.Warnings)
	}
}

func TestLenient_TruncatedBody(t *testing.T) {
	// CR-2: body shorter than Content-Length — read all available, warn, Partial=true.
	data := []byte("POST / HTTP/1.1\r\nContent-Length: 100\r\n\r\nshort")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if string(result.Request.Body) != "short" {
		t.Errorf("Body = %q, want short", string(result.Request.Body))
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "Content-Length declared") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected Content-Length mismatch warning, got %v", result.Warnings)
	}
}

func TestLenient_InvalidStatusCode(t *testing.T) {
	data := []byte("HTTP/1.1 abc OK\r\n\r\n")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Response == nil {
		t.Fatal("expected response")
	}
	if result.Response.StatusCode != 0 {
		t.Errorf("StatusCode = %d, want 0", result.Response.StatusCode)
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "invalid status code") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'invalid status code' warning, got %v", result.Warnings)
	}
}

func TestLenient_EmptyInput(t *testing.T) {
	p := NewLenientParser([]byte{})
	result := p.Parse()

	if result.Request != nil || result.Response != nil {
		t.Error("expected nil request and response for empty input")
	}
	if !result.Partial {
		t.Error("expected Partial=true for empty input")
	}
}

func TestLenient_BareLF(t *testing.T) {
	data := []byte("GET / HTTP/1.1\nHost: example.com\n\n")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if result.Request.Method != "GET" {
		t.Errorf("Method = %q, want GET", result.Request.Method)
	}
	// Bare LF should be accepted without warning in lenient mode
}

func TestLenient_MixedLineEndings(t *testing.T) {
	data := []byte("GET / HTTP/1.1\r\nHost: example.com\nAccept: text/html\r\n\r\n")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if len(result.Request.Headers) != 2 {
		t.Errorf("Headers count = %d, want 2", len(result.Request.Headers))
	}
}

func TestLenient_HeadersOnly_NoBody(t *testing.T) {
	// Truncated message: headers but no empty line
	data := []byte("GET / HTTP/1.1\r\nHost: example.com")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if getHeader(result.Request.Headers, "Host") != "example.com" {
		t.Errorf("Host = %q, want example.com", getHeader(result.Request.Headers, "Host"))
	}
}

// getHeader is a test helper to look up a header by key (case-insensitive).
func getHeader(headers []Header, key string) string {
	for _, h := range headers {
		if eqFold(h.Key, key) {
			return h.Value
		}
	}
	return ""
}

func TestLenient_StatusLineVersionOnly(t *testing.T) {
	// Case 1 in parseStatusLineLenient: only one part (just the version)
	data := []byte("HTTP/1.1\r\n\r\n")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Response == nil {
		t.Fatal("expected response")
	}
	if result.Response.StatusCode != 0 {
		t.Errorf("StatusCode = %d, want 0", result.Response.StatusCode)
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "no status code") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'no status code' warning, got %v", result.Warnings)
	}
}

func TestLenient_StatusLineEmptyParts(t *testing.T) {
	// Case 0 in parseStatusLineLenient: all-whitespace status line
	// bytes.Fields("   ") returns empty slice
	data := []byte("HTTP/1.1   \r\n\r\n")
	p := NewLenientParser(data)
	result := p.Parse()

	// Should parse as response (starts with HTTP/)
	if result.Response == nil {
		t.Fatal("expected response")
	}
}

func TestLenient_StatusLineTwoParts_ValidCode(t *testing.T) {
	// Case 2 in parseStatusLineLenient: version + status code, no reason
	data := []byte("HTTP/1.1 204\r\n\r\n")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Response == nil {
		t.Fatal("expected response")
	}
	if result.Response.StatusCode != 204 {
		t.Errorf("StatusCode = %d, want 204", result.Response.StatusCode)
	}
	if result.Response.Reason != "" {
		t.Errorf("Reason = %q, want empty", result.Response.Reason)
	}
}

func TestLenient_StatusLineTwoParts_InvalidCode(t *testing.T) {
	// Case 2 with invalid status code: version + non-numeric code, no reason
	data := []byte("HTTP/1.1 abc\r\n\r\n")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Response == nil {
		t.Fatal("expected response")
	}
	if result.Response.StatusCode != 0 {
		t.Errorf("StatusCode = %d, want 0 for invalid code", result.Response.StatusCode)
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "invalid status code") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'invalid status code' warning, got %v", result.Warnings)
	}
}

func TestLenient_RequestLineMethodOnly(t *testing.T) {
	// Case 1 in parseRequestLineLenient: only the method, no path or version
	data := []byte("GET\r\nHost: example.com\r\n\r\n")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if result.Request.Method != "GET" {
		t.Errorf("Method = %q, want GET", result.Request.Method)
	}
	if result.Request.Path != "/" {
		t.Errorf("Path = %q, want / (default)", result.Request.Path)
	}
	if result.Request.Version != "HTTP/1.1" {
		t.Errorf("Version = %q, want HTTP/1.1 (default)", result.Request.Version)
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "no path or version") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'no path or version' warning, got %v", result.Warnings)
	}
}

func TestLenient_RequestLineEmpty(t *testing.T) {
	// Case 0 in parseRequestLineLenient: whitespace-only request line
	data := []byte("   \r\nHost: example.com\r\n\r\n")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	// Empty line → empty method, default path and version
	if result.Request.Version != "HTTP/1.1" {
		t.Errorf("Version = %q, want HTTP/1.1 (default)", result.Request.Version)
	}
}

func TestLenient_BareCR(t *testing.T) {
	// Bare CR line endings (no LF) — lenient parser should accept them
	data := []byte("GET / HTTP/1.1\rHost: example.com\r\r")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if result.Request.Method != "GET" {
		t.Errorf("Method = %q, want GET", result.Request.Method)
	}
}

func TestLenient_PartialChunkedBody(t *testing.T) {
	// Malformed chunked body — partial decode should set partial=true
	data := []byte("POST / HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nhello")
	// missing terminating chunk
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	// Body should be available (raw remaining bytes on decode error)
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "chunked encoding error") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected chunked error warning, got %v", result.Warnings)
	}
}

func TestLenient_TruncatedBodyResponse(t *testing.T) {
	// CR-2: response body shorter than Content-Length — read all available, warn, Partial=true.
	data := []byte("HTTP/1.1 200 OK\r\nContent-Length: 100\r\n\r\nshort body")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Response == nil {
		t.Fatal("expected response")
	}
	if string(result.Response.Body) != "short body" {
		t.Errorf("Body = %q, want 'short body'", string(result.Response.Body))
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "Content-Length declared") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected Content-Length mismatch warning, got %v", result.Warnings)
	}
}

func TestLenient_ValidChunkedBody(t *testing.T) {
	// Valid chunked body should decode cleanly
	data := []byte("HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nhello\r\n0\r\n\r\n")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Response == nil {
		t.Fatal("expected response")
	}
	if string(result.Response.Body) != "hello" {
		t.Errorf("Body = %q, want hello", string(result.Response.Body))
	}
}

func TestLenient_TabOWSInHeaderValue(t *testing.T) {
	// Header value with tab whitespace — tests trimOWSBytes tab case
	data := []byte("GET / HTTP/1.1\r\nX-Custom:\tvalue\t\r\n\r\n")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if len(result.Request.Headers) != 1 {
		t.Fatalf("Headers count = %d, want 1", len(result.Request.Headers))
	}
	if result.Request.Headers[0].Value != "value" {
		t.Errorf("Header value = %q, want value", result.Request.Headers[0].Value)
	}
}

func TestLenient_ObsFoldHeader(t *testing.T) {
	// Obs-fold continuation line in lenient parser
	data := []byte("GET / HTTP/1.1\r\nX-Multi: part1\r\n continued\r\n\r\n")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	// Should have one header with combined value
	if len(result.Request.Headers) != 1 {
		t.Fatalf("Headers count = %d, want 1 (obs-fold joined)", len(result.Request.Headers))
	}
}

// TestLenient_BodyNoContentLength exercises parseBodyLenient's "remaining bytes"
// path — triggered when there is no Content-Length and no Transfer-Encoding.
func TestLenient_BodyNoContentLength(t *testing.T) {
	data := []byte("POST / HTTP/1.1\r\nHost: example.com\r\n\r\nhello world")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if string(result.Request.Body) != "hello world" {
		t.Errorf("Body = %q, want hello world", string(result.Request.Body))
	}
}

// TestLenient_ResponseBodyNoContentLength exercises parseBodyLenient's
// "remaining bytes" path for a response.
func TestLenient_ResponseBodyNoContentLength(t *testing.T) {
	data := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\nhello")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Response == nil {
		t.Fatal("expected response")
	}
	if string(result.Response.Body) != "hello" {
		t.Errorf("Body = %q, want hello", string(result.Response.Body))
	}
}

// TestLenient_HeadersWithBareHeaderSeparator exercises parseHeadersLenient
// when it sees a bare '\r' line ending as the empty-line terminator.
func TestLenient_BareCarriageReturnHeaders(t *testing.T) {
	data := []byte("GET / HTTP/1.1\rHost: example.com\r\r")
	p := NewLenientParser(data)
	result := p.Parse()
	if result.Request == nil {
		t.Fatal("expected request")
	}
}

// TestReadLineLenient_PosAtLength directly exercises the nil-return path of
// readLineLenient when pos >= length.
func TestReadLineLenient_PosAtLength(t *testing.T) {
	p := NewLenientParser([]byte("some data"))
	p.pos = p.length // advance past all data
	line := p.readLineLenient()
	if line != nil {
		t.Errorf("expected nil, got %q", string(line))
	}
}

// TestParseRequestLenient_NilLine directly exercises the "empty request, no start
// line found" path in parseRequestLenient by setting pos = length so that
// readLineLenient returns nil.
func TestParseRequestLenient_NilLine(t *testing.T) {
	p := NewLenientParser([]byte("GET /"))
	p.pos = p.length // exhaust the data
	req := p.parseRequestLenient()
	if req == nil {
		t.Fatal("expected non-nil request struct")
	}
	// The returned request has empty fields (method/path/version defaulted)
	if len(p.warnings) == 0 {
		t.Error("expected warning for empty request")
	}
}

// TestParseResponseLenient_NilLine directly exercises the "empty response, no start
// line found" path in parseResponseLenient.
func TestParseResponseLenient_NilLine(t *testing.T) {
	p := NewLenientParser([]byte("HTTP/1.1 200"))
	p.pos = p.length // exhaust the data
	resp := p.parseResponseLenient()
	if resp == nil {
		t.Fatal("expected non-nil response struct")
	}
	if len(p.warnings) == 0 {
		t.Error("expected warning for empty response")
	}
}

// TestParseStatusLineLenient_EmptyLine directly exercises case 0 of
// parseStatusLineLenient (bytes.Fields returns empty slice for all-whitespace input).
func TestParseStatusLineLenient_EmptyLine(t *testing.T) {
	p := NewLenientParser([]byte("HTTP/1.1"))
	// Pass an all-whitespace line so bytes.Fields returns []
	version, code, reason := p.parseStatusLineLenient([]byte("   "))
	if version != "HTTP/1.1" || code != 0 || reason != "" {
		t.Errorf("got (%q, %d, %q), want (HTTP/1.1, 0, \"\")", version, code, reason)
	}
	if len(p.warnings) == 0 {
		t.Error("expected warning for empty status line")
	}
}

// ── CR-1: bare hostname treated as implicit Host header ────────────────────

func TestLenient_BareHostname(t *testing.T) {
	// "example.com" on its own line — should become Host: example.com
	data := []byte("POST /api/users HTTP/1.1\r\nexample.com\r\nContent-Type: application/json\r\n\r\n{}")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if getHeader(result.Request.Headers, "Host") != "example.com" {
		t.Errorf("Host = %q, want example.com", getHeader(result.Request.Headers, "Host"))
	}
	if getHeader(result.Request.Headers, "Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", getHeader(result.Request.Headers, "Content-Type"))
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "implicit Host header") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected implicit Host header warning, got %v", result.Warnings)
	}
}

func TestLenient_BareHostnameWithPort(t *testing.T) {
	// "localhost:8080" — single-label host with a numeric port.
	// isSingleLabelHost detects the key and isPortStr detects the value,
	// so CR-3 now correctly converts it to Host: localhost:8080.
	data := []byte("GET /health HTTP/1.1\r\nlocalhost:8080\r\n\r\n")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if getHeader(result.Request.Headers, "Host") != "localhost:8080" {
		t.Errorf("Host = %q, want localhost:8080", getHeader(result.Request.Headers, "Host"))
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "bare host:port") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected bare host:port warning, got %v", result.Warnings)
	}
}

func TestLenient_BareHostnameIP(t *testing.T) {
	// IP address — should become Host: 192.168.1.1
	data := []byte("GET / HTTP/1.1\r\n192.168.1.1\r\n\r\n")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if getHeader(result.Request.Headers, "Host") != "192.168.1.1" {
		t.Errorf("Host = %q, want 192.168.1.1", getHeader(result.Request.Headers, "Host"))
	}
}

func TestLenient_BareWord_NotHostname(t *testing.T) {
	// "localhost" — no dot, no port — should still be skipped (not treated as Host)
	data := []byte("GET / HTTP/1.1\r\nlocalhost\r\nAccept: */*\r\n\r\n")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if getHeader(result.Request.Headers, "Host") != "" {
		t.Errorf("Host = %q, want empty — bare word without dot/port should not become Host", getHeader(result.Request.Headers, "Host"))
	}
	// Should still have a malformed-header warning
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "malformed header") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected malformed header warning for bare word, got %v", result.Warnings)
	}
}

// ── CR-2: Content-Length treated as advisory ───────────────────────────────

func TestLenient_ContentLength_BodyLonger(t *testing.T) {
	// Body is longer than Content-Length — read all bytes, warn, Partial=false.
	data := []byte("POST / HTTP/1.1\r\nContent-Length: 5\r\n\r\nhello world")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if string(result.Request.Body) != "hello world" {
		t.Errorf("Body = %q, want hello world (full body, not truncated at Content-Length)", string(result.Request.Body))
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "Content-Length declared") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected Content-Length mismatch warning, got %v", result.Warnings)
	}
	if result.Partial {
		t.Error("Partial = true, want false — actual body exceeds declared length, nothing is missing")
	}
}

func TestLenient_ContentLength_BodyShorter(t *testing.T) {
	// Body is shorter than Content-Length — read all available bytes, warn, Partial=true.
	data := []byte("POST / HTTP/1.1\r\nContent-Length: 100\r\n\r\nonly this")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if string(result.Request.Body) != "only this" {
		t.Errorf("Body = %q, want only this", string(result.Request.Body))
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "Content-Length declared") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected Content-Length mismatch warning, got %v", result.Warnings)
	}
}

// ── Stray blank line before headers ────────────────────────────────────────

func TestLenient_StrayBlankLineBeforeHeaders(t *testing.T) {
	// Extra blank line between request-line and headers — common in
	// hand-written / editor-generated requests.
	data := []byte("POST https://example.com:8080/api/users HTTP/1.1\r\n\r\nContent-Type: application/json\r\nContent-Length: 5\r\n\r\nhello")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if result.Request.Path != "/api/users" {
		t.Errorf("Path = %q, want /api/users", result.Request.Path)
	}
	if getHeader(result.Request.Headers, "Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", getHeader(result.Request.Headers, "Content-Type"))
	}
	if string(result.Request.Body) != "hello" {
		t.Errorf("Body = %q, want hello", string(result.Request.Body))
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "stray blank line") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected stray blank line warning, got %v", result.Warnings)
	}
}

func TestLenient_StrayBlankLine_BodyNotMisidentified(t *testing.T) {
	// Blank line before a JSON body (no headers) — body must NOT be parsed
	// as headers. The blank line is the real end of the (empty) headers section.
	data := []byte("POST / HTTP/1.1\r\n\r\n{\"key\":\"value\"}")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	// No headers — the blank line was a legitimate end-of-headers
	if len(result.Request.Headers) != 0 {
		t.Errorf("Headers = %v, want empty — JSON body must not be parsed as headers", result.Request.Headers)
	}
	if string(result.Request.Body) != `{"key":"value"}` {
		t.Errorf("Body = %q, want {\"key\":\"value\"}", string(result.Request.Body))
	}
}

// ── IPv6 literal address handling ──────────────────────────────────────────

func TestLenient_IPv6_BareHeaderLine(t *testing.T) {
	// "[::1]" on its own header line → Host: [::1]
	data := []byte("GET / HTTP/1.1\r\n[::1]\r\n\r\n")
	p := NewLenientParser(data)
	result := p.Parse()
	if result.Request == nil {
		t.Fatal("expected request")
	}
	if getHeader(result.Request.Headers, "Host") != "[::1]" {
		t.Errorf("Host = %q, want [::1]", getHeader(result.Request.Headers, "Host"))
	}
}

func TestLenient_IPv6_BareHeaderLineWithPort(t *testing.T) {
	// "[::1]:8080" on its own header line → Host: [::1]:8080
	data := []byte("POST /api HTTP/1.1\r\n[::1]:8080\r\nContent-Type: application/json\r\n\r\n{}")
	p := NewLenientParser(data)
	result := p.Parse()
	if result.Request == nil {
		t.Fatal("expected request")
	}
	if getHeader(result.Request.Headers, "Host") != "[::1]:8080" {
		t.Errorf("Host = %q, want [::1]:8080", getHeader(result.Request.Headers, "Host"))
	}
}

func TestLenient_IPv6_PathWithPort(t *testing.T) {
	// GET [::1]:8080/api/users HTTP/1.1
	data := []byte("GET [::1]:8080/api/users HTTP/1.1\r\n\r\n")
	p := NewLenientParser(data)
	result := p.Parse()
	if result.Request == nil {
		t.Fatal("expected request")
	}
	if result.Request.Path != "/api/users" {
		t.Errorf("Path = %q, want /api/users", result.Request.Path)
	}
	if getHeader(result.Request.Headers, "Host") != "[::1]:8080" {
		t.Errorf("Host = %q, want [::1]:8080", getHeader(result.Request.Headers, "Host"))
	}
}

func TestLenient_IPv6_PathNoPort(t *testing.T) {
	// GET [::1]/api/users HTTP/1.1
	data := []byte("GET [::1]/api/users HTTP/1.1\r\n\r\n")
	p := NewLenientParser(data)
	result := p.Parse()
	if result.Request == nil {
		t.Fatal("expected request")
	}
	if result.Request.Path != "/api/users" {
		t.Errorf("Path = %q, want /api/users", result.Request.Path)
	}
	if getHeader(result.Request.Headers, "Host") != "[::1]" {
		t.Errorf("Host = %q, want [::1]", getHeader(result.Request.Headers, "Host"))
	}
}

// ── localhost (single-label host) ──────────────────────────────────────────

func TestLenient_Localhost_PathWithPort(t *testing.T) {
	// GET localhost:8080/api/users — path prefix form
	data := []byte("GET localhost:8080/api/users\r\n\r\n")
	p := NewLenientParser(data)
	result := p.Parse()
	if result.Request == nil {
		t.Fatal("expected request")
	}
	if result.Request.Path != "/api/users" {
		t.Errorf("Path = %q, want /api/users", result.Request.Path)
	}
	if getHeader(result.Request.Headers, "Host") != "localhost:8080" {
		t.Errorf("Host = %q, want localhost:8080", getHeader(result.Request.Headers, "Host"))
	}
}

// ── CR-3: bare host:port in header section ─────────────────────────────────

func TestLenient_CR3_BareHostPort(t *testing.T) {
	// "example.com:8080" in the header section has a colon, so it is parsed by
	// the normal key:value path first. CR-3 then detects that the key looks
	// like a hostname (contains a dot, hostname chars only) and the value is
	// an all-digit port, and re-emits it as Host: example.com:8080.
	data := []byte("POST /api/users HTTP/1.1\r\nexample.com:8080\r\nContent-Type: application/json\r\n\r\n{}")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if getHeader(result.Request.Headers, "Host") != "example.com:8080" {
		t.Errorf("Host = %q, want example.com:8080", getHeader(result.Request.Headers, "Host"))
	}
	if getHeader(result.Request.Headers, "Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", getHeader(result.Request.Headers, "Content-Type"))
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "bare host:port") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected bare host:port warning, got %v", result.Warnings)
	}
}

func TestLenient_CR3_NoTrigger_LocalhostNoPort(t *testing.T) {
	// "localhost" (no dot, no port) does not trigger CR-3 — stays as malformed header.
	data := []byte("GET / HTTP/1.1\r\nlocalhost\r\n\r\n")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if getHeader(result.Request.Headers, "Host") != "" {
		t.Errorf("Host = %q, want empty — bare word without dot does not trigger CR-3", getHeader(result.Request.Headers, "Host"))
	}
}

func TestLenient_CR3_NoTrigger_ContentLength(t *testing.T) {
	// "Content-Length: 42" must not be affected — key has no dot, value is digits.
	data := []byte("POST / HTTP/1.1\r\nContent-Length: 42\r\n\r\nhello world, here is some text!!! ok")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	// Content-Length should stay as-is (no CR-3 conversion)
	if getHeader(result.Request.Headers, "Content-Length") != "42" {
		t.Errorf("Content-Length = %q, want 42", getHeader(result.Request.Headers, "Content-Length"))
	}
	if getHeader(result.Request.Headers, "Host") != "" {
		t.Errorf("Host unexpectedly set — CR-3 must not fire on Content-Length")
	}
}

// ── Path normalization (cases 1-4, 6) ──────────────────────────────────────

func TestLenient_PathBareAuthorityWithPort(t *testing.T) {
	// Case 1: POST example.com:8080/api/users HTTP/1.1
	// Path starts with "host:port/" — extract host, use remainder as path.
	data := []byte("POST example.com:8080/api/users HTTP/1.1\r\nContent-Type: application/json\r\n\r\n{}")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if result.Request.Path != "/api/users" {
		t.Errorf("Path = %q, want /api/users", result.Request.Path)
	}
	if getHeader(result.Request.Headers, "Host") != "example.com:8080" {
		t.Errorf("Host = %q, want example.com:8080", getHeader(result.Request.Headers, "Host"))
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "bare host prefix") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected bare host prefix warning, got %v", result.Warnings)
	}
}

func TestLenient_PathAbsoluteFormWithPort(t *testing.T) {
	// Case 2: POST https://example.com:8080/api/users HTTP/1.1
	// Valid absolute-form — extract host and normalize to origin-form path.
	data := []byte("POST https://example.com:8080/api/users HTTP/1.1\r\nContent-Type: application/json\r\n\r\n{}")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if result.Request.Path != "/api/users" {
		t.Errorf("Path = %q, want /api/users", result.Request.Path)
	}
	if getHeader(result.Request.Headers, "Host") != "example.com:8080" {
		t.Errorf("Host = %q, want example.com:8080", getHeader(result.Request.Headers, "Host"))
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "absolute-form request-target") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected absolute-form warning, got %v", result.Warnings)
	}
}

func TestLenient_PathAbsoluteFormWithPort_MissingVersion(t *testing.T) {
	// Case 3: POST https://example.com:8080/api/users  (no HTTP version)
	// Missing version AND absolute-form path — lenient handles both.
	data := []byte("POST https://example.com:8080/api/users\r\nContent-Type: application/json\r\n\r\n{}")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if result.Request.Version != "HTTP/1.1" {
		t.Errorf("Version = %q, want HTTP/1.1 (defaulted)", result.Request.Version)
	}
	if result.Request.Path != "/api/users" {
		t.Errorf("Path = %q, want /api/users", result.Request.Path)
	}
	if getHeader(result.Request.Headers, "Host") != "example.com:8080" {
		t.Errorf("Host = %q, want example.com:8080", getHeader(result.Request.Headers, "Host"))
	}
}

func TestLenient_PathAbsoluteFormNoPort_MissingVersion(t *testing.T) {
	// Case 4: POST https://example.com/api/users  (no port, no HTTP version)
	data := []byte("POST https://example.com/api/users\r\nContent-Type: application/json\r\n\r\n{}")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if result.Request.Version != "HTTP/1.1" {
		t.Errorf("Version = %q, want HTTP/1.1 (defaulted)", result.Request.Version)
	}
	if result.Request.Path != "/api/users" {
		t.Errorf("Path = %q, want /api/users", result.Request.Path)
	}
	if getHeader(result.Request.Headers, "Host") != "example.com" {
		t.Errorf("Host = %q, want example.com", getHeader(result.Request.Headers, "Host"))
	}
}

func TestLenient_PathBareAuthorityNoPort(t *testing.T) {
	// Case 6: POST example.com/api/users HTTP/1.1
	// Bare hostname prefix (with dot, no port) — extract host, fix path.
	data := []byte("POST example.com/api/users HTTP/1.1\r\nContent-Type: application/json\r\n\r\n{}")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if result.Request.Path != "/api/users" {
		t.Errorf("Path = %q, want /api/users", result.Request.Path)
	}
	if getHeader(result.Request.Headers, "Host") != "example.com" {
		t.Errorf("Host = %q, want example.com", getHeader(result.Request.Headers, "Host"))
	}
}

func TestLenient_PathNormalization_ExplicitHostWins(t *testing.T) {
	// If both the path and a header supply a host, the explicit header wins.
	data := []byte("POST https://from-path.example.com/api HTTP/1.1\r\nHost: explicit.example.com\r\n\r\n")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if result.Request.Path != "/api" {
		t.Errorf("Path = %q, want /api", result.Request.Path)
	}
	// Explicit Host header must be preserved; path-derived host must not override it.
	if getHeader(result.Request.Headers, "Host") != "explicit.example.com" {
		t.Errorf("Host = %q, want explicit.example.com", getHeader(result.Request.Headers, "Host"))
	}
}

func TestLenient_PathAbsoluteFormNoPath(t *testing.T) {
	// Absolute URL with no path component — path should default to "/".
	data := []byte("GET https://example.com HTTP/1.1\r\n\r\n")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if result.Request.Path != "/" {
		t.Errorf("Path = %q, want /", result.Request.Path)
	}
	if getHeader(result.Request.Headers, "Host") != "example.com" {
		t.Errorf("Host = %q, want example.com", getHeader(result.Request.Headers, "Host"))
	}
}

func TestLenient_ContentLength_Exact(t *testing.T) {
	// Content-Length matches exactly — no warning, no Partial.
	data := []byte("POST / HTTP/1.1\r\nContent-Length: 5\r\n\r\nhello")
	p := NewLenientParser(data)
	result := p.Parse()

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if string(result.Request.Body) != "hello" {
		t.Errorf("Body = %q, want hello", string(result.Request.Body))
	}
	for _, w := range result.Warnings {
		if strings.Contains(w, "Content-Length") {
			t.Errorf("unexpected Content-Length warning when lengths match: %s", w)
		}
	}
}
