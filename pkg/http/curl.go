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
// The leading "curl" word is optional — commands pasted without it (starting
// directly with a flag or URL) are accepted.
//
// # Authentication
//
// The following auth patterns are supported and produce the correct headers:
//
//	-u user:pass            → Authorization: Basic <base64(user:pass)>
//	-u :token               → Authorization: Basic <base64(:token)>  (GitHub pattern)
//	https://user:pass@host  → Authorization: Basic <base64(user:pass)>, Host: host
//	-H "Authorization: Bearer <token>"   → passed through as-is
//	-H "Authorization: OAuth ..."        → passed through as-is
//	-H "X-API-Key: <key>"               → passed through as-is
//	--digest                → silently ignored (digest challenge cannot be simulated)
//	--cert / --key          → silently ignored (TLS options, no file access)
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
//	-b / --cookie           Cookie header value → Cookie: <value>
//	-I / --head             Set method to HEAD
//	--http2                 Set version to HTTP/2
//	--http3                 Set version to HTTP/3
//	--http1.0               Set version to HTTP/1.0
//	--http1.1               Set version to HTTP/1.1
//
// # Compound short flags
//
// Multiple single-character flags may be combined into one token as curl
// allows (e.g. -sS, -vk, -sLk, -XPOST). Each character is expanded and
// processed individually before flag dispatch, so combined flags never
// produce spurious "unknown flag" warnings.
//
// # Silently ignored flags
//
//	-v / --verbose, -s / --silent, -S / --show-error,
//	-L / --location, --compressed, -k / --insecure,
//	-i / --include, -O, -o / --output, -A / --user-agent,
//	and other display/behaviour flags that do not affect the request.
//
// # URL fragments
//
// Fragments (#section) are stripped from the URL before building the
// request path because they are never sent over the wire.
//
// # Multi-line commands
//
// Lines ending with a backslash (\) are joined before parsing, so commands
// copied from a terminal work without modification. Remaining bare newlines
// (e.g. leading or trailing blank lines) are also treated as whitespace.
//
// # Comment and separator lines
//
// Lines whose first non-whitespace character is '#' are stripped before
// parsing, as are markdown separator lines consisting only of '-' characters
// (e.g. "---"). This means commands pasted from README files or API docs
// together with their surrounding commentary parse correctly.
//
// # URLs without a scheme
//
// If the URL has no "http://" or "https://" prefix the host is still
// extracted from the authority component (e.g. "example.com/api",
// "localhost:8080/path", "192.168.0.50/path" all produce the correct
// Host header and path).
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
	}

	return result
}
