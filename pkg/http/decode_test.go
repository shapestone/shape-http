package http

import (
	"bytes"
	"testing"
)

func TestDecoder_Request(t *testing.T) {
	data := "GET /api HTTP/1.1\r\nHost: example.com\r\nContent-Length: 0\r\n\r\n"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	req := &Request{}
	err := dec.Decode(req)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if req.Method != "GET" {
		t.Errorf("Method = %q, want GET", req.Method)
	}
	if req.Path != "/api" {
		t.Errorf("Path = %q, want /api", req.Path)
	}
	if req.Version != "HTTP/1.1" {
		t.Errorf("Version = %q, want HTTP/1.1", req.Version)
	}
}

func TestDecoder_RequestWithBody(t *testing.T) {
	data := "POST /api HTTP/1.1\r\nHost: example.com\r\nContent-Length: 11\r\n\r\nhello world"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	req, err := dec.DecodeRequest()
	if err != nil {
		t.Fatalf("DecodeRequest() error = %v", err)
	}

	if string(req.Body) != "hello world" {
		t.Errorf("Body = %q, want hello world", string(req.Body))
	}
}

func TestDecoder_Response(t *testing.T) {
	data := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 5\r\n\r\nHello"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	resp := &Response{}
	err := dec.Decode(resp)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
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

func TestDecoder_ResponseWithChunkedBody(t *testing.T) {
	data := "HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nHello\r\n7\r\n, World\r\n0\r\n\r\n"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	resp, err := dec.DecodeResponse()
	if err != nil {
		t.Fatalf("DecodeResponse() error = %v", err)
	}

	if string(resp.Body) != "Hello, World" {
		t.Errorf("Body = %q, want Hello, World", string(resp.Body))
	}
}

func TestDecoder_TypeMismatch(t *testing.T) {
	data := "HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	req := &Request{}
	err := dec.Decode(req)
	if err == nil {
		t.Error("expected error for type mismatch")
	}
}

func TestDecoder_DecodeRequest_Convenience(t *testing.T) {
	data := "GET / HTTP/1.1\r\nHost: example.com\r\nContent-Length: 0\r\n\r\n"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	req, err := dec.DecodeRequest()
	if err != nil {
		t.Fatalf("DecodeRequest() error = %v", err)
	}

	if req.Method != "GET" {
		t.Errorf("Method = %q, want GET", req.Method)
	}
}

func TestDecoder_DecodeResponse_Convenience(t *testing.T) {
	data := "HTTP/1.1 404 Not Found\r\nContent-Length: 9\r\n\r\nNot Found"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	resp, err := dec.DecodeResponse()
	if err != nil {
		t.Fatalf("DecodeResponse() error = %v", err)
	}

	if resp.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", resp.StatusCode)
	}
	if string(resp.Body) != "Not Found" {
		t.Errorf("Body = %q, want 'Not Found'", string(resp.Body))
	}
}
