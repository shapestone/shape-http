package http

import (
	"testing"

	"github.com/shapestone/shape-core/pkg/ast"
)

func TestRequestToNode_AndBack(t *testing.T) {
	req := &Request{
		Method:  "POST",
		Path:    "/api/users",
		Version: "HTTP/1.1",
		Headers: Headers{
			{Key: "Host", Value: "example.com"},
			{Key: "Content-Type", Value: "application/json"},
		},
		Body: []byte(`{"name":"Alice"}`),
	}

	node := RequestToNode(req)

	obj, ok := node.(*ast.ObjectNode)
	if !ok {
		t.Fatalf("expected ObjectNode, got %T", node)
	}

	props := obj.Properties()
	if lit := props["type"].(*ast.LiteralNode); lit.Value() != "request" {
		t.Errorf("type = %v, want request", lit.Value())
	}
	if lit := props["method"].(*ast.LiteralNode); lit.Value() != "POST" {
		t.Errorf("method = %v, want POST", lit.Value())
	}

	// Convert back
	req2, err := NodeToRequest(node)
	if err != nil {
		t.Fatalf("NodeToRequest() error = %v", err)
	}
	if req2.Method != "POST" {
		t.Errorf("Method = %q, want POST", req2.Method)
	}
	if req2.Path != "/api/users" {
		t.Errorf("Path = %q, want /api/users", req2.Path)
	}
	if string(req2.Body) != `{"name":"Alice"}` {
		t.Errorf("Body = %q", string(req2.Body))
	}
}

func TestResponseToNode_AndBack(t *testing.T) {
	resp := &Response{
		Version:    "HTTP/1.1",
		StatusCode: 404,
		Reason:     "Not Found",
		Headers: Headers{
			{Key: "Content-Type", Value: "text/html"},
		},
		Body: []byte("<h1>Not Found</h1>"),
	}

	node := ResponseToNode(resp)

	obj, ok := node.(*ast.ObjectNode)
	if !ok {
		t.Fatalf("expected ObjectNode, got %T", node)
	}

	props := obj.Properties()
	if lit := props["statusCode"].(*ast.LiteralNode); lit.Value() != int64(404) {
		t.Errorf("statusCode = %v, want 404", lit.Value())
	}

	// Convert back
	resp2, err := NodeToResponse(node)
	if err != nil {
		t.Fatalf("NodeToResponse() error = %v", err)
	}
	if resp2.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", resp2.StatusCode)
	}
	if resp2.Reason != "Not Found" {
		t.Errorf("Reason = %q, want Not Found", resp2.Reason)
	}
}

func TestNodeToRequest_NonObjectNode(t *testing.T) {
	// Passing a non-ObjectNode should return an error
	node := ast.NewLiteralNode("not an object", zeroPos)
	_, err := NodeToRequest(node)
	if err == nil {
		t.Error("NodeToRequest() = nil, want error for non-ObjectNode")
	}
}

func TestNodeToResponse_NonObjectNode(t *testing.T) {
	// Passing a non-ObjectNode should return an error
	node := ast.NewLiteralNode("not an object", zeroPos)
	_, err := NodeToResponse(node)
	if err == nil {
		t.Error("NodeToResponse() = nil, want error for non-ObjectNode")
	}
}

func TestNodeToRequest_NoBody(t *testing.T) {
	// Request without body property should have nil body
	req := &Request{
		Method:  "GET",
		Path:    "/",
		Version: "HTTP/1.1",
		Headers: Headers{{Key: "Host", Value: "example.com"}},
	}
	node := RequestToNode(req)
	req2, err := NodeToRequest(node)
	if err != nil {
		t.Fatalf("NodeToRequest() error = %v", err)
	}
	if req2.Body != nil {
		t.Errorf("Body = %v, want nil", req2.Body)
	}
}

func TestNodeToInterface(t *testing.T) {
	req := &Request{
		Method:  "GET",
		Path:    "/",
		Version: "HTTP/1.1",
		Headers: Headers{{Key: "Host", Value: "example.com"}},
	}
	node := RequestToNode(req)

	iface := NodeToInterface(node)
	m, ok := iface.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", iface)
	}

	if m["type"] != "request" {
		t.Errorf("type = %v, want request", m["type"])
	}
	if m["method"] != "GET" {
		t.Errorf("method = %v, want GET", m["method"])
	}
}

func TestNodeToInterface_Array(t *testing.T) {
	// Test with an ArrayDataNode to cover the array branch
	req := &Request{
		Method:  "GET",
		Path:    "/",
		Version: "HTTP/1.1",
		Headers: Headers{{Key: "Host", Value: "example.com"}},
	}
	node := RequestToNode(req)

	// The "headers" property is an ArrayDataNode
	obj := node.(*ast.ObjectNode)
	headersNode := obj.Properties()["headers"]

	iface := NodeToInterface(headersNode)
	arr, ok := iface.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", iface)
	}
	if len(arr) != 1 {
		t.Errorf("expected 1 header, got %d", len(arr))
	}
}

func TestNodeToInterface_UnknownType(t *testing.T) {
	// Passing a nil or unknown node type returns nil
	result := NodeToInterface(nil)
	if result != nil {
		t.Errorf("NodeToInterface(nil) = %v, want nil", result)
	}
}
