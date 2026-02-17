package http

import (
	"bytes"
	"testing"
)

func TestEncoder_Request(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)

	req := &Request{
		Method:  "GET",
		Path:    "/api",
		Version: "HTTP/1.1",
		Headers: Headers{{Key: "Host", Value: "example.com"}},
	}

	err := enc.Encode(req)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	want := "GET /api HTTP/1.1\r\nHost: example.com\r\n\r\n"
	if buf.String() != want {
		t.Errorf("Encode() =\n%q\nwant:\n%q", buf.String(), want)
	}
}

func TestEncoder_Response(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)

	resp := &Response{
		Version:    "HTTP/1.1",
		StatusCode: 200,
		Reason:     "OK",
		Headers: Headers{
			{Key: "Content-Type", Value: "text/plain"},
			{Key: "Content-Length", Value: "5"},
		},
		Body: []byte("Hello"),
	}

	err := enc.Encode(resp)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	want := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 5\r\n\r\nHello"
	if buf.String() != want {
		t.Errorf("Encode() =\n%q\nwant:\n%q", buf.String(), want)
	}
}
