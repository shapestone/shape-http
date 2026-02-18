package http

import (
	"strings"
	"testing"
)

// Canonical baselines — both are well-formed and parse with zero warnings.
const (
	baseGoodRequest  = "GET /api/users HTTP/1.1\r\nHost: example.com\r\nContent-Length: 4\r\n\r\ntest"
	baseGoodResponse = "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 5\r\n\r\nHello"
)

// ── Mutation operators ─────────────────────────────────────────────────────
//
// Each operator takes a well-formed HTTP message string and returns a mutated
// version. Operators are pure functions. An operator that targets content
// absent in the input (e.g. a request-only mutation on a response) is a no-op,
// which is intentional — the fuzz layer relies on this.

// mutRemoveVersion removes the HTTP version token from a request line.
//
//	"GET /api/users HTTP/1.1\r\n..." → "GET /api/users\r\n..."
func mutRemoveVersion(s string) string {
	return strings.Replace(s, " HTTP/1.1\r\n", "\r\n", 1)
}

// mutRemovePath removes the path token from a request line, leaving the
// version as the second token — lenient parser treats it as a missing version.
//
//	"GET /api/users HTTP/1.1\r\n..." → "GET HTTP/1.1\r\n..."
func mutRemovePath(s string) string {
	return strings.Replace(s, " /api/users", "", 1)
}

// mutInvalidStatusCode replaces the numeric status code with a non-numeric token.
//
//	"HTTP/1.1 200 OK\r\n..." → "HTTP/1.1 abc OK\r\n..."
func mutInvalidStatusCode(s string) string {
	return strings.Replace(s, " 200 ", " abc ", 1)
}

// mutRemoveStatusCode removes the status code token so that the reason phrase
// shifts into the code position — Atoi on "OK" fails → invalid status code.
//
//	"HTTP/1.1 200 OK\r\n..." → "HTTP/1.1 OK\r\n..."
func mutRemoveStatusCode(s string) string {
	return strings.Replace(s, " 200", "", 1)
}

// mutTruncateBody cuts the body to half its length, leaving Content-Length
// unchanged — this creates a deliberate mismatch that the lenient parser
// must detect and flag.
func mutTruncateBody(s string) string {
	idx := strings.Index(s, "\r\n\r\n")
	if idx == -1 {
		if len(s) > 1 {
			return s[:len(s)/2]
		}
		return s
	}
	body := s[idx+4:]
	if len(body) == 0 {
		return s
	}
	return s[:idx+4] + body[:len(body)/2]
}

// mutCorruptFirstHeader removes the colon from the first header line, turning
// a valid "Key: Value" into an invalid "Key Value" that has no separator.
func mutCorruptFirstHeader(s string) string {
	first := strings.Index(s, "\r\n")
	if first == -1 {
		return s
	}
	rest := s[first+2:]
	end := strings.Index(rest, "\r\n")
	if end == -1 {
		return s
	}
	corrupted := strings.Replace(rest[:end], ":", "", 1)
	return s[:first+2] + corrupted + rest[end:]
}

// mutAddSpaceBeforeColon inserts a space before the colon in the first header,
// producing "Host : example.com" — invalid per RFC 9112 but accepted leniently.
func mutAddSpaceBeforeColon(s string) string {
	first := strings.Index(s, "\r\n")
	if first == -1 {
		return s
	}
	rest := s[first+2:]
	end := strings.Index(rest, "\r\n")
	if end == -1 {
		return s
	}
	header := rest[:end]
	colon := strings.Index(header, ":")
	if colon == -1 {
		return s
	}
	return s[:first+2] + header[:colon] + " " + header[colon:] + rest[end:]
}

// mutBareLineFeed replaces all CRLF line endings with bare LF (\n only).
func mutBareLineFeed(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

// mutAddGarbageHeader inserts a line without a colon just before the blank-line
// separator, so it lands between valid headers and the body.
func mutAddGarbageHeader(s string) string {
	idx := strings.Index(s, "\r\n\r\n")
	if idx == -1 {
		return s + "\r\nX-Garbage No Colon Here"
	}
	return s[:idx] + "\r\nX-Garbage No Colon Here" + s[idx:]
}

// mutRemoveBlankLine removes the blank-line separator between headers and body,
// causing the body to be parsed as a (malformed) header line.
func mutRemoveBlankLine(s string) string {
	return strings.Replace(s, "\r\n\r\n", "\r\n", 1)
}

// ── Shared bitmask dispatch (used by the fuzz target) ─────────────────────

var mutationOps = []func(string) string{
	mutRemoveVersion,       // bit 0
	mutRemovePath,          // bit 1
	mutTruncateBody,        // bit 2
	mutCorruptFirstHeader,  // bit 3
	mutAddSpaceBeforeColon, // bit 4
	mutBareLineFeed,        // bit 5
	mutAddGarbageHeader,    // bit 6
	mutRemoveBlankLine,     // bit 7
}

// applyMutations applies the operators selected by the bitmask in order.
// Operators that don't match any content in s are no-ops.
func applyMutations(s string, mask uint8) string {
	for i, op := range mutationOps {
		if mask&(1<<uint(i)) != 0 {
			s = op(s)
		}
	}
	return s
}

// ── Test helper ───────────────────────────────────────────────────────────

func hasWarningSubstr(result *ParseResult, substr string) bool {
	for _, w := range result.Warnings {
		if strings.Contains(w, substr) {
			return true
		}
	}
	return false
}

// ── Table-driven mutation tests: request ──────────────────────────────────

func TestLenientMutations_Request(t *testing.T) {
	type testCase struct {
		name        string
		apply       func(string) string
		wantWarning string // substring expected in at least one warning; "" = skip check
		wantPartial bool
		check       func(*testing.T, *ParseResult)
	}

	tests := []testCase{
		// ── Missing parts ─────────────────────────────────────────────────
		{
			name:        "remove version",
			apply:       mutRemoveVersion,
			wantWarning: "missing HTTP version",
			check: func(t *testing.T, r *ParseResult) {
				if r.Request == nil {
					t.Fatal("expected request")
				}
				if r.Request.Version != "HTTP/1.1" {
					t.Errorf("Version = %q, want HTTP/1.1 (lenient default)", r.Request.Version)
				}
				if r.Request.Method != "GET" {
					t.Errorf("Method = %q, want GET", r.Request.Method)
				}
			},
		},
		{
			// Removing the path shifts HTTP/1.1 into the path position, so only
			// two tokens remain — lenient treats the second as path and warns
			// about the missing version.
			name:        "remove path",
			apply:       mutRemovePath,
			wantWarning: "missing HTTP version",
			check: func(t *testing.T, r *ParseResult) {
				if r.Request == nil {
					t.Fatal("expected request")
				}
				if r.Request.Method != "GET" {
					t.Errorf("Method = %q, want GET", r.Request.Method)
				}
			},
		},
		{
			name:        "truncate body",
			apply:       mutTruncateBody,
			wantWarning: "truncated",
			wantPartial: true,
			check: func(t *testing.T, r *ParseResult) {
				if r.Request == nil {
					t.Fatal("expected request")
				}
				if len(r.Request.Body) == 0 {
					t.Error("expected non-empty partial body")
				}
			},
		},
		{
			// Without the blank-line separator the body is parsed as a header;
			// "test" has no colon → malformed header warning.
			name:        "remove blank line separator",
			apply:       mutRemoveBlankLine,
			wantWarning: "malformed header",
			check: func(t *testing.T, r *ParseResult) {
				if r.Request == nil {
					t.Fatal("expected request")
				}
			},
		},

		// ── Modified parts ────────────────────────────────────────────────
		{
			name:        "corrupt first header (remove colon)",
			apply:       mutCorruptFirstHeader,
			wantWarning: "malformed header",
			check: func(t *testing.T, r *ParseResult) {
				if r.Request == nil {
					t.Fatal("expected request")
				}
				// Host was corrupted; the second header (Content-Length) must survive.
				if r.Request.Headers.Get("Content-Length") == "" {
					t.Error("expected Content-Length header to survive the corruption")
				}
			},
		},
		{
			name:        "whitespace before colon in first header",
			apply:       mutAddSpaceBeforeColon,
			wantWarning: "whitespace before colon",
			check: func(t *testing.T, r *ParseResult) {
				if r.Request == nil {
					t.Fatal("expected request")
				}
				// Lenient parser must accept and trim — Host must still be parsed.
				if r.Request.Headers.Get("Host") != "example.com" {
					t.Errorf("Host = %q, want example.com (lenient should accept)", r.Request.Headers.Get("Host"))
				}
			},
		},
		{
			// Bare LF is accepted without any warning — it is the most common
			// real-world deviation from strict RFC 9112 line endings.
			name:  "bare LF line endings",
			apply: mutBareLineFeed,
			check: func(t *testing.T, r *ParseResult) {
				if r.Request == nil {
					t.Fatal("expected request")
				}
				if r.Request.Method != "GET" {
					t.Errorf("Method = %q, want GET", r.Request.Method)
				}
				if r.Request.Headers.Get("Host") != "example.com" {
					t.Errorf("Host = %q, want example.com", r.Request.Headers.Get("Host"))
				}
			},
		},

		// ── Added invalid parts ───────────────────────────────────────────
		{
			name:        "garbage header (no colon)",
			apply:       mutAddGarbageHeader,
			wantWarning: "malformed header",
			check: func(t *testing.T, r *ParseResult) {
				if r.Request == nil {
					t.Fatal("expected request")
				}
				// Valid headers around the garbage line must survive.
				if r.Request.Headers.Get("Host") != "example.com" {
					t.Errorf("Host = %q, want example.com", r.Request.Headers.Get("Host"))
				}
			},
		},

		// ── Combined mutations ─────────────────────────────────────────────
		{
			name: "remove version + garbage header",
			apply: func(s string) string {
				return mutAddGarbageHeader(mutRemoveVersion(s))
			},
			wantWarning: "missing HTTP version",
			check: func(t *testing.T, r *ParseResult) {
				if r.Request == nil {
					t.Fatal("expected request")
				}
				if r.Request.Version != "HTTP/1.1" {
					t.Errorf("Version = %q, want HTTP/1.1 (default)", r.Request.Version)
				}
				if !hasWarningSubstr(r, "malformed header") {
					t.Errorf("expected malformed header warning too, got %v", r.Warnings)
				}
			},
		},
		{
			name: "corrupt header + truncate body",
			apply: func(s string) string {
				return mutTruncateBody(mutCorruptFirstHeader(s))
			},
			wantWarning: "malformed header",
			wantPartial: true,
			check: func(t *testing.T, r *ParseResult) {
				if r.Request == nil {
					t.Fatal("expected request")
				}
			},
		},
		{
			// Apply bare LF last so the blank-line separator is still findable
			// when mutAddGarbageHeader and mutTruncateBody run, preserving the
			// Content-Length mismatch that drives Partial=true.
			name: "bare LF + garbage header + truncate body",
			apply: func(s string) string {
				return mutBareLineFeed(mutTruncateBody(mutAddGarbageHeader(s)))
			},
			wantWarning: "malformed header",
			wantPartial: true,
			check: func(t *testing.T, r *ParseResult) {
				if r.Request == nil {
					t.Fatal("expected request")
				}
			},
		},
		{
			// All eight operators applied at once — the lenient parser must
			// return something coherent rather than panic or return nil.
			name: "all mutations combined",
			apply: func(s string) string {
				return applyMutations(s, 0xFF)
			},
			check: func(t *testing.T, r *ParseResult) {
				if r.Request == nil {
					t.Fatal("expected request even under all mutations")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.apply(baseGoodRequest)
			result := UnmarshalLenient([]byte(input))

			if result == nil {
				t.Fatal("UnmarshalLenient returned nil")
			}
			if tt.wantPartial && !result.Partial {
				t.Errorf("Partial = false, want true; warnings: %v", result.Warnings)
			}
			if tt.wantWarning != "" && !hasWarningSubstr(result, tt.wantWarning) {
				t.Errorf("expected warning containing %q, got %v", tt.wantWarning, result.Warnings)
			}
			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

// ── Table-driven mutation tests: response ─────────────────────────────────

func TestLenientMutations_Response(t *testing.T) {
	type testCase struct {
		name        string
		apply       func(string) string
		wantWarning string
		wantPartial bool
		check       func(*testing.T, *ParseResult)
	}

	tests := []testCase{
		// ── Missing parts ─────────────────────────────────────────────────
		{
			name:        "truncate body",
			apply:       mutTruncateBody,
			wantWarning: "truncated",
			wantPartial: true,
			check: func(t *testing.T, r *ParseResult) {
				if r.Response == nil {
					t.Fatal("expected response")
				}
				if r.Response.StatusCode != 200 {
					t.Errorf("StatusCode = %d, want 200", r.Response.StatusCode)
				}
			},
		},
		{
			name:        "remove blank line separator",
			apply:       mutRemoveBlankLine,
			wantWarning: "malformed header",
			check: func(t *testing.T, r *ParseResult) {
				if r.Response == nil {
					t.Fatal("expected response")
				}
			},
		},

		// ── Modified parts ────────────────────────────────────────────────
		{
			name:        "invalid status code (non-numeric token)",
			apply:       mutInvalidStatusCode,
			wantWarning: "invalid status code",
			check: func(t *testing.T, r *ParseResult) {
				if r.Response == nil {
					t.Fatal("expected response")
				}
				if r.Response.StatusCode != 0 {
					t.Errorf("StatusCode = %d, want 0 (invalid code defaults to 0)", r.Response.StatusCode)
				}
				// Reason phrase must be preserved despite the bad code.
				if r.Response.Reason != "OK" {
					t.Errorf("Reason = %q, want OK", r.Response.Reason)
				}
			},
		},
		{
			// Removing the code shifts "OK" into the code position;
			// Atoi("OK") fails → invalid status code warning.
			name:        "remove status code",
			apply:       mutRemoveStatusCode,
			wantWarning: "invalid status code",
			check: func(t *testing.T, r *ParseResult) {
				if r.Response == nil {
					t.Fatal("expected response")
				}
				if r.Response.StatusCode != 0 {
					t.Errorf("StatusCode = %d, want 0", r.Response.StatusCode)
				}
			},
		},
		{
			name:        "corrupt first header (remove colon)",
			apply:       mutCorruptFirstHeader,
			wantWarning: "malformed header",
			check: func(t *testing.T, r *ParseResult) {
				if r.Response == nil {
					t.Fatal("expected response")
				}
				// Content-Length (second header) must survive.
				if r.Response.Headers.Get("Content-Length") == "" {
					t.Error("expected Content-Length to survive the corruption")
				}
			},
		},
		{
			name:        "whitespace before colon in first header",
			apply:       mutAddSpaceBeforeColon,
			wantWarning: "whitespace before colon",
			check: func(t *testing.T, r *ParseResult) {
				if r.Response == nil {
					t.Fatal("expected response")
				}
				if r.Response.Headers.Get("Content-Type") != "text/plain" {
					t.Errorf("Content-Type = %q, want text/plain (lenient should accept)", r.Response.Headers.Get("Content-Type"))
				}
			},
		},
		{
			name:  "bare LF line endings",
			apply: mutBareLineFeed,
			check: func(t *testing.T, r *ParseResult) {
				if r.Response == nil {
					t.Fatal("expected response")
				}
				if r.Response.StatusCode != 200 {
					t.Errorf("StatusCode = %d, want 200", r.Response.StatusCode)
				}
			},
		},

		// ── Added invalid parts ───────────────────────────────────────────
		{
			name:        "garbage header (no colon)",
			apply:       mutAddGarbageHeader,
			wantWarning: "malformed header",
			check: func(t *testing.T, r *ParseResult) {
				if r.Response == nil {
					t.Fatal("expected response")
				}
				if r.Response.Headers.Get("Content-Type") != "text/plain" {
					t.Errorf("Content-Type = %q, want text/plain", r.Response.Headers.Get("Content-Type"))
				}
			},
		},

		// ── Combined mutations ─────────────────────────────────────────────
		{
			name: "invalid status code + garbage header",
			apply: func(s string) string {
				return mutAddGarbageHeader(mutInvalidStatusCode(s))
			},
			wantWarning: "invalid status code",
			check: func(t *testing.T, r *ParseResult) {
				if r.Response == nil {
					t.Fatal("expected response")
				}
				if r.Response.StatusCode != 0 {
					t.Errorf("StatusCode = %d, want 0", r.Response.StatusCode)
				}
				if !hasWarningSubstr(r, "malformed header") {
					t.Errorf("expected malformed header warning too, got %v", r.Warnings)
				}
			},
		},
		{
			name: "corrupt header + truncate body",
			apply: func(s string) string {
				return mutTruncateBody(mutCorruptFirstHeader(s))
			},
			wantWarning: "malformed header",
			wantPartial: true,
			check: func(t *testing.T, r *ParseResult) {
				if r.Response == nil {
					t.Fatal("expected response")
				}
			},
		},
		{
			name: "all mutations combined",
			apply: func(s string) string {
				return applyMutations(s, 0xFF)
			},
			check: func(t *testing.T, r *ParseResult) {
				if r.Response == nil {
					t.Fatal("expected response even under all mutations")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.apply(baseGoodResponse)
			result := UnmarshalLenient([]byte(input))

			if result == nil {
				t.Fatal("UnmarshalLenient returned nil")
			}
			if tt.wantPartial && !result.Partial {
				t.Errorf("Partial = false, want true; warnings: %v", result.Warnings)
			}
			if tt.wantWarning != "" && !hasWarningSubstr(result, tt.wantWarning) {
				t.Errorf("expected warning containing %q, got %v", tt.wantWarning, result.Warnings)
			}
			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

// ── Fuzz: randomized mutation combinations ────────────────────────────────
//
// FuzzLenientMutations applies random combinations of mutation operators to
// canonical baselines (and arbitrary fuzz-generated strings) and verifies
// structural invariants. It does not check for specific warning text — only
// that the parser always returns a consistent, non-nil result.
//
// The mutation operator set is the same as the deterministic table tests;
// the fuzz engine explores all 2^8 = 256 operator combinations as well as
// completely arbitrary base strings, discovering interactions that humans
// might not think to test.
func FuzzLenientMutations(f *testing.F) {
	// Seed: baselines with no mutations applied.
	f.Add(baseGoodRequest, uint8(0))
	f.Add(baseGoodResponse, uint8(0))

	// Seed: each single mutation on each baseline.
	for i := 0; i < 8; i++ {
		mask := uint8(1 << uint(i))
		f.Add(baseGoodRequest, mask)
		f.Add(baseGoodResponse, mask)
	}

	// Seed: all mutations combined.
	f.Add(baseGoodRequest, uint8(0xFF))
	f.Add(baseGoodResponse, uint8(0xFF))

	// Seed: a few interesting hand-picked combinations.
	f.Add(baseGoodRequest, uint8(0b00000101))  // remove version + truncate body
	f.Add(baseGoodRequest, uint8(0b10001000))  // corrupt header + remove blank line
	f.Add(baseGoodResponse, uint8(0b01000100)) // truncate body + garbage header
	f.Add(baseGoodResponse, uint8(0b00010010)) // remove status code + corrupt header

	f.Fuzz(func(t *testing.T, base string, mutationMask uint8) {
		input := applyMutations(base, mutationMask)
		result := UnmarshalLenient([]byte(input))

		// Invariant 1: never returns nil — the lenient API must always
		// produce a result regardless of how broken the input is.
		if result == nil {
			t.Fatal("UnmarshalLenient returned nil")
			return // unreachable; silences static analysis nil-deref warnings below
		}

		// Invariant 2: Request and Response are never both non-nil —
		// a message is either a request or a response, never both.
		if result.Request != nil && result.Response != nil {
			t.Fatal("both Request and Response are non-nil")
		}

		// Invariant 3: if nothing was detected, Partial must be true —
		// an empty result without Partial would be misleading.
		if result.Request == nil && result.Response == nil && !result.Partial {
			t.Fatal("empty result (no request, no response) with Partial=false")
		}

		// Invariant 4: Request.Version is always set — the lenient parser
		// defaults to "HTTP/1.1" for every missing/empty version case.
		if result.Request != nil && result.Request.Version == "" {
			t.Error("Request.Version is empty — lenient parser should always set a default")
		}

		// Invariant 5: Response.Version is always set — same guarantee.
		if result.Response != nil && result.Response.Version == "" {
			t.Error("Response.Version is empty — lenient parser should always set a version")
		}
	})
}
