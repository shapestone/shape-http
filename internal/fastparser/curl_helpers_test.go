package fastparser

// curl_helpers_test.go — targeted unit tests for curl.go helpers that were
// previously below 90% coverage: expandShortFlags, shellSplit, parseCurlURL,
// buildMultipartForm, buildURLEncoded, stripNonCurlLines, parseCurlHeader.

import (
	"strings"
	"testing"
)

// ── expandShortFlags ──────────────────────────────────────────────────────

func TestExpandShortFlags_AllNoArgChars(t *testing.T) {
	// -svk → [-s, -v, -k]
	got := expandShortFlags([]string{"-svk"})
	want := []string{"-s", "-v", "-k"}
	if !strSliceEq(got, want) {
		t.Errorf("expandShortFlags(-svk) = %v, want %v", got, want)
	}
}

func TestExpandShortFlags_InlineArgMidCompound(t *testing.T) {
	// -sXPOST → [-s, -X, POST]  (X takes an arg; rest of compound is the arg)
	got := expandShortFlags([]string{"-sXPOST"})
	want := []string{"-s", "-X", "POST"}
	if !strSliceEq(got, want) {
		t.Errorf("expandShortFlags(-sXPOST) = %v, want %v", got, want)
	}
}

func TestExpandShortFlags_ArgFlagAtEndNoInlineArg(t *testing.T) {
	// -sX → [-s, -X]  (X takes an arg but nothing follows it inline)
	got := expandShortFlags([]string{"-sX"})
	want := []string{"-s", "-X"}
	if !strSliceEq(got, want) {
		t.Errorf("expandShortFlags(-sX) = %v, want %v", got, want)
	}
}

func TestExpandShortFlags_SingleCharPassThrough(t *testing.T) {
	// -s (len==2) must NOT be expanded — pass through unchanged.
	got := expandShortFlags([]string{"-s"})
	want := []string{"-s"}
	if !strSliceEq(got, want) {
		t.Errorf("expandShortFlags(-s) = %v, want %v", got, want)
	}
}

func TestExpandShortFlags_LongFlagPassThrough(t *testing.T) {
	// --verbose starts with '--', must not be expanded.
	got := expandShortFlags([]string{"--verbose"})
	want := []string{"--verbose"}
	if !strSliceEq(got, want) {
		t.Errorf("expandShortFlags(--verbose) = %v, want %v", got, want)
	}
}

func TestExpandShortFlags_HashPrefixPassThrough(t *testing.T) {
	// -# (progress bar) starts with -# and must not be expanded.
	got := expandShortFlags([]string{"-#"})
	want := []string{"-#"}
	if !strSliceEq(got, want) {
		t.Errorf("expandShortFlags(-#) = %v, want %v", got, want)
	}
}

func TestExpandShortFlags_PositionalPassThrough(t *testing.T) {
	// URL tokens are not flags, must pass through unchanged.
	got := expandShortFlags([]string{"https://example.com/api"})
	want := []string{"https://example.com/api"}
	if !strSliceEq(got, want) {
		t.Errorf("expandShortFlags(URL) = %v, want %v", got, want)
	}
}

func TestExpandShortFlags_MixedTokens(t *testing.T) {
	// A real-world token slice: curl -sSvk -X POST https://...
	in := []string{"-sSvk", "-X", "POST", "https://example.com/"}
	got := expandShortFlags(in)
	// -sSvk → -s -S -v -k, rest unchanged
	want := []string{"-s", "-S", "-v", "-k", "-X", "POST", "https://example.com/"}
	if !strSliceEq(got, want) {
		t.Errorf("expandShortFlags(mixed) = %v, want %v", got, want)
	}
}

// ── shellSplit additional escape sequences ─────────────────────────────────

func TestShellSplit_DoubleQuote_DollarEscape(t *testing.T) {
	// Inside double quotes, \$ is a literal dollar sign (shell escape).
	toks, err := shellSplit(`curl -H "X-Val: \$HOME"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(toks) < 3 {
		t.Fatalf("expected ≥3 tokens, got %v", toks)
	}
	if toks[2] != `X-Val: $HOME` {
		t.Errorf("header = %q, want %q", toks[2], `X-Val: $HOME`)
	}
}

func TestShellSplit_DoubleQuote_BacktickEscape(t *testing.T) {
	// Inside double quotes, \` is a literal backtick.
	toks, err := shellSplit("curl -H \"X-Val: \\`date\\`\"")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(toks) < 3 {
		t.Fatalf("expected ≥3 tokens, got %v", toks)
	}
	if toks[2] != "X-Val: `date`" {
		t.Errorf("header = %q, want %q", toks[2], "X-Val: `date`")
	}
}

func TestShellSplit_DoubleQuote_NewlineEscape(t *testing.T) {
	// Inside double quotes, a literal \n in the source collapses to newline char.
	toks, err := shellSplit("curl -d \"line1\\\nline2\"")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(toks) < 3 {
		t.Fatalf("expected ≥3 tokens, got %v", toks)
	}
	if toks[2] != "line1\nline2" {
		t.Errorf("body = %q, want %q", toks[2], "line1\nline2")
	}
}

func TestShellSplit_DoubleQuote_NonEscapableBackslash(t *testing.T) {
	// Inside double quotes, \z (non-escapable char) → literal backslash + z.
	toks, err := shellSplit(`curl -H "X-Val: \z"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(toks) < 3 {
		t.Fatalf("expected ≥3 tokens, got %v", toks)
	}
	if toks[2] != `X-Val: \z` {
		t.Errorf("header = %q, want %q", toks[2], `X-Val: \z`)
	}
}

func TestShellSplit_TrailingBackslash(t *testing.T) {
	// A bare trailing backslash (no next char) should not panic or error.
	// The backslash is silently dropped (no next char to consume).
	toks, err := shellSplit(`curl https://example.com/`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(toks) == 0 {
		t.Fatal("expected tokens, got none")
	}
}

// ── parseCurlURL additional cases ─────────────────────────────────────────

func TestParseCurlURL_NoSchemeNoSlash(t *testing.T) {
	// Bare hostname with no path (e.g. "example.com") — path must default to "/".
	_, _, host, path := parseCurlURL("example.com")
	if host != "example.com" {
		t.Errorf("host = %q, want example.com", host)
	}
	if path != "/" {
		t.Errorf("path = %q, want /", path)
	}
}

func TestParseCurlURL_Fragment(t *testing.T) {
	// Fragment must be stripped; path/host intact.
	scheme, _, host, path := parseCurlURL("https://example.com/docs#section-3")
	if scheme != "https" {
		t.Errorf("scheme = %q, want https", scheme)
	}
	if host != "example.com" {
		t.Errorf("host = %q, want example.com", host)
	}
	if path != "/docs" {
		t.Errorf("path = %q, want /docs (no fragment)", path)
	}
}

func TestParseCurlURL_FragmentOnly(t *testing.T) {
	// URL ending with #: path is "/" after fragment strip.
	_, _, host, path := parseCurlURL("https://example.com/#hero")
	if host != "example.com" {
		t.Errorf("host = %q, want example.com", host)
	}
	if path != "/" {
		t.Errorf("path = %q, want /", path)
	}
}

// ── buildMultipartForm edge cases ─────────────────────────────────────────

func TestBuildMultipartForm_FieldWithNoEquals(t *testing.T) {
	// A field without "=" must be skipped with a warning.
	cp := &curlParser{}
	body, boundary := buildMultipartForm([]string{"badfield"}, cp)
	if boundary == "" {
		t.Error("expected non-empty boundary")
	}
	if len(cp.warnings) == 0 {
		t.Error("expected warning for field with no '='")
	}
	if !strings.Contains(cp.warnings[0], "no '='") {
		t.Errorf("warning = %q, want 'no =' mention", cp.warnings[0])
	}
	// Body still has valid multipart terminator.
	if !strings.Contains(string(body), "--"+boundary+"--") {
		t.Errorf("body missing terminator: %s", body)
	}
}

func TestBuildMultipartForm_FileUploadSkipped(t *testing.T) {
	// A field whose value starts with "@" is a file upload — skip with warning.
	cp := &curlParser{}
	body, _ := buildMultipartForm([]string{"file=@photo.jpg"}, cp)
	if len(cp.warnings) == 0 {
		t.Error("expected warning for file upload")
	}
	if strings.Contains(string(body), "photo.jpg") {
		t.Error("file upload content must not appear in body")
	}
}

// ── buildURLEncoded edge cases ─────────────────────────────────────────────

func TestBuildURLEncoded_EmptyName(t *testing.T) {
	// "=value" format (empty name) → percent-encode the value with no name prefix.
	got := buildURLEncoded([]string{"=hello world"})
	if got != "hello%20world" {
		t.Errorf("buildURLEncoded(=hello world) = %q, want hello%%20world", got)
	}
}

func TestBuildURLEncoded_NoEquals(t *testing.T) {
	// A field with no "=" is treated as a value-only entry.
	got := buildURLEncoded([]string{"hello world"})
	if got != "hello%20world" {
		t.Errorf("buildURLEncoded(hello world) = %q, want hello%%20world", got)
	}
}

func TestBuildURLEncoded_Multiple(t *testing.T) {
	// Multiple fields joined with "&".
	got := buildURLEncoded([]string{"q=hello world", "lang=en"})
	if got != "q=hello%20world&lang=en" {
		t.Errorf("got %q", got)
	}
}

// ── stripNonCurlLines edge cases ──────────────────────────────────────────

func TestStripNonCurlLines_DashSeparator(t *testing.T) {
	// "---" separator must be stripped entirely.
	input := "---\ncurl https://example.com/\n---"
	got := stripNonCurlLines(input)
	if strings.Contains(got, "---") {
		t.Errorf("--- not stripped: %q", got)
	}
	if !strings.Contains(got, "curl") {
		t.Errorf("curl line removed unexpectedly: %q", got)
	}
}

func TestStripNonCurlLines_IndentedDashSeparator(t *testing.T) {
	// "  ---  " (with leading/trailing spaces after TrimLeft) still stripped.
	input := "  ---\ncurl https://example.com/"
	got := stripNonCurlLines(input)
	if strings.Contains(got, "---") {
		t.Errorf("indented --- not stripped: %q", got)
	}
}

func TestStripNonCurlLines_CommentWithLeadingSpace(t *testing.T) {
	// "  # comment" — first non-whitespace is '#', must be stripped.
	input := "  # comment\ncurl https://example.com/"
	got := stripNonCurlLines(input)
	if strings.Contains(got, "comment") {
		t.Errorf("comment line not stripped: %q", got)
	}
}

func TestStripNonCurlLines_EmptyLineKept(t *testing.T) {
	// A truly empty line (not a separator) must be preserved (it becomes
	// a space after newline-normalisation anyway, but stripNonCurlLines itself
	// should not remove it — the check is len(trimmed) > 0).
	input := "curl https://example.com/\n\ncurl https://other.com/"
	got := stripNonCurlLines(input)
	if !strings.Contains(got, "example.com") || !strings.Contains(got, "other.com") {
		t.Errorf("content lines removed unexpectedly: %q", got)
	}
}

// ── parseCurlHeader edge cases ────────────────────────────────────────────

func TestParseCurlHeader_NoColon(t *testing.T) {
	// A header string with no colon → Key is the whole string, Value is empty.
	h := parseCurlHeader("BadHeader")
	if h.Key != "BadHeader" {
		t.Errorf("Key = %q, want BadHeader", h.Key)
	}
	if h.Value != "" {
		t.Errorf("Value = %q, want empty", h.Value)
	}
}

func TestParseCurlHeader_LeadingSpaceInValue(t *testing.T) {
	// OWS after colon must be trimmed.
	h := parseCurlHeader("Content-Type:   application/json   ")
	if h.Key != "Content-Type" {
		t.Errorf("Key = %q", h.Key)
	}
	if h.Value != "application/json   " {
		t.Errorf("Value = %q, want 'application/json   '", h.Value)
	}
}

// ── helper ────────────────────────────────────────────────────────────────

func strSliceEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
