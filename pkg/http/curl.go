package http

import "github.com/shapestone/shape-http/internal/fastparser"

// ParseCurl parses a curl command string and returns a ParseResult with
// best-effort extraction, matching the output format of UnmarshalLenient.
//
// The returned ParseResult always has Request set (never Response) because
// curl only issues requests. Warnings contains human-readable descriptions
// of any issues or unsupported flags encountered. Partial is true when the
// command could not be fully parsed (e.g., missing URL).
//
// # Supported flags
//
//	-X / --request          HTTP method
//	-H / --header           Request header (repeatable)
//	-d / --data             Request body
//	--data-raw              Request body (no special @file handling)
//	--data-binary           Request body (as-is)
//	-F / --form             multipart/form-data field (repeatable)
//	--data-urlencode        URL-encoded form field (repeatable)
//	-u / --user             Basic Auth → Authorization: Basic <base64>
//	--http2                 Set version to HTTP/2
//	--http3                 Set version to HTTP/3
//
// # Silently ignored flags
//
//	-v / --verbose, -s / --silent, -L / --location, --compressed,
//	-k / --insecure, -i / --include, -o / --output, -A / --user-agent,
//	and other display/behaviour flags that do not affect the request.
//
// # Multi-line commands
//
// Lines ending with a backslash (\) are joined before parsing, so commands
// copied from a terminal work without modification.
func ParseCurl(cmd string) *ParseResult {
	internal := fastparser.ParseCurl(cmd)

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
		for _, w := range internal.Warnings {
			if w == "message body is incomplete" {
				result.Partial = true
			}
		}
	}

	return result
}
