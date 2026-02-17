package http

import (
	"fmt"

	"github.com/shapestone/shape-core/pkg/ast"
)

// Render converts an AST node (from Parse) back to HTTP wire format bytes.
//
// The node must be an ObjectNode with a "type" property of "request" or "response",
// as produced by Parse() or ParseReader().
func Render(node ast.SchemaNode) ([]byte, error) {
	obj, ok := node.(*ast.ObjectNode)
	if !ok {
		return nil, fmt.Errorf("http: Render: expected ObjectNode, got %T", node)
	}

	props := obj.Properties()
	typeProp, ok := props["type"]
	if !ok {
		return nil, fmt.Errorf("http: Render: missing 'type' property")
	}

	typeLit, ok := typeProp.(*ast.LiteralNode)
	if !ok {
		return nil, fmt.Errorf("http: Render: 'type' is not a literal")
	}

	msgType, ok := typeLit.Value().(string)
	if !ok {
		return nil, fmt.Errorf("http: Render: 'type' is not a string")
	}

	switch msgType {
	case "request":
		req, err := NodeToRequest(node)
		if err != nil {
			return nil, fmt.Errorf("http: Render: %w", err)
		}
		return Marshal(req)

	case "response":
		resp, err := NodeToResponse(node)
		if err != nil {
			return nil, fmt.Errorf("http: Render: %w", err)
		}
		return Marshal(resp)

	default:
		return nil, fmt.Errorf("http: Render: unknown message type %q", msgType)
	}
}
