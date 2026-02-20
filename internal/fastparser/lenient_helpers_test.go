package fastparser

// lenient_helpers_test.go — targeted unit tests for the small helper functions
// in lenient.go that were below 90% coverage:
// looksLikeHeaderField, parseIPv6HostLine, isHostnameKeyStr,
// isSingleLabelHost, isPortStr, isHostnameLike.

import "testing"

// ── looksLikeHeaderField ──────────────────────────────────────────────────

func TestLooksLikeHeaderField(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		// True cases
		{"Content-Type: application/json", true},
		{"X-API-Key: abc", true},
		{"Authorization: Bearer tok", true},
		{"Accept:", true}, // colon immediately after name
		// First char is digit → false
		{"1Bad: val", false},
		// First char is hyphen → false
		{"-Bad: val", false},
		// Non-token char before colon → false
		{"Bad Header: val", false}, // space before colon
		{"Bad/Header: val", false},
		// No colon at all → false
		{"NoColonHere", false},
		// Empty → false
		{"", false},
	}
	for _, tc := range cases {
		got := looksLikeHeaderField([]byte(tc.in))
		if got != tc.want {
			t.Errorf("looksLikeHeaderField(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// ── parseIPv6HostLine ─────────────────────────────────────────────────────

func TestParseIPv6HostLine(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		// Valid IPv6 without port
		{"[::1]", "[::1]"},
		{"[2001:db8::1]", "[2001:db8::1]"},
		// Valid IPv6 with port
		{"[::1]:8080", "[::1]:8080"},
		// Brackets but no colon inside → not IPv6
		{"[word]", ""},
		// No opening bracket → not IPv6
		{"::1", ""},
		// Too short
		{"[:", ""},
		// No closing bracket
		{"[::1", ""},
		// Unexpected suffix after port (non-digit)
		{"[::1]:abc", ""},
		// Port suffix with no digits after colon
		{"[::1]:", ""},
	}
	for _, tc := range cases {
		got := parseIPv6HostLine([]byte(tc.in))
		if got != tc.want {
			t.Errorf("parseIPv6HostLine(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// ── isHostnameKeyStr ──────────────────────────────────────────────────────

func TestIsHostnameKeyStr(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"example.com", true},
		{"api.example.com", true},
		{"my-host.example.org", true},
		// No dot → false (discriminator requires dot)
		{"localhost", false},
		{"noDotsHere", false},
		// Empty → false
		{"", false},
		// Invalid char → false
		{"example.com/path", false},
		{"exam ple.com", false},
	}
	for _, tc := range cases {
		got := isHostnameKeyStr(tc.in)
		if got != tc.want {
			t.Errorf("isHostnameKeyStr(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// ── isSingleLabelHost ─────────────────────────────────────────────────────

func TestIsSingleLabelHost(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"localhost", true},
		{"db", true},
		{"host123", true},
		// Uppercase → false (only lowercase + digits accepted)
		{"Localhost", false},
		{"MyServer", false},
		// Hyphen → false
		{"my-server", false},
		// Dot → false
		{"my.server", false},
		// Empty → false
		{"", false},
	}
	for _, tc := range cases {
		got := isSingleLabelHost(tc.in)
		if got != tc.want {
			t.Errorf("isSingleLabelHost(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// ── isPortStr ─────────────────────────────────────────────────────────────

func TestIsPortStr(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"8080", true},
		{"443", true},
		{"0", true},
		// Empty → false
		{"", false},
		// Non-digit chars → false
		{"80x", false},
		{"abc", false},
		{"8 080", false},
	}
	for _, tc := range cases {
		got := isPortStr(tc.in)
		if got != tc.want {
			t.Errorf("isPortStr(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// ── isHostnameLike ────────────────────────────────────────────────────────

func TestIsHostnameLike(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		// With dot
		{"example.com", true},
		{"api.example.com", true},
		{"192.168.1.1", true},
		// With port (no dot required when port present)
		{"localhost:8080", true},
		{"example.com:8443", true},
		// Invalid first char
		{"_bad.com", false},
		{"-bad.com", false},
		// Empty
		{"", false},
		// Bare word without dot and without port → false
		{"localhost", false},
		// Invalid char in host part
		{"exam ple.com", false},
	}
	for _, tc := range cases {
		got := isHostnameLike([]byte(tc.in))
		if got != tc.want {
			t.Errorf("isHostnameLike(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
