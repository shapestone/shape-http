package http

import (
	"fmt"
	"strconv"

	"github.com/shapestone/shape-core/pkg/ast"
	"github.com/shapestone/shape-http/internal/fastparser"
	"github.com/shapestone/shape-http/internal/parser"
)

// NodeToRequest converts an AST ObjectNode to a Request.
func NodeToRequest(node ast.SchemaNode) (*Request, error) {
	fpReq, err := parser.NodeToRequest(node)
	if err != nil {
		return nil, err
	}
	return &Request{
		Method:  fpReq.Method,
		Path:    fpReq.Path,
		Version: fpReq.Version,
		Headers: convertHeaders(fpReq.Headers),
		Body:    fpReq.Body,
	}, nil
}

// NodeToResponse converts an AST ObjectNode to a Response.
func NodeToResponse(node ast.SchemaNode) (*Response, error) {
	fpResp, err := parser.NodeToResponse(node)
	if err != nil {
		return nil, err
	}
	return &Response{
		Version:    fpResp.Version,
		StatusCode: fpResp.StatusCode,
		Reason:     fpResp.Reason,
		Headers:    convertHeaders(fpResp.Headers),
		Body:       fpResp.Body,
	}, nil
}

var zeroPos = ast.Position{}

// RequestToNode converts a Request to an AST ObjectNode.
func RequestToNode(req *Request) ast.SchemaNode {
	props := map[string]ast.SchemaNode{
		"type":    ast.NewLiteralNode("request", zeroPos),
		"method":  ast.NewLiteralNode(req.Method, zeroPos),
		"path":    ast.NewLiteralNode(req.Path, zeroPos),
		"version": ast.NewLiteralNode(req.Version, zeroPos),
		"headers": pubHeadersToNode(req.Headers),
	}
	if req.Body != nil {
		props["body"] = ast.NewLiteralNode(string(req.Body), zeroPos)
	}
	return ast.NewObjectNode(props, zeroPos)
}

// ResponseToNode converts a Response to an AST ObjectNode.
func ResponseToNode(resp *Response) ast.SchemaNode {
	props := map[string]ast.SchemaNode{
		"type":       ast.NewLiteralNode("response", zeroPos),
		"version":    ast.NewLiteralNode(resp.Version, zeroPos),
		"statusCode": ast.NewLiteralNode(int64(resp.StatusCode), zeroPos),
		"reason":     ast.NewLiteralNode(resp.Reason, zeroPos),
		"headers":    pubHeadersToNode(resp.Headers),
	}
	if resp.Body != nil {
		props["body"] = ast.NewLiteralNode(string(resp.Body), zeroPos)
	}
	return ast.NewObjectNode(props, zeroPos)
}

// NodeToInterface converts an AST node to native Go types.
func NodeToInterface(node ast.SchemaNode) interface{} {
	switch n := node.(type) {
	case *ast.LiteralNode:
		return n.Value()
	case *ast.ArrayDataNode:
		elements := n.Elements()
		arr := make([]interface{}, len(elements))
		for i, elem := range elements {
			arr[i] = NodeToInterface(elem)
		}
		return arr
	case *ast.ObjectNode:
		props := n.Properties()
		m := make(map[string]interface{}, len(props))
		for k, v := range props {
			m[k] = NodeToInterface(v)
		}
		return m
	default:
		return nil
	}
}

func pubHeadersToNode(headers Headers) ast.SchemaNode {
	elements := make([]ast.SchemaNode, len(headers))
	for i, h := range headers {
		elements[i] = ast.NewObjectNode(map[string]ast.SchemaNode{
			"key":   ast.NewLiteralNode(h.Key, zeroPos),
			"value": ast.NewLiteralNode(h.Value, zeroPos),
		}, zeroPos)
	}
	return ast.NewArrayDataNode(elements, zeroPos)
}

// nodeToInternalHeaders converts an AST headers array to internal fastparser headers.
func nodeToInternalHeaders(node ast.SchemaNode) ([]fastparser.Header, error) {
	arr, ok := node.(*ast.ArrayDataNode)
	if !ok {
		return nil, fmt.Errorf("expected ArrayDataNode for headers, got %T", node)
	}
	elements := arr.Elements()
	headers := make([]fastparser.Header, 0, len(elements))
	for _, elem := range elements {
		obj, ok := elem.(*ast.ObjectNode)
		if !ok {
			continue
		}
		props := obj.Properties()
		var h fastparser.Header
		if v, ok := props["key"]; ok {
			if lit, ok := v.(*ast.LiteralNode); ok {
				h.Key, _ = lit.Value().(string)
			}
		}
		if v, ok := props["value"]; ok {
			if lit, ok := v.(*ast.LiteralNode); ok {
				h.Value, _ = lit.Value().(string)
			}
		}
		headers = append(headers, h)
	}
	return headers, nil
}

// nodeToStatusCode extracts the status code from a literal node.
func nodeToStatusCode(node ast.SchemaNode) int {
	lit, ok := node.(*ast.LiteralNode)
	if !ok {
		return 0
	}
	switch code := lit.Value().(type) {
	case int64:
		return int(code)
	case float64:
		return int(code)
	case string:
		n, _ := strconv.Atoi(code)
		return n
	}
	return 0
}
