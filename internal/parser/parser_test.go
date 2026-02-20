package parser

import (
	"testing"

	"github.com/shapestone/shape-core/pkg/ast"
	"github.com/shapestone/shape-http/internal/fastparser"
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

func TestParse_InvalidRequest(t *testing.T) {
	// Malformed request → parseRequest error path
	data := []byte("NOTHTTP\r\n\r\n")
	p := NewParser(data)
	_, err := p.Parse()
	if err == nil {
		t.Error("Parse() = nil, want error for invalid request")
	}
}

func TestParse_InvalidResponse(t *testing.T) {
	// Malformed response (starts with HTTP/ but invalid status)
	data := []byte("HTTP/1.1 abc Bad\r\n\r\n")
	p := NewParser(data)
	_, err := p.Parse()
	if err == nil {
		t.Error("Parse() = nil, want error for invalid response")
	}
}

func TestNodeToRequest_NonObjectNode(t *testing.T) {
	node := ast.NewLiteralNode("not an object", zeroPos)
	_, err := NodeToRequest(node)
	if err == nil {
		t.Error("NodeToRequest() = nil, want error for non-ObjectNode")
	}
}

func TestNodeToResponse_NonObjectNode(t *testing.T) {
	node := ast.NewLiteralNode("not an object", zeroPos)
	_, err := NodeToResponse(node)
	if err == nil {
		t.Error("NodeToResponse() = nil, want error for non-ObjectNode")
	}
}

func TestNodeToRequest_HeadersNotArray(t *testing.T) {
	// "headers" property is not an ArrayDataNode — should error
	node := ast.NewObjectNode(map[string]ast.SchemaNode{
		"type":    ast.NewLiteralNode("request", zeroPos),
		"method":  ast.NewLiteralNode("GET", zeroPos),
		"path":    ast.NewLiteralNode("/", zeroPos),
		"version": ast.NewLiteralNode("HTTP/1.1", zeroPos),
		"headers": ast.NewLiteralNode("not an array", zeroPos), // wrong type
	}, zeroPos)
	_, err := NodeToRequest(node)
	if err == nil {
		t.Error("NodeToRequest() = nil, want error when headers is not ArrayDataNode")
	}
}

func TestNodeToResponse_StatusCodeAsString(t *testing.T) {
	// statusCode as a string value — exercises the string case in type switch
	node := ast.NewObjectNode(map[string]ast.SchemaNode{
		"type":       ast.NewLiteralNode("response", zeroPos),
		"version":    ast.NewLiteralNode("HTTP/1.1", zeroPos),
		"statusCode": ast.NewLiteralNode("200", zeroPos), // string, not int64
		"reason":     ast.NewLiteralNode("OK", zeroPos),
		"headers":    ast.NewArrayDataNode(nil, zeroPos),
	}, zeroPos)
	resp, err := NodeToResponse(node)
	if err != nil {
		t.Fatalf("NodeToResponse() error = %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
}

func TestNodeToHeaders_NonObjectElement(t *testing.T) {
	// Array element is not an ObjectNode — should be skipped (continue branch)
	node := ast.NewObjectNode(map[string]ast.SchemaNode{
		"type":    ast.NewLiteralNode("request", zeroPos),
		"method":  ast.NewLiteralNode("GET", zeroPos),
		"path":    ast.NewLiteralNode("/", zeroPos),
		"version": ast.NewLiteralNode("HTTP/1.1", zeroPos),
		"headers": ast.NewArrayDataNode([]ast.SchemaNode{
			ast.NewLiteralNode("not an object", zeroPos), // should be skipped
		}, zeroPos),
	}, zeroPos)
	req, err := NodeToRequest(node)
	if err != nil {
		t.Fatalf("NodeToRequest() error = %v", err)
	}
	// Non-object element should be skipped, resulting in 0 headers
	if len(req.Headers) != 0 {
		t.Errorf("Headers count = %d, want 0 (non-object element skipped)", len(req.Headers))
	}
}

func TestNodeToResponse_Float64StatusCode(t *testing.T) {
	// statusCode as float64 value — exercises the float64 case in the type switch
	node := ast.NewObjectNode(map[string]ast.SchemaNode{
		"type":       ast.NewLiteralNode("response", zeroPos),
		"version":    ast.NewLiteralNode("HTTP/1.1", zeroPos),
		"statusCode": ast.NewLiteralNode(float64(200), zeroPos),
		"reason":     ast.NewLiteralNode("OK", zeroPos),
		"headers":    ast.NewArrayDataNode(nil, zeroPos),
	}, zeroPos)
	resp, err := NodeToResponse(node)
	if err != nil {
		t.Fatalf("NodeToResponse() error = %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
}

func TestNodeToResponse_HeadersNotArray(t *testing.T) {
	// "headers" property is not an ArrayDataNode — should error
	node := ast.NewObjectNode(map[string]ast.SchemaNode{
		"type":       ast.NewLiteralNode("response", zeroPos),
		"version":    ast.NewLiteralNode("HTTP/1.1", zeroPos),
		"statusCode": ast.NewLiteralNode(int64(200), zeroPos),
		"reason":     ast.NewLiteralNode("OK", zeroPos),
		"headers":    ast.NewLiteralNode("not an array", zeroPos), // wrong type
	}, zeroPos)
	_, err := NodeToResponse(node)
	if err == nil {
		t.Error("NodeToResponse() = nil, want error when headers is not ArrayDataNode")
	}
}

func TestRequestToNode_WithScheme(t *testing.T) {
	// Exercise the `if req.Scheme != ""` branch in requestToNode and the
	// corresponding "scheme" property read in NodeToRequest.
	req := &fastparser.Request{
		Method:  "GET",
		Path:    "/api",
		Version: "HTTP/1.1",
		Scheme:  "https",
		Headers: []fastparser.Header{{Key: "Host", Value: "example.com"}},
	}
	node := requestToNode(req)
	obj, ok := node.(*ast.ObjectNode)
	if !ok {
		t.Fatalf("expected ObjectNode, got %T", node)
	}
	schemeProp, ok := obj.Properties()["scheme"]
	if !ok {
		t.Fatal("scheme key missing from ObjectNode")
	}
	lit, ok := schemeProp.(*ast.LiteralNode)
	if !ok {
		t.Fatalf("scheme expected LiteralNode, got %T", schemeProp)
	}
	if lit.Value() != "https" {
		t.Errorf("scheme = %v, want https", lit.Value())
	}

	// Also round-trip through NodeToRequest to cover the scheme-read branch.
	req2, err := NodeToRequest(node)
	if err != nil {
		t.Fatalf("NodeToRequest() error = %v", err)
	}
	if req2.Scheme != "https" {
		t.Errorf("Scheme = %q, want https", req2.Scheme)
	}
}

func TestRequestToNode_SchemeAbsent(t *testing.T) {
	// When Scheme is empty the "scheme" key must not appear in the ObjectNode.
	req := &fastparser.Request{
		Method:  "GET",
		Path:    "/",
		Version: "HTTP/1.1",
	}
	node := requestToNode(req)
	obj := node.(*ast.ObjectNode)
	if _, ok := obj.Properties()["scheme"]; ok {
		t.Error("scheme key present in ObjectNode but Scheme was empty")
	}
}
