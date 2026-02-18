package http

import (
	"testing"

	"github.com/shapestone/shape-core/pkg/ast"
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

func TestRender_NonObjectNode(t *testing.T) {
	// Passing a non-ObjectNode should return an error
	node := ast.NewLiteralNode("hello", zeroPos)
	_, err := Render(node)
	if err == nil {
		t.Error("expected error for non-ObjectNode input")
	}
}

func TestRender_MissingTypeProperty(t *testing.T) {
	// ObjectNode without "type" property
	node := ast.NewObjectNode(map[string]ast.SchemaNode{
		"method": ast.NewLiteralNode("GET", zeroPos),
	}, zeroPos)
	_, err := Render(node)
	if err == nil {
		t.Error("expected error for missing 'type' property")
	}
}

func TestRender_TypeNotLiteral(t *testing.T) {
	// "type" property is an array, not a literal
	node := ast.NewObjectNode(map[string]ast.SchemaNode{
		"type": ast.NewArrayDataNode(nil, zeroPos),
	}, zeroPos)
	_, err := Render(node)
	if err == nil {
		t.Error("expected error when 'type' is not a literal")
	}
}

func TestRender_TypeNotString(t *testing.T) {
	// "type" property is a literal but not a string (e.g., int64)
	node := ast.NewObjectNode(map[string]ast.SchemaNode{
		"type": ast.NewLiteralNode(int64(42), zeroPos),
	}, zeroPos)
	_, err := Render(node)
	if err == nil {
		t.Error("expected error when 'type' literal is not a string")
	}
}

func TestRender_UnknownType(t *testing.T) {
	// "type" is a valid string but unknown (not "request" or "response")
	node := ast.NewObjectNode(map[string]ast.SchemaNode{
		"type": ast.NewLiteralNode("unknown", zeroPos),
	}, zeroPos)
	_, err := Render(node)
	if err == nil {
		t.Error("expected error for unknown message type")
	}
}

func TestRender_RequestConversionError(t *testing.T) {
	// type="request" but headers is not an ArrayDataNode → NodeToRequest fails → Render returns error
	node := ast.NewObjectNode(map[string]ast.SchemaNode{
		"type":    ast.NewLiteralNode("request", zeroPos),
		"method":  ast.NewLiteralNode("GET", zeroPos),
		"path":    ast.NewLiteralNode("/", zeroPos),
		"version": ast.NewLiteralNode("HTTP/1.1", zeroPos),
		"headers": ast.NewLiteralNode("not an array", zeroPos), // wrong type
	}, zeroPos)
	_, err := Render(node)
	if err == nil {
		t.Error("expected error when NodeToRequest fails during Render")
	}
}

func TestRender_ResponseConversionError(t *testing.T) {
	// type="response" but headers is not an ArrayDataNode → NodeToResponse fails → Render returns error
	node := ast.NewObjectNode(map[string]ast.SchemaNode{
		"type":       ast.NewLiteralNode("response", zeroPos),
		"version":    ast.NewLiteralNode("HTTP/1.1", zeroPos),
		"statusCode": ast.NewLiteralNode(int64(200), zeroPos),
		"reason":     ast.NewLiteralNode("OK", zeroPos),
		"headers":    ast.NewLiteralNode("not an array", zeroPos), // wrong type
	}, zeroPos)
	_, err := Render(node)
	if err == nil {
		t.Error("expected error when NodeToResponse fails during Render")
	}
}
