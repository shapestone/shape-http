package fastparser

import (
	"testing"
)

func TestParseRequest_Simple(t *testing.T) {
	data := []byte("GET /api/users HTTP/1.1\r\nHost: example.com\r\n\r\n")
	p := NewParser(data)
	req, err := p.ParseRequest()
	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}

	if req.Method != "GET" {
		t.Errorf("Method = %q, want GET", req.Method)
	}
	if req.Path != "/api/users" {
		t.Errorf("Path = %q, want /api/users", req.Path)
	}
	if req.Version != "HTTP/1.1" {
		t.Errorf("Version = %q, want HTTP/1.1", req.Version)
	}
	if len(req.Headers) != 1 {
		t.Fatalf("Headers count = %d, want 1", len(req.Headers))
	}
}

func TestParseResponse_Simple(t *testing.T) {
	data := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 5\r\n\r\nHello")
	p := NewParser(data)
	resp, err := p.ParseResponse()
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}

	if resp.Version != "HTTP/1.1" {
		t.Errorf("Version = %q, want HTTP/1.1", resp.Version)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if resp.Reason != "OK" {
		t.Errorf("Reason = %q, want OK", resp.Reason)
	}
	if string(resp.Body) != "Hello" {
		t.Errorf("Body = %q, want Hello", string(resp.Body))
	}
}

func TestParseRequest_WithQueryString(t *testing.T) {
	data := []byte("GET /search?q=hello&page=1 HTTP/1.1\r\nHost: example.com\r\n\r\n")
	p := NewParser(data)
	req, err := p.ParseRequest()
	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}

	if req.Path != "/search?q=hello&page=1" {
		t.Errorf("Path = %q, want /search?q=hello&page=1", req.Path)
	}
}

func TestParseRequest_MalformedRequestLine(t *testing.T) {
	data := []byte("GETHTTP/1.1\r\n\r\n")
	p := NewParser(data)
	_, err := p.ParseRequest()
	if err == nil {
		t.Error("expected error for malformed request line")
	}
}

func TestParseResponse_MalformedStatusLine(t *testing.T) {
	data := []byte("HTTP/1.1 abc OK\r\n\r\n")
	p := NewParser(data)
	_, err := p.ParseResponse()
	if err == nil {
		t.Error("expected error for invalid status code")
	}
}

func TestEqFold(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"Content-Type", "content-type", true},
		{"HOST", "host", true},
		{"Host", "Host", true},
		{"Host", "Hos", false},
		{"", "", true},
	}
	for _, tt := range tests {
		got := eqFold(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("eqFold(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
