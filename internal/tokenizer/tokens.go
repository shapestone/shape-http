// Package tokenizer provides HTTP tokenization using Shape's tokenizer framework.
package tokenizer

// Token type constants for HTTP format.
// HTTP is line-oriented, so tokens represent logical elements of HTTP messages.
const (
	// Start-line tokens
	TokenMethod     = "Method"     // GET, POST, PUT, DELETE, etc.
	TokenPath       = "Path"       // request-target /api/users?q=foo
	TokenVersion    = "Version"    // HTTP/1.0, HTTP/1.1
	TokenStatusCode = "StatusCode" // 200, 404, etc.
	TokenReason     = "Reason"     // OK, Not Found, etc.

	// Header tokens
	TokenHeaderName  = "HeaderName"  // field-name before colon
	TokenHeaderColon = "HeaderColon" // :
	TokenHeaderValue = "HeaderValue" // field-value after colon

	// Structural tokens
	TokenSP   = "SP"   // Space separator in start-line
	TokenCRLF = "CRLF" // Line ending \r\n or \n

	// Body tokens
	TokenBody = "Body" // Raw body content

	// Special
	TokenEOF = "EOF" // End of input
)
