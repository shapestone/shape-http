package http

import (
	"testing"
)

func TestHeaders_Get(t *testing.T) {
	h := Headers{
		{Key: "Content-Type", Value: "application/json"},
		{Key: "Host", Value: "example.com"},
		{Key: "X-Custom", Value: "value1"},
	}

	tests := []struct {
		key  string
		want string
	}{
		{"Content-Type", "application/json"},
		{"content-type", "application/json"},
		{"CONTENT-TYPE", "application/json"},
		{"Host", "example.com"},
		{"X-Missing", ""},
	}

	for _, tt := range tests {
		got := h.Get(tt.key)
		if got != tt.want {
			t.Errorf("Get(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestHeaders_Values(t *testing.T) {
	h := Headers{
		{Key: "Set-Cookie", Value: "a=1"},
		{Key: "Content-Type", Value: "text/html"},
		{Key: "Set-Cookie", Value: "b=2"},
		{Key: "Set-Cookie", Value: "c=3"},
	}

	vals := h.Values("Set-Cookie")
	if len(vals) != 3 {
		t.Fatalf("Values(Set-Cookie) returned %d values, want 3", len(vals))
	}
	if vals[0] != "a=1" || vals[1] != "b=2" || vals[2] != "c=3" {
		t.Errorf("Values(Set-Cookie) = %v, want [a=1 b=2 c=3]", vals)
	}

	vals = h.Values("X-Missing")
	if len(vals) != 0 {
		t.Errorf("Values(X-Missing) = %v, want empty", vals)
	}
}

func TestHeaders_Set(t *testing.T) {
	h := Headers{
		{Key: "Content-Type", Value: "text/plain"},
		{Key: "Host", Value: "example.com"},
		{Key: "Content-Type", Value: "duplicate"},
	}

	h.Set("Content-Type", "application/json")

	if got := h.Get("Content-Type"); got != "application/json" {
		t.Errorf("after Set, Get(Content-Type) = %q, want %q", got, "application/json")
	}

	// Should have removed duplicates
	vals := h.Values("Content-Type")
	if len(vals) != 1 {
		t.Errorf("after Set, Content-Type count = %d, want 1", len(vals))
	}

	// Set new header
	h.Set("Accept", "text/html")
	if got := h.Get("Accept"); got != "text/html" {
		t.Errorf("after Set new, Get(Accept) = %q, want %q", got, "text/html")
	}
}

func TestHeaders_Add(t *testing.T) {
	var h Headers
	h.Add("Set-Cookie", "a=1")
	h.Add("Set-Cookie", "b=2")

	vals := h.Values("Set-Cookie")
	if len(vals) != 2 {
		t.Fatalf("after Add, Values(Set-Cookie) returned %d, want 2", len(vals))
	}
}

func TestHeaders_Del(t *testing.T) {
	h := Headers{
		{Key: "Content-Type", Value: "text/plain"},
		{Key: "Host", Value: "example.com"},
		{Key: "Content-Type", Value: "duplicate"},
	}

	h.Del("Content-Type")

	if len(h) != 1 {
		t.Fatalf("after Del, len = %d, want 1", len(h))
	}
	if h[0].Key != "Host" {
		t.Errorf("after Del, remaining header = %q, want Host", h[0].Key)
	}
}

func TestHeaders_Clone(t *testing.T) {
	original := Headers{
		{Key: "Content-Type", Value: "text/plain"},
		{Key: "Host", Value: "example.com"},
	}

	clone := original.Clone()

	// Modify clone should not affect original
	clone[0].Value = "modified"
	if original[0].Value == "modified" {
		t.Error("Clone is not a deep copy")
	}

	// Nil clone
	var nilHeaders Headers
	if nilHeaders.Clone() != nil {
		t.Error("Clone of nil should return nil")
	}
}

func TestHeaders_ContentLength(t *testing.T) {
	tests := []struct {
		name    string
		headers Headers
		want    int64
	}{
		{
			name:    "valid",
			headers: Headers{{Key: "Content-Length", Value: "42"}},
			want:    42,
		},
		{
			name:    "with whitespace",
			headers: Headers{{Key: "Content-Length", Value: " 42 "}},
			want:    42,
		},
		{
			name:    "absent",
			headers: Headers{},
			want:    -1,
		},
		{
			name:    "invalid",
			headers: Headers{{Key: "Content-Length", Value: "abc"}},
			want:    -1,
		},
		{
			name:    "zero",
			headers: Headers{{Key: "Content-Length", Value: "0"}},
			want:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.headers.ContentLength()
			if got != tt.want {
				t.Errorf("ContentLength() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestHeaders_IsChunked(t *testing.T) {
	tests := []struct {
		name    string
		headers Headers
		want    bool
	}{
		{
			name:    "chunked",
			headers: Headers{{Key: "Transfer-Encoding", Value: "chunked"}},
			want:    true,
		},
		{
			name:    "mixed case",
			headers: Headers{{Key: "Transfer-Encoding", Value: "Chunked"}},
			want:    true,
		},
		{
			name:    "gzip then chunked",
			headers: Headers{{Key: "Transfer-Encoding", Value: "gzip, chunked"}},
			want:    true,
		},
		{
			name:    "absent",
			headers: Headers{},
			want:    false,
		},
		{
			name:    "identity",
			headers: Headers{{Key: "Transfer-Encoding", Value: "identity"}},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.headers.IsChunked()
			if got != tt.want {
				t.Errorf("IsChunked() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessage_Interface(t *testing.T) {
	// Verify Request implements Message
	var _ Message = &Request{
		Method:  "GET",
		Path:    "/",
		Version: "HTTP/1.1",
		Headers: Headers{{Key: "Host", Value: "example.com"}},
		Body:    []byte("body"),
	}

	// Verify Response implements Message
	var _ Message = &Response{
		Version:    "HTTP/1.1",
		StatusCode: 200,
		Reason:     "OK",
		Headers:    Headers{{Key: "Content-Type", Value: "text/plain"}},
		Body:       []byte("body"),
	}
}

func TestRequest_MessageMethods(t *testing.T) {
	req := &Request{
		Method:  "POST",
		Path:    "/api",
		Version: "HTTP/1.1",
		Headers: Headers{{Key: "Host", Value: "example.com"}},
		Body:    []byte("hello"),
	}

	if req.GetVersion() != "HTTP/1.1" {
		t.Errorf("GetVersion() = %q, want HTTP/1.1", req.GetVersion())
	}
	if len(req.GetHeaders()) != 1 {
		t.Errorf("GetHeaders() len = %d, want 1", len(req.GetHeaders()))
	}
	if string(req.GetBody()) != "hello" {
		t.Errorf("GetBody() = %q, want hello", string(req.GetBody()))
	}
}

func TestResponse_MessageMethods(t *testing.T) {
	resp := &Response{
		Version:    "HTTP/1.1",
		StatusCode: 200,
		Reason:     "OK",
		Headers:    Headers{{Key: "Content-Type", Value: "text/plain"}},
		Body:       []byte("world"),
	}

	if resp.GetVersion() != "HTTP/1.1" {
		t.Errorf("GetVersion() = %q, want HTTP/1.1", resp.GetVersion())
	}
	if len(resp.GetHeaders()) != 1 {
		t.Errorf("GetHeaders() len = %d, want 1", len(resp.GetHeaders()))
	}
	if string(resp.GetBody()) != "world" {
		t.Errorf("GetBody() = %q, want world", string(resp.GetBody()))
	}
}

func TestParseError(t *testing.T) {
	e1 := newParseError("bad request line", 1)
	if e1.Error() != "http: parse error at line 1: bad request line" {
		t.Errorf("Error() = %q", e1.Error())
	}

	e2 := newParseErrorAtPos("unexpected byte", 42)
	if e2.Error() != "http: parse error at position 42: unexpected byte" {
		t.Errorf("Error() = %q", e2.Error())
	}

	e3 := &ParseError{Message: "generic error"}
	if e3.Error() != "http: generic error" {
		t.Errorf("Error() = %q", e3.Error())
	}
}
