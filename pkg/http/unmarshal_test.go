package http

import (
	"testing"
)

func TestUnmarshalRequest_Simple(t *testing.T) {
	data := []byte("GET /api/users HTTP/1.1\r\nHost: example.com\r\nAccept: application/json\r\n\r\n")

	req := &Request{}
	err := Unmarshal(data, req)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
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
	if len(req.Headers) != 2 {
		t.Fatalf("Headers count = %d, want 2", len(req.Headers))
	}
	if req.Headers.Get("Host") != "example.com" {
		t.Errorf("Host = %q, want example.com", req.Headers.Get("Host"))
	}
	if req.Headers.Get("Accept") != "application/json" {
		t.Errorf("Accept = %q, want application/json", req.Headers.Get("Accept"))
	}
	if req.Body != nil {
		t.Errorf("Body = %v, want nil", req.Body)
	}
}

func TestUnmarshalRequest_WithBody(t *testing.T) {
	data := []byte("POST /api/users HTTP/1.1\r\nHost: example.com\r\nContent-Type: application/json\r\nContent-Length: 19\r\n\r\n{\"name\":\"John Doe\"}")

	req := &Request{}
	err := Unmarshal(data, req)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if req.Method != "POST" {
		t.Errorf("Method = %q, want POST", req.Method)
	}
	if string(req.Body) != `{"name":"John Doe"}` {
		t.Errorf("Body = %q, want {\"name\":\"John Doe\"}", string(req.Body))
	}
}

func TestUnmarshalRequest_NoBody_ConnectionClose(t *testing.T) {
	// No Content-Length, no chunked = remaining bytes are body
	data := []byte("POST /api HTTP/1.1\r\nHost: example.com\r\n\r\nhello world")

	req := &Request{}
	err := Unmarshal(data, req)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if string(req.Body) != "hello world" {
		t.Errorf("Body = %q, want hello world", string(req.Body))
	}
}

func TestUnmarshalRequest_Function(t *testing.T) {
	data := []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")

	req, err := UnmarshalRequest(data)
	if err != nil {
		t.Fatalf("UnmarshalRequest() error = %v", err)
	}

	if req.Method != "GET" {
		t.Errorf("Method = %q, want GET", req.Method)
	}
	if req.Path != "/" {
		t.Errorf("Path = %q, want /", req.Path)
	}
}

func TestUnmarshalResponse_Simple(t *testing.T) {
	data := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\n")

	resp := &Response{}
	err := Unmarshal(data, resp)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
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
}

func TestUnmarshalResponse_WithBody(t *testing.T) {
	data := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 13\r\n\r\nHello, World!")

	resp := &Response{}
	err := Unmarshal(data, resp)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if string(resp.Body) != "Hello, World!" {
		t.Errorf("Body = %q, want Hello, World!", string(resp.Body))
	}
}

func TestUnmarshalResponse_404(t *testing.T) {
	data := []byte("HTTP/1.1 404 Not Found\r\nContent-Type: text/html\r\nContent-Length: 18\r\n\r\n<h1>Not Found</h1>")

	resp, err := UnmarshalResponse(data)
	if err != nil {
		t.Fatalf("UnmarshalResponse() error = %v", err)
	}

	if resp.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", resp.StatusCode)
	}
	if resp.Reason != "Not Found" {
		t.Errorf("Reason = %q, want Not Found", resp.Reason)
	}
	if string(resp.Body) != "<h1>Not Found</h1>" {
		t.Errorf("Body = %q, want <h1>Not Found</h1>", string(resp.Body))
	}
}

func TestUnmarshalResponse_NoReasonPhrase(t *testing.T) {
	data := []byte("HTTP/1.1 200\r\nContent-Length: 0\r\n\r\n")

	resp, err := UnmarshalResponse(data)
	if err != nil {
		t.Fatalf("UnmarshalResponse() error = %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if resp.Reason != "" {
		t.Errorf("Reason = %q, want empty", resp.Reason)
	}
}

func TestUnmarshal_ChunkedBody(t *testing.T) {
	data := []byte("HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n" +
		"5\r\nHello\r\n" +
		"7\r\n, World\r\n" +
		"0\r\n\r\n")

	resp, err := UnmarshalResponse(data)
	if err != nil {
		t.Fatalf("UnmarshalResponse() error = %v", err)
	}

	if string(resp.Body) != "Hello, World" {
		t.Errorf("Body = %q, want Hello, World", string(resp.Body))
	}
}

func TestUnmarshal_HeaderOWS(t *testing.T) {
	data := []byte("GET / HTTP/1.1\r\nHost:   example.com   \r\n\r\n")

	req, err := UnmarshalRequest(data)
	if err != nil {
		t.Fatalf("UnmarshalRequest() error = %v", err)
	}

	if req.Headers.Get("Host") != "example.com" {
		t.Errorf("Host = %q, want example.com (trimmed OWS)", req.Headers.Get("Host"))
	}
}

func TestUnmarshal_HeaderObsFold(t *testing.T) {
	data := []byte("GET / HTTP/1.1\r\nX-Long: value1\r\n value2\r\nHost: example.com\r\n\r\n")

	req, err := UnmarshalRequest(data)
	if err != nil {
		t.Fatalf("UnmarshalRequest() error = %v", err)
	}

	if req.Headers.Get("X-Long") != "value1 value2" {
		t.Errorf("X-Long = %q, want 'value1 value2'", req.Headers.Get("X-Long"))
	}
}

func TestUnmarshal_BareLF(t *testing.T) {
	// Accept bare LF for robustness
	data := []byte("GET / HTTP/1.1\nHost: example.com\n\n")

	req, err := UnmarshalRequest(data)
	if err != nil {
		t.Fatalf("UnmarshalRequest() error = %v", err)
	}

	if req.Method != "GET" {
		t.Errorf("Method = %q, want GET", req.Method)
	}
}

func TestUnmarshal_TypeMismatch(t *testing.T) {
	// Try to unmarshal response into request
	data := []byte("HTTP/1.1 200 OK\r\n\r\n")
	req := &Request{}
	err := Unmarshal(data, req)
	if err == nil {
		t.Error("expected error for type mismatch")
	}

	// Try to unmarshal request into response
	data = []byte("GET / HTTP/1.1\r\n\r\n")
	resp := &Response{}
	err = Unmarshal(data, resp)
	if err == nil {
		t.Error("expected error for type mismatch")
	}
}

func TestUnmarshal_UnsupportedType(t *testing.T) {
	data := []byte("GET / HTTP/1.1\r\n\r\n")
	var s string
	err := Unmarshal(data, &s)
	if err == nil {
		t.Error("expected error for unsupported type")
	}
}

func TestUnmarshal_Nil(t *testing.T) {
	err := Unmarshal([]byte("GET / HTTP/1.1\r\n\r\n"), nil)
	if err == nil {
		t.Error("expected error for nil target")
	}
}

func TestDetectMessageType(t *testing.T) {
	tests := []struct {
		data []byte
		want string
	}{
		{[]byte("GET / HTTP/1.1\r\n"), "request"},
		{[]byte("POST /api HTTP/1.1\r\n"), "request"},
		{[]byte("HTTP/1.1 200 OK\r\n"), "response"},
		{[]byte("HTTP/1.0 404 Not Found\r\n"), "response"},
	}

	for _, tt := range tests {
		got := DetectMessageType(tt.data)
		if got != tt.want {
			t.Errorf("DetectMessageType(%q) = %q, want %q", tt.data, got, tt.want)
		}
	}
}

func TestUnmarshal_MultipleHeaders_SameKey(t *testing.T) {
	data := []byte("HTTP/1.1 200 OK\r\nSet-Cookie: a=1\r\nSet-Cookie: b=2\r\nContent-Length: 0\r\n\r\n")

	resp, err := UnmarshalResponse(data)
	if err != nil {
		t.Fatalf("UnmarshalResponse() error = %v", err)
	}

	vals := resp.Headers.Values("Set-Cookie")
	if len(vals) != 2 {
		t.Fatalf("Set-Cookie count = %d, want 2", len(vals))
	}
	if vals[0] != "a=1" || vals[1] != "b=2" {
		t.Errorf("Set-Cookie values = %v, want [a=1 b=2]", vals)
	}
}

func TestUnmarshal_WhitespaceBeforeColon(t *testing.T) {
	// RFC 9112: no whitespace between field-name and colon
	data := []byte("GET / HTTP/1.1\r\nHost : example.com\r\n\r\n")

	_, err := UnmarshalRequest(data)
	if err == nil {
		t.Error("expected error for whitespace before colon")
	}
}

func TestUnmarshal_BodyTruncated(t *testing.T) {
	data := []byte("POST / HTTP/1.1\r\nContent-Length: 100\r\n\r\nshort")

	_, err := UnmarshalRequest(data)
	if err == nil {
		t.Error("expected error for truncated body")
	}
}

// customUnmarshaler implements the Unmarshaler interface for testing the
// interface dispatch path in Unmarshal.
type customUnmarshaler struct {
	called bool
}

func (u *customUnmarshaler) UnmarshalHTTP(data []byte) error {
	u.called = true
	return nil
}

func TestUnmarshal_UnmarshalerInterface(t *testing.T) {
	// A type implementing Unmarshaler should have its UnmarshalHTTP method called.
	cu := &customUnmarshaler{}
	data := []byte("GET / HTTP/1.1\r\n\r\n")
	err := Unmarshal(data, cu)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !cu.called {
		t.Error("UnmarshalHTTP was not called on Unmarshaler")
	}
}
