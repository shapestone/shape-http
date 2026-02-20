package http

import (
	"github.com/shapestone/shape-core/pkg/ast"
	"github.com/shapestone/shape-http/internal/fastparser"
)

// UnmarshalLenient performs best-effort parsing of an HTTP message.
// It never returns an error for malformed input — instead it extracts
// whatever parts are valid and reports issues as Warnings.
//
// The returned ParseResult will have either Request or Response set
// (not both), depending on auto-detection. Warnings contains human-readable
// descriptions of any issues encountered. Partial is true if the message
// was incomplete or truncated.
//
// # Authentication
//
// Authentication headers are parsed as ordinary HTTP headers and are available
// via result.Request.Headers.Get. All standard schemes are supported:
//
//	result.Request.Headers.Get("Authorization")   // "Basic ...", "Bearer ...", "OAuth ..."
//	result.Request.Headers.Get("X-API-Key")       // "abc123def456"
//	result.Request.Headers.Get("X-Session-Token") // "sess_xyz789"
//
// Query-string API keys are preserved as part of result.Request.Path:
//
//	// GET /api/users?api_key=abc123  →  result.Request.Path = "/api/users?api_key=abc123"
//
// The lenient parser additionally accepts:
//   - Absolute-form request targets (e.g. "GET https://example.com/path HTTP/1.1"),
//     extracting the scheme into result.Request.Scheme and the host into
//     result.Request.Headers.Get("Host").
//   - Bare LF line endings in addition to CRLF.
//   - Missing HTTP version (defaults to "HTTP/1.1").
func UnmarshalLenient(data []byte) *ParseResult {
	lp := fastparser.NewLenientParser(data)
	internal := lp.Parse()

	result := &ParseResult{
		Warnings: internal.Warnings,
		Partial:  internal.Partial,
	}

	if internal.Request != nil {
		result.Request = &Request{
			Method:  internal.Request.Method,
			Path:    internal.Request.Path,
			Version: internal.Request.Version,
			Scheme:  internal.Request.Scheme,
			Headers: convertHeaders(internal.Request.Headers),
			Body:    internal.Request.Body,
		}
		// Check if body was incomplete
		for _, w := range internal.Warnings {
			if w == "message body is incomplete" {
				result.Partial = true
			}
		}
	}

	if internal.Response != nil {
		result.Response = &Response{
			Version:    internal.Response.Version,
			StatusCode: internal.Response.StatusCode,
			Reason:     internal.Response.Reason,
			Headers:    convertHeaders(internal.Response.Headers),
			Body:       internal.Response.Body,
		}
		for _, w := range internal.Warnings {
			if w == "message body is incomplete" {
				result.Partial = true
			}
		}
	}

	return result
}

// ParseLenient is the AST path equivalent of UnmarshalLenient.
// It returns an AST node (ObjectNode), a list of warnings, and an error.
// The error is only non-nil for truly unrecoverable situations (e.g., nil input
// that can't produce any node). For malformed HTTP messages, the AST will
// contain whatever was extractable and warnings will describe issues.
func ParseLenient(input string) (ast.SchemaNode, []string, error) {
	result := UnmarshalLenient([]byte(input))

	if result.Request != nil {
		node := RequestToNode(result.Request)
		return node, result.Warnings, nil
	}
	if result.Response != nil {
		node := ResponseToNode(result.Response)
		return node, result.Warnings, nil
	}

	// Empty/unparseable — return an empty object node
	return ast.NewObjectNode(map[string]ast.SchemaNode{
		"type": ast.NewLiteralNode("unknown", zeroPos),
	}, zeroPos), result.Warnings, nil
}
