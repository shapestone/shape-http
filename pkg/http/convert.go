package http

import (
	"github.com/shapestone/shape-core/pkg/ast"
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
