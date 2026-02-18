package http

import (
	"strings"
	"testing"
)

func TestUnmarshalLenient_ValidRequest(t *testing.T) {
	data := []byte("GET /api HTTP/1.1\r\nHost: example.com\r\n\r\n")
	result := UnmarshalLenient(data)

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if result.Request.Method != "GET" {
		t.Errorf("Method = %q, want GET", result.Request.Method)
	}
	if result.Partial {
		t.Error("expected Partial=false")
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings, got %v", result.Warnings)
	}
}

func TestUnmarshalLenient_ValidResponse(t *testing.T) {
	data := []byte("HTTP/1.1 200 OK\r\nContent-Length: 5\r\n\r\nHello")
	result := UnmarshalLenient(data)

	if result.Response == nil {
		t.Fatal("expected response")
	}
	if result.Response.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", result.Response.StatusCode)
	}
	if string(result.Response.Body) != "Hello" {
		t.Errorf("Body = %q, want Hello", string(result.Response.Body))
	}
}

func TestUnmarshalLenient_MissingVersion(t *testing.T) {
	data := []byte("GET /path\r\nHost: example.com\r\n\r\n")
	result := UnmarshalLenient(data)

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if result.Request.Version != "HTTP/1.1" {
		t.Errorf("Version = %q, want HTTP/1.1 (default)", result.Request.Version)
	}
	hasWarning := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "missing HTTP version") {
			hasWarning = true
		}
	}
	if !hasWarning {
		t.Errorf("expected warning about missing version, got %v", result.Warnings)
	}
}

func TestUnmarshalLenient_TruncatedBody(t *testing.T) {
	// CR-2: body shorter than Content-Length → read all available, warn, Partial=true.
	data := []byte("POST / HTTP/1.1\r\nContent-Length: 100\r\n\r\npartial data")
	result := UnmarshalLenient(data)

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if string(result.Request.Body) != "partial data" {
		t.Errorf("Body = %q, want 'partial data'", string(result.Request.Body))
	}
	if !result.Partial {
		t.Error("expected Partial=true when body is shorter than Content-Length")
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

func TestUnmarshalLenient_MalformedHeaders(t *testing.T) {
	data := []byte("GET / HTTP/1.1\r\nHost: ok.com\r\nBad Line Without Colon\r\nAccept: text/html\r\n\r\n")
	result := UnmarshalLenient(data)

	if result.Request == nil {
		t.Fatal("expected request")
	}
	// Malformed header should be skipped
	if len(result.Request.Headers) != 2 {
		t.Errorf("Headers count = %d, want 2 (malformed one skipped)", len(result.Request.Headers))
	}
}

func TestUnmarshalLenient_InvalidStatusCode(t *testing.T) {
	data := []byte("HTTP/1.1 xyz OK\r\n\r\n")
	result := UnmarshalLenient(data)

	if result.Response == nil {
		t.Fatal("expected response")
	}
	if result.Response.StatusCode != 0 {
		t.Errorf("StatusCode = %d, want 0 for invalid code", result.Response.StatusCode)
	}
}

func TestUnmarshalLenient_EmptyInput(t *testing.T) {
	result := UnmarshalLenient([]byte{})
	if !result.Partial {
		t.Error("expected Partial=true for empty input")
	}
}

func TestUnmarshalLenient_WhitespaceBeforeColon(t *testing.T) {
	data := []byte("GET / HTTP/1.1\r\nContent-Type : text/html\r\n\r\n")
	result := UnmarshalLenient(data)

	if result.Request == nil {
		t.Fatal("expected request")
	}
	// Lenient should accept and trim whitespace before colon
	if result.Request.Headers.Get("Content-Type") != "text/html" {
		t.Errorf("Content-Type = %q, want text/html", result.Request.Headers.Get("Content-Type"))
	}
}

func TestParseLenient_Request(t *testing.T) {
	node, warnings, err := ParseLenient("GET /api HTTP/1.1\r\nHost: example.com\r\n\r\n")
	if err != nil {
		t.Fatalf("ParseLenient() error = %v", err)
	}
	if node == nil {
		t.Fatal("expected node, got nil")
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for valid request, got %v", warnings)
	}

	// Node must be renderable back to wire format
	wire, err := Render(node)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if len(wire) == 0 {
		t.Error("expected non-empty wire bytes")
	}
}

func TestParseLenient_Response(t *testing.T) {
	node, warnings, err := ParseLenient("HTTP/1.1 200 OK\r\nContent-Length: 5\r\n\r\nHello")
	if err != nil {
		t.Fatalf("ParseLenient() error = %v", err)
	}
	if node == nil {
		t.Fatal("expected node, got nil")
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for valid response, got %v", warnings)
	}
}

func TestParseLenient_MalformedRequest(t *testing.T) {
	// Missing version — lenient should still return a node with a warning
	node, warnings, err := ParseLenient("GET /path\r\nHost: example.com\r\n\r\n")
	if err != nil {
		t.Fatalf("ParseLenient() error = %v", err)
	}
	if node == nil {
		t.Fatal("expected node, got nil")
	}
	if len(warnings) == 0 {
		t.Error("expected warnings for missing version")
	}
}

func TestParseLenient_EmptyInput(t *testing.T) {
	// Empty input: should return an "unknown" object node with no error
	node, _, err := ParseLenient("")
	if err != nil {
		t.Fatalf("ParseLenient() error = %v", err)
	}
	if node == nil {
		t.Fatal("expected node, got nil")
	}
	// Node should be the "unknown" placeholder
	wire, renderErr := Render(node)
	// Render may fail for "unknown" type — that's OK, the node itself exists
	_ = wire
	_ = renderErr
}

func TestParseLenient_Garbage(t *testing.T) {
	// Completely garbage input — treated as request attempt, returns node with warnings
	node, warnings, err := ParseLenient("not http at all !!!")
	if err != nil {
		t.Fatalf("ParseLenient() error = %v", err)
	}
	if node == nil {
		t.Fatal("expected node, got nil")
	}
	_ = warnings // may or may not have warnings depending on what was extracted
}

func TestUnmarshalLenient_TruncatedResponseBody(t *testing.T) {
	// CR-2: response body shorter than Content-Length → read all available, warn, Partial=true.
	data := []byte("HTTP/1.1 200 OK\r\nContent-Length: 100\r\n\r\nshort response body")
	result := UnmarshalLenient(data)

	if result.Response == nil {
		t.Fatal("expected response")
	}
	if !result.Partial {
		t.Error("expected Partial=true when response body is shorter than Content-Length")
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

func TestUnmarshalLenient_BareHostname(t *testing.T) {
	// CR-1: bare hostname on its own line → treated as Host header.
	data := []byte("POST /api/users HTTP/1.1\r\nexample.com\r\nContent-Type: application/json\r\n\r\n{}")
	result := UnmarshalLenient(data)

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if result.Request.Headers.Get("Host") != "example.com" {
		t.Errorf("Host = %q, want example.com", result.Request.Headers.Get("Host"))
	}
	if result.Request.Headers.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", result.Request.Headers.Get("Content-Type"))
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

func TestUnmarshalLenient_ContentLength_BodyLonger(t *testing.T) {
	// CR-2: body longer than Content-Length → read full body, warn, Partial=false.
	data := []byte("POST / HTTP/1.1\r\nContent-Length: 5\r\n\r\nhello world")
	result := UnmarshalLenient(data)

	if result.Request == nil {
		t.Fatal("expected request")
	}
	if string(result.Request.Body) != "hello world" {
		t.Errorf("Body = %q, want hello world", string(result.Request.Body))
	}
	if result.Partial {
		t.Error("Partial = true, want false — body exceeds declared length, nothing is missing")
	}
}
