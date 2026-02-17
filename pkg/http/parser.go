package http

import (
	"io"

	"github.com/shapestone/shape-core/pkg/ast"
	"github.com/shapestone/shape-http/internal/parser"
)

// Parse parses HTTP wire format into an AST from a string.
//
// The input is a complete HTTP/1.1 message (request or response).
// Returns an ast.ObjectNode with properties matching the message type.
//
// For requests:
//
//	{ "type": "request", "method": "GET", "path": "/api",
//	  "version": "HTTP/1.1",
//	  "headers": [{"key": "Host", "value": "example.com"}, ...],
//	  "body": "..." }
//
// For responses:
//
//	{ "type": "response", "version": "HTTP/1.1", "statusCode": 200,
//	  "reason": "OK",
//	  "headers": [{"key": "Content-Type", "value": "text/plain"}, ...],
//	  "body": "..." }
func Parse(input string) (ast.SchemaNode, error) {
	p := parser.NewParser([]byte(input))
	return p.Parse()
}

// ParseReader reads all data from r and parses it as an HTTP message into an AST.
func ParseReader(r io.Reader) (ast.SchemaNode, error) {
	data, err := readAll(r)
	if err != nil {
		return nil, err
	}
	p := parser.NewParser(data)
	return p.Parse()
}
