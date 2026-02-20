package fastparser

// curl_parse_test.go — targeted unit tests for the curlParser.parse() branches
// not covered by curl_test.go: empty-after-strip, shellSplit error, next()
// false, -d @file, -b, -u no-colon, --http3/1.0/1.1, -I, discard flags,
// second positional, URL userinfo, parseCurlURL bare-host-slash, shellSplit
// trailing backslash.

import (
	"strings"
	"testing"
)

// ── empty after stripNonCurlLines ──────────────────────────────────────────

func TestParseCurl_EmptyAfterStripComments(t *testing.T) {
	// Input that becomes empty once comment and separator lines are removed.
	cmd := "# just a comment\n---\n# another comment"
	result := ParseCurl(cmd)
	if !result.Partial {
		t.Error("expected Partial=true for comment-only input")
	}
	if result.Request != nil {
		t.Error("expected nil Request for comment-only input")
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "empty curl command") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'empty curl command' warning, got %v", result.Warnings)
	}
}

// ── shellSplit error (unclosed quote) ─────────────────────────────────────

func TestParseCurl_UnclosedQuote(t *testing.T) {
	// Unclosed single quote → shellSplit returns error → partial result.
	result := ParseCurl(`curl -H 'X-Unclosed https://example.com/`)
	if !result.Partial {
		t.Error("expected Partial=true for unclosed quote")
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "malformed curl command") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'malformed curl command' warning, got %v", result.Warnings)
	}
}

// ── next() returning false (flag with no following arg) ───────────────────

func TestParseCurl_FlagWithNoArg(t *testing.T) {
	// -X at the very end of the token list → next() returns ("", false).
	// The method defaults to GET (no body, no explicit method).
	result := ParseCurl("curl https://example.com/ -X")
	if result.Request == nil {
		t.Fatal("expected non-nil Request")
	}
	if result.Request.Method != "GET" {
		t.Errorf("Method = %q, want GET (next() false path)", result.Request.Method)
	}
}

// ── -d @file file upload warning ──────────────────────────────────────────

func TestParseCurl_DataFileUpload(t *testing.T) {
	// -d @filename should warn and skip the body.
	result := ParseCurl("curl -d @payload.json https://example.com/api")
	if result.Request == nil {
		t.Fatal("expected non-nil Request")
	}
	if result.Request.Body != nil {
		t.Errorf("expected nil Body for @file upload, got %q", result.Request.Body)
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "file upload") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected file-upload warning, got %v", result.Warnings)
	}
}

// ── -b / --cookie ────────────────────────────────────────────────────────

func TestParseCurl_CookieFlag(t *testing.T) {
	result := ParseCurl(`curl -b "session=abc123; user=42" https://example.com/api`)
	if result.Request == nil {
		t.Fatal("expected non-nil Request")
	}
	var cookie string
	for _, h := range result.Request.Headers {
		if h.Key == "Cookie" {
			cookie = h.Value
			break
		}
	}
	if cookie != "session=abc123; user=42" {
		t.Errorf("Cookie = %q, want %q", cookie, "session=abc123; user=42")
	}
}

func TestParseCurl_CookieLongFlag(t *testing.T) {
	result := ParseCurl("curl --cookie token=xyz https://example.com/api")
	if result.Request == nil {
		t.Fatal("expected non-nil Request")
	}
	var cookie string
	for _, h := range result.Request.Headers {
		if h.Key == "Cookie" {
			cookie = h.Value
			break
		}
	}
	if cookie != "token=xyz" {
		t.Errorf("Cookie = %q, want token=xyz", cookie)
	}
}

// ── -u username only (no colon) ───────────────────────────────────────────

func TestParseCurl_UserNoColon(t *testing.T) {
	// -u with no colon should warn and still produce an Authorization header.
	result := ParseCurl("curl -u myuser https://example.com/api")
	if result.Request == nil {
		t.Fatal("expected non-nil Request")
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "no colon") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'no colon' warning for -u without colon, got %v", result.Warnings)
	}
	var auth string
	for _, h := range result.Request.Headers {
		if h.Key == "Authorization" {
			auth = h.Value
			break
		}
	}
	if !strings.HasPrefix(auth, "Basic ") {
		t.Errorf("Authorization = %q, want Basic ...", auth)
	}
}

// ── HTTP version flags ────────────────────────────────────────────────────

func TestParseCurl_HTTP3(t *testing.T) {
	result := ParseCurl("curl --http3 https://example.com/")
	if result.Request == nil {
		t.Fatal("expected non-nil Request")
	}
	if result.Request.Version != "HTTP/3" {
		t.Errorf("Version = %q, want HTTP/3", result.Request.Version)
	}
}

func TestParseCurl_HTTP10(t *testing.T) {
	result := ParseCurl("curl --http1.0 https://example.com/")
	if result.Request == nil {
		t.Fatal("expected non-nil Request")
	}
	if result.Request.Version != "HTTP/1.0" {
		t.Errorf("Version = %q, want HTTP/1.0", result.Request.Version)
	}
}

func TestParseCurl_HTTP11Explicit(t *testing.T) {
	result := ParseCurl("curl --http1.1 https://example.com/")
	if result.Request == nil {
		t.Fatal("expected non-nil Request")
	}
	if result.Request.Version != "HTTP/1.1" {
		t.Errorf("Version = %q, want HTTP/1.1", result.Request.Version)
	}
}

// ── -I / --head ──────────────────────────────────────────────────────────

func TestParseCurl_HeadFlag(t *testing.T) {
	result := ParseCurl("curl -I https://example.com/")
	if result.Request == nil {
		t.Fatal("expected non-nil Request")
	}
	if result.Request.Method != "HEAD" {
		t.Errorf("Method = %q, want HEAD", result.Request.Method)
	}
}

func TestParseCurl_HeadFlagOverriddenByExplicitMethod(t *testing.T) {
	// When -X is provided before -I, the explicit method should win.
	result := ParseCurl("curl -X DELETE -I https://example.com/resource")
	if result.Request == nil {
		t.Fatal("expected non-nil Request")
	}
	// explicitMethod=true so -I must not override the method.
	if result.Request.Method != "DELETE" {
		t.Errorf("Method = %q, want DELETE (explicit method wins over -I)", result.Request.Method)
	}
}

// ── consume-and-discard flags (-o, --output, etc.) ───────────────────────

func TestParseCurl_OutputFlag(t *testing.T) {
	// -o consumes the next token; the URL is still correctly found.
	result := ParseCurl("curl -o /tmp/out.html https://example.com/")
	if result.Request == nil {
		t.Fatal("expected non-nil Request")
	}
	if result.Request.Path != "/" {
		t.Errorf("Path = %q, want /", result.Request.Path)
	}
}

func TestParseCurl_MaxTimeFlag(t *testing.T) {
	// -m / --max-time consume one argument.
	result := ParseCurl("curl -m 30 https://example.com/api")
	if result.Request == nil {
		t.Fatal("expected non-nil Request")
	}
	if result.Request.Path != "/api" {
		t.Errorf("Path = %q, want /api", result.Request.Path)
	}
}

// ── second positional argument ────────────────────────────────────────────

func TestParseCurl_SecondPositionalArg(t *testing.T) {
	// A second positional token generates a warning and is ignored.
	result := ParseCurl("curl https://example.com/ https://other.com/")
	if result.Request == nil {
		t.Fatal("expected non-nil Request")
	}
	// First URL is used.
	if result.Request.Path != "/" {
		t.Errorf("Path = %q, want /", result.Request.Path)
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "unexpected positional argument") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'unexpected positional argument' warning, got %v", result.Warnings)
	}
}

// ── URL userinfo (https://user:pass@host) ────────────────────────────────

func TestParseCurl_URLUserinfo(t *testing.T) {
	// Credentials embedded in the URL → Authorization: Basic header.
	// The -u flag's Authorization header must NOT be added when userinfo is present
	// and Authorization is absent.
	result := ParseCurl("curl https://alice:secret@api.example.com/v1/users")
	if result.Request == nil {
		t.Fatal("expected non-nil Request")
	}
	var auth, host string
	for _, h := range result.Request.Headers {
		if h.Key == "Authorization" {
			auth = h.Value
		}
		if h.Key == "Host" {
			host = h.Value
		}
	}
	if !strings.HasPrefix(auth, "Basic ") {
		t.Errorf("Authorization = %q, want Basic ...", auth)
	}
	if host != "api.example.com" {
		t.Errorf("Host = %q, want api.example.com", host)
	}
	// Path must not contain the userinfo or host.
	if result.Request.Path != "/v1/users" {
		t.Errorf("Path = %q, want /v1/users", result.Request.Path)
	}
}

// ── parseCurlURL: bare host with slash (no scheme) ────────────────────────

func TestParseCurlURL_BareHostWithSlash(t *testing.T) {
	// "example.com/api" → no scheme, slashIdx >= 0 → the bare-host-slash branch.
	scheme, userinfo, host, path := parseCurlURL("example.com/api")
	if scheme != "" {
		t.Errorf("scheme = %q, want empty", scheme)
	}
	if userinfo != "" {
		t.Errorf("userinfo = %q, want empty", userinfo)
	}
	if host != "example.com" {
		t.Errorf("host = %q, want example.com", host)
	}
	if path != "/api" {
		t.Errorf("path = %q, want /api", path)
	}
}

// ── shellSplit: trailing backslash with no following char ─────────────────

func TestShellSplit_TrailingBackslashNoNext(t *testing.T) {
	// A bare backslash at the very end of the string — i+1 >= len(s), so the
	// `if i+1 < len(s)` branch is NOT taken; the backslash is silently dropped.
	toks, err := shellSplit("curl https://example.com/\\")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(toks) < 2 {
		t.Fatalf("expected ≥2 tokens, got %v", toks)
	}
	// The URL token must end with "com/" (the trailing backslash is dropped).
	if !strings.HasSuffix(toks[1], "com/") {
		t.Errorf("URL token = %q, want suffix 'com/'", toks[1])
	}
}

// ── shellSplit: unclosed single quote ─────────────────────────────────────

func TestShellSplit_UnclosedSingleQuote(t *testing.T) {
	_, err := shellSplit("curl 'unclosed")
	if err == nil {
		t.Error("expected error for unclosed single quote")
	}
	if !strings.Contains(err.Error(), "unclosed single quote") {
		t.Errorf("error = %v, want 'unclosed single quote'", err)
	}
}

// ── shellSplit: bare backslash escape (i+1 < len(s) true) ─────────────────

func TestShellSplit_BareBackslashEscape(t *testing.T) {
	// Outside quotes, backslash followed by a non-final char escapes the next
	// char. "hello\ world" → "hello world" (backslash-space joins tokens).
	// This exercises the `cur.WriteByte(s[i+1]); i++` branch in shellSplit.
	toks, err := shellSplit("hello\\ world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(toks) != 1 {
		t.Fatalf("expected 1 token, got %v", toks)
	}
	if toks[0] != "hello world" {
		t.Errorf("token = %q, want 'hello world'", toks[0])
	}
}
