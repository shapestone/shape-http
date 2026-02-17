package http

import (
	"strings"
	"testing"

	"github.com/shapestone/shape-core/pkg/ast"
)

func TestParse_Request(t *testing.T) {
	input := "GET /api HTTP/1.1\r\nHost: example.com\r\n\r\n"
	node, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	obj, ok := node.(*ast.ObjectNode)
	if !ok {
		t.Fatalf("expected ObjectNode, got %T", node)
	}

	props := obj.Properties()
	typeLit := props["type"].(*ast.LiteralNode)
	if typeLit.Value() != "request" {
		t.Errorf("type = %v, want request", typeLit.Value())
	}
}

func TestParse_Response(t *testing.T) {
	input := "HTTP/1.1 200 OK\r\nContent-Length: 5\r\n\r\nHello"
	node, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	obj, ok := node.(*ast.ObjectNode)
	if !ok {
		t.Fatalf("expected ObjectNode, got %T", node)
	}

	props := obj.Properties()
	codeLit := props["statusCode"].(*ast.LiteralNode)
	if codeLit.Value() != int64(200) {
		t.Errorf("statusCode = %v, want 200", codeLit.Value())
	}
}

func TestParseReader(t *testing.T) {
	r := strings.NewReader("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")
	node, err := ParseReader(r)
	if err != nil {
		t.Fatalf("ParseReader() error = %v", err)
	}

	obj, ok := node.(*ast.ObjectNode)
	if !ok {
		t.Fatalf("expected ObjectNode, got %T", node)
	}

	props := obj.Properties()
	if lit := props["method"].(*ast.LiteralNode); lit.Value() != "GET" {
		t.Errorf("method = %v, want GET", lit.Value())
	}
}

func TestParse_Invalid(t *testing.T) {
	// Empty input should fail
	_, err := Parse("")
	if err == nil {
		t.Error("expected error for empty input")
	}

	// Malformed request line with no spaces should fail
	_, err = Parse("GETHTTP/1.1\r\n\r\n")
	if err == nil {
		t.Error("expected error for malformed request line")
	}
}
