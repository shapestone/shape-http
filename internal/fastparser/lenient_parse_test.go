package fastparser

// lenient_parse_test.go — targeted unit tests for the branches in
// LenientParser.Parse(), normalizePathLenient(), and parseHeadersLenient()
// that were below coverage:
//   - Parse(): pos >= length after skipping leading blank lines
//   - normalizePathLenient(): userinfo (user:pass@) stripping in absolute URL
//   - normalizePathLenient(): authority == "" (e.g. "https:///path")
//   - parseHeadersLenient(): pos >= length without finding empty line
//   - parseHeadersLenient(): bare non-hostname line with no colon → malformed
//   - isHostnameLike(): port contains non-digit char (allDigits = false, break)

import (
	"strings"
	"testing"
)

// ── Parse(): pos >= length after skipping blank lines ────────────────────

func TestLenientParse_AllNewlines(t *testing.T) {
	// Input that consists only of newlines — after skipping leading blank lines
	// pos reaches length, triggering the second "empty input" warning path.
	p := &LenientParser{data: []byte("\n\n\n"), length: 3}
	result := p.Parse()
	if !result.Partial {
		t.Error("expected Partial=true for all-newline input")
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "empty input") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'empty input' warning, got %v", result.Warnings)
	}
}

// ── normalizePathLenient(): userinfo stripping ────────────────────────────

func TestLenientParse_AbsoluteURLWithUserinfo(t *testing.T) {
	// Absolute-form request target containing userinfo (user:pass@host).
	// normalizePathLenient must strip the userinfo and set Host to the bare host.
	input := "GET https://alice:secret@api.example.com/v1/users HTTP/1.1\r\n\r\n"
	p := NewLenientParser([]byte(input))
	result := p.Parse()
	if result.Request == nil {
		t.Fatal("expected non-nil Request")
	}
	// Path must be the bare path, without userinfo or host.
	if result.Request.Path != "/v1/users" {
		t.Errorf("Path = %q, want /v1/users", result.Request.Path)
	}
	// Scheme must be extracted.
	if result.Request.Scheme != "https" {
		t.Errorf("Scheme = %q, want https", result.Request.Scheme)
	}
	// Host header must be the bare host (userinfo stripped).
	var host string
	for _, h := range result.Request.Headers {
		if strings.EqualFold(h.Key, "Host") {
			host = h.Value
			break
		}
	}
	if host != "api.example.com" {
		t.Errorf("Host = %q, want api.example.com", host)
	}
}

// ── normalizePathLenient(): authority == "" (https:///path) ───────────────

func TestLenientParse_AbsoluteURLNoAuthority(t *testing.T) {
	// "https:///path" — scheme present but no authority (empty between // and /).
	// normalizePathLenient returns (path, "", scheme) when authority is empty.
	input := "GET https:///internal/path HTTP/1.1\r\n\r\n"
	p := NewLenientParser([]byte(input))
	result := p.Parse()
	if result.Request == nil {
		t.Fatal("expected non-nil Request")
	}
	// Path must be extracted; no Host header injected (authority was empty).
	if result.Request.Path != "/internal/path" {
		t.Errorf("Path = %q, want /internal/path", result.Request.Path)
	}
	if result.Request.Scheme != "https" {
		t.Errorf("Scheme = %q, want https", result.Request.Scheme)
	}
}

// ── parseHeadersLenient(): pos >= length without empty line ───────────────

func TestLenientParse_TruncatedHeaders(t *testing.T) {
	// Input that ends mid-headers without a blank line separating headers
	// from body. parseHeadersLenient exits via the pos >= length early return.
	input := "GET / HTTP/1.1\r\nHost: example.com\r\nX-Custom: value"
	// Note: no trailing CRLF or empty line.
	p := NewLenientParser([]byte(input))
	result := p.Parse()
	if result.Request == nil {
		t.Fatal("expected non-nil Request")
	}
	// At minimum, Host header must be parsed.
	var host string
	for _, h := range result.Request.Headers {
		if strings.EqualFold(h.Key, "Host") {
			host = h.Value
			break
		}
	}
	if host != "example.com" {
		t.Errorf("Host = %q, want example.com", host)
	}
}

// ── parseHeadersLenient(): bare non-hostname line with no colon ───────────

func TestLenientParse_MalformedHeaderNonHostname(t *testing.T) {
	// A line that has no colon AND is not a valid hostname (e.g. a bare word
	// that starts with an invalid char or is otherwise non-hostname) must
	// produce a "malformed header (no colon)" warning and be skipped.
	//
	// We use a line starting with a digit followed by non-dot content,
	// which satisfies neither isHostnameLike nor IPv6 patterns.
	input := "GET / HTTP/1.1\r\n123notaheader\r\nHost: example.com\r\n\r\n"
	p := NewLenientParser([]byte(input))
	result := p.Parse()
	if result.Request == nil {
		t.Fatal("expected non-nil Request")
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "malformed header") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'malformed header' warning, got %v", result.Warnings)
	}
}

// ── parseHeadersLenient(): IPv6-bracket line that is not a valid IPv6 addr ──

func TestLenientParse_MalformedIPv6HeaderLine(t *testing.T) {
	// A header line that starts with '[' but contains no colon inside the
	// brackets — parseIPv6HostLine returns "" and the else branch fires:
	// "malformed header (no colon), skipped".
	input := "GET / HTTP/1.1\r\n[notanipv6]\r\nHost: example.com\r\n\r\n"
	p := NewLenientParser([]byte(input))
	result := p.Parse()
	if result.Request == nil {
		t.Fatal("expected non-nil Request")
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "malformed header") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'malformed header' warning for non-IPv6 bracket line, got %v", result.Warnings)
	}
}

// ── isHostnameLike(): port with non-digit char ────────────────────────────

func TestIsHostnameLike_PortWithNonDigit(t *testing.T) {
	// "localhost:abc" — colon exists, port is "abc" which fails the allDigits
	// check (the `allDigits = false; break` branch is taken), so hasPort stays
	// false. The host loop then hits ':' as an invalid char → returns false.
	if isHostnameLike([]byte("localhost:abc")) {
		t.Error("isHostnameLike(localhost:abc) = true, want false")
	}
}

func TestIsHostnameLike_PortDigitsValid(t *testing.T) {
	// Ensure the positive path for allDigits=true still works.
	if !isHostnameLike([]byte("localhost:8080")) {
		t.Error("isHostnameLike(localhost:8080) = false, want true")
	}
}
