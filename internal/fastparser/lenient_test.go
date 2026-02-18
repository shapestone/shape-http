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
		if strings.Contains(w, "truncated") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected truncation warning, got %v", result.Warnings)
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
	// Response with truncated body — should set partial=true via warning
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
		if strings.Contains(w, "truncated") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected truncation warning, got %v", result.Warnings)
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
