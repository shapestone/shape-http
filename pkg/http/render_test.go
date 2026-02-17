package http

import (
	"testing"
)

func TestRender_Request(t *testing.T) {
	input := "GET /api HTTP/1.1\r\nHost: example.com\r\n\r\n"
	node, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	data, err := Render(node)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if string(data) != input {
		t.Errorf("Render() =\n%q\nwant:\n%q", string(data), input)
	}
}

func TestRender_Response(t *testing.T) {
	input := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 5\r\n\r\nHello"
	node, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	data, err := Render(node)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if string(data) != input {
		t.Errorf("Render() =\n%q\nwant:\n%q", string(data), input)
	}
}

func TestRender_ResponseWithBody(t *testing.T) {
	input := "HTTP/1.1 404 Not Found\r\nContent-Type: text/html\r\nContent-Length: 18\r\n\r\n<h1>Not Found</h1>"
	node, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	data, err := Render(node)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if string(data) != input {
		t.Errorf("Render() =\n%q\nwant:\n%q", string(data), input)
	}
}
