package parser

import (
	"testing"

	"github.com/shapestone/shape-core/pkg/ast"
)

func TestParse_Request(t *testing.T) {
	data := []byte("GET /api/users HTTP/1.1\r\nHost: example.com\r\n\r\n")
	p := NewParser(data)
	node, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	obj, ok := node.(*ast.ObjectNode)
	if !ok {
		t.Fatalf("expected ObjectNode, got %T", node)
	}

	props := obj.Properties()

	// Check type
	typeLit, ok := props["type"].(*ast.LiteralNode)
	if !ok || typeLit.Value() != "request" {
		t.Errorf("type = %v, want 'request'", props["type"])
	}

	// Check method
	methodLit, ok := props["method"].(*ast.LiteralNode)
	if !ok || methodLit.Value() != "GET" {
		t.Errorf("method = %v, want 'GET'", props["method"])
	}

	// Check path
	pathLit, ok := props["path"].(*ast.LiteralNode)
	if !ok || pathLit.Value() != "/api/users" {
		t.Errorf("path = %v, want '/api/users'", props["path"])
	}

	// Check headers array
	headers, ok := props["headers"].(*ast.ArrayDataNode)
	if !ok {
		t.Fatalf("headers expected ArrayDataNode, got %T", props["headers"])
	}
	if len(headers.Elements()) != 1 {
		t.Errorf("headers count = %d, want 1", len(headers.Elements()))
	}
}

func TestParse_Response(t *testing.T) {
	data := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 5\r\n\r\nHello")
	p := NewParser(data)
	node, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	obj, ok := node.(*ast.ObjectNode)
	if !ok {
		t.Fatalf("expected ObjectNode, got %T", node)
	}

	props := obj.Properties()

	typeLit := props["type"].(*ast.LiteralNode)
	if typeLit.Value() != "response" {
		t.Errorf("type = %v, want 'response'", typeLit.Value())
	}

	codeLit := props["statusCode"].(*ast.LiteralNode)
	if codeLit.Value() != int64(200) {
		t.Errorf("statusCode = %v, want 200", codeLit.Value())
	}

	bodyLit := props["body"].(*ast.LiteralNode)
	if bodyLit.Value() != "Hello" {
		t.Errorf("body = %v, want 'Hello'", bodyLit.Value())
	}
}

func TestNodeToRequest_RoundTrip(t *testing.T) {
	data := []byte("POST /api HTTP/1.1\r\nHost: example.com\r\nContent-Length: 4\r\n\r\ntest")
	p := NewParser(data)
	node, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	req, err := NodeToRequest(node)
	if err != nil {
		t.Fatalf("NodeToRequest() error = %v", err)
	}

	if req.Method != "POST" {
		t.Errorf("Method = %q, want POST", req.Method)
	}
	if req.Path != "/api" {
		t.Errorf("Path = %q, want /api", req.Path)
	}
	if string(req.Body) != "test" {
		t.Errorf("Body = %q, want test", string(req.Body))
	}
}

func TestNodeToResponse_RoundTrip(t *testing.T) {
	data := []byte("HTTP/1.1 404 Not Found\r\nContent-Length: 9\r\n\r\nNot Found")
	p := NewParser(data)
	node, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	resp, err := NodeToResponse(node)
	if err != nil {
		t.Fatalf("NodeToResponse() error = %v", err)
	}

	if resp.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", resp.StatusCode)
	}
	if resp.Reason != "Not Found" {
		t.Errorf("Reason = %q, want 'Not Found'", resp.Reason)
	}
	if string(resp.Body) != "Not Found" {
		t.Errorf("Body = %q, want 'Not Found'", string(resp.Body))
	}
}
