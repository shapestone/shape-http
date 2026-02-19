// Package parser implements an AST parser for HTTP/1.1 messages.
// It produces shape-core AST nodes (ObjectNode, LiteralNode, ArrayDataNode)
// from HTTP wire-format input.
//
// The HTTP message is mapped to an ObjectNode with the following structure:
//
// Request:
//
//	{ "type": "request", "method": "POST", "path": "/api",
//	  "version": "HTTP/1.1",
//	  "headers": [{"key": "Host", "value": "example.com"}, ...],
//	  "body": "..." }
//
// Response:
//
//	{ "type": "response", "version": "HTTP/1.1", "statusCode": 200,
//	  "reason": "OK",
//	  "headers": [{"key": "Content-Type", "value": "text/plain"}, ...],
//	  "body": "..." }
package parser

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/shapestone/shape-core/pkg/ast"
	"github.com/shapestone/shape-http/internal/fastparser"
)

var zeroPos = ast.Position{}

// Parser produces AST nodes from HTTP wire-format data.
type Parser struct {
	data []byte
}

// NewParser creates a new AST parser for the given input.
func NewParser(data []byte) *Parser {
	return &Parser{data: data}
}

// Parse parses the HTTP message and returns an AST ObjectNode.
func (p *Parser) Parse() (ast.SchemaNode, error) {
	if bytes.HasPrefix(p.data, []byte("HTTP/")) {
		return p.parseResponse()
	}
	return p.parseRequest()
}

func (p *Parser) parseRequest() (ast.SchemaNode, error) {
	fp := fastparser.NewParser(p.data)
	req, err := fp.ParseRequest()
	if err != nil {
		return nil, err
	}
	return requestToNode(req), nil
}

func (p *Parser) parseResponse() (ast.SchemaNode, error) {
	fp := fastparser.NewParser(p.data)
	resp, err := fp.ParseResponse()
	if err != nil {
		return nil, err
	}
	return responseToNode(resp), nil
}

func requestToNode(req *fastparser.Request) ast.SchemaNode {
	props := map[string]ast.SchemaNode{
		"type":    ast.NewLiteralNode("request", zeroPos),
		"method":  ast.NewLiteralNode(req.Method, zeroPos),
		"path":    ast.NewLiteralNode(req.Path, zeroPos),
		"version": ast.NewLiteralNode(req.Version, zeroPos),
		"headers": headersToNode(req.Headers),
	}

	if req.Scheme != "" {
		props["scheme"] = ast.NewLiteralNode(req.Scheme, zeroPos)
	}
	if req.Body != nil {
		props["body"] = ast.NewLiteralNode(string(req.Body), zeroPos)
	}

	return ast.NewObjectNode(props, zeroPos)
}

func responseToNode(resp *fastparser.Response) ast.SchemaNode {
	props := map[string]ast.SchemaNode{
		"type":       ast.NewLiteralNode("response", zeroPos),
		"version":    ast.NewLiteralNode(resp.Version, zeroPos),
		"statusCode": ast.NewLiteralNode(int64(resp.StatusCode), zeroPos),
		"reason":     ast.NewLiteralNode(resp.Reason, zeroPos),
		"headers":    headersToNode(resp.Headers),
	}

	if resp.Body != nil {
		props["body"] = ast.NewLiteralNode(string(resp.Body), zeroPos)
	}

	return ast.NewObjectNode(props, zeroPos)
}

func headersToNode(headers []fastparser.Header) ast.SchemaNode {
	elements := make([]ast.SchemaNode, len(headers))
	for i, h := range headers {
		elements[i] = ast.NewObjectNode(map[string]ast.SchemaNode{
			"key":   ast.NewLiteralNode(h.Key, zeroPos),
			"value": ast.NewLiteralNode(h.Value, zeroPos),
		}, zeroPos)
	}
	return ast.NewArrayDataNode(elements, zeroPos)
}

// NodeToRequest converts an AST ObjectNode back to a fastparser.Request.
func NodeToRequest(node ast.SchemaNode) (*fastparser.Request, error) {
	obj, ok := node.(*ast.ObjectNode)
	if !ok {
		return nil, fmt.Errorf("expected ObjectNode, got %T", node)
	}

	props := obj.Properties()
	req := &fastparser.Request{}

	if v, ok := props["method"]; ok {
		if lit, ok := v.(*ast.LiteralNode); ok {
			req.Method, _ = lit.Value().(string)
		}
	}
	if v, ok := props["path"]; ok {
		if lit, ok := v.(*ast.LiteralNode); ok {
			req.Path, _ = lit.Value().(string)
		}
	}
	if v, ok := props["version"]; ok {
		if lit, ok := v.(*ast.LiteralNode); ok {
			req.Version, _ = lit.Value().(string)
		}
	}
	if v, ok := props["scheme"]; ok {
		if lit, ok := v.(*ast.LiteralNode); ok {
			req.Scheme, _ = lit.Value().(string)
		}
	}
	if v, ok := props["headers"]; ok {
		hdrs, err := nodeToHeaders(v)
		if err != nil {
			return nil, err
		}
		req.Headers = hdrs
	}
	if v, ok := props["body"]; ok {
		if lit, ok := v.(*ast.LiteralNode); ok {
			if s, ok := lit.Value().(string); ok {
				req.Body = []byte(s)
			}
		}
	}

	return req, nil
}

// NodeToResponse converts an AST ObjectNode back to a fastparser.Response.
func NodeToResponse(node ast.SchemaNode) (*fastparser.Response, error) {
	obj, ok := node.(*ast.ObjectNode)
	if !ok {
		return nil, fmt.Errorf("expected ObjectNode, got %T", node)
	}

	props := obj.Properties()
	resp := &fastparser.Response{}

	if v, ok := props["version"]; ok {
		if lit, ok := v.(*ast.LiteralNode); ok {
			resp.Version, _ = lit.Value().(string)
		}
	}
	if v, ok := props["statusCode"]; ok {
		if lit, ok := v.(*ast.LiteralNode); ok {
			switch code := lit.Value().(type) {
			case int64:
				resp.StatusCode = int(code)
			case float64:
				resp.StatusCode = int(code)
			case string:
				resp.StatusCode, _ = strconv.Atoi(code)
			}
		}
	}
	if v, ok := props["reason"]; ok {
		if lit, ok := v.(*ast.LiteralNode); ok {
			resp.Reason, _ = lit.Value().(string)
		}
	}
	if v, ok := props["headers"]; ok {
		hdrs, err := nodeToHeaders(v)
		if err != nil {
			return nil, err
		}
		resp.Headers = hdrs
	}
	if v, ok := props["body"]; ok {
		if lit, ok := v.(*ast.LiteralNode); ok {
			if s, ok := lit.Value().(string); ok {
				resp.Body = []byte(s)
			}
		}
	}

	return resp, nil
}

func nodeToHeaders(node ast.SchemaNode) ([]fastparser.Header, error) {
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
