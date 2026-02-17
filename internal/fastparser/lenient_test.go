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
