package fastparser

import (
	"strings"
	"testing"
)

// helper: find a header value by key (case-insensitive)
func findHeader(headers []Header, key string) string {
	for _, h := range headers {
		if eqFold(h.Key, key) {
			return h.Value
		}
	}
	return ""
}

func TestParseCurl_Empty(t *testing.T) {
	result := ParseCurl("")
	if result.Request != nil {
		t.Error("expected nil Request for empty input")
	}
	if !result.Partial {
		t.Error("expected Partial=true for empty input")
	}
	if len(result.Warnings) == 0 {
		t.Error("expected at least one warning for empty input")
	}
}

func TestParseCurl_NoURL(t *testing.T) {
	result := ParseCurl("curl -X GET")
	if result.Request != nil {
		t.Error("expected nil Request when URL is missing")
	}
	if !result.Partial {
		t.Error("expected Partial=true")
	}
	hasURLWarn := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "URL") || strings.Contains(w, "url") {
			hasURLWarn = true
		}
	}
	if !hasURLWarn {
		t.Errorf("expected URL-missing warning, got %v", result.Warnings)
	}
}

func TestParseCurl_SimpleGET(t *testing.T) {
	result := ParseCurl(`curl https://example.com/api/users`)
	if result.Request == nil {
		t.Fatalf("expected request, got nil; warnings: %v", result.Warnings)
	}
	req := result.Request
	if req.Method != "GET" {
		t.Errorf("Method = %q, want GET", req.Method)
	}
	if req.Path != "/api/users" {
		t.Errorf("Path = %q, want /api/users", req.Path)
	}
	if req.Scheme != "https" {
		t.Errorf("Scheme = %q, want https", req.Scheme)
	}
	if req.Version != "HTTP/1.1" {
		t.Errorf("Version = %q, want HTTP/1.1", req.Version)
	}
	if findHeader(req.Headers, "Host") != "example.com" {
		t.Errorf("Host = %q, want example.com", findHeader(req.Headers, "Host"))
	}
}

func TestParseCurl_GETWithQueryParams(t *testing.T) {
	result := ParseCurl(`curl "https://example.com/api/users?page=1&limit=10"`)
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	req := result.Request
	if req.Method != "GET" {
		t.Errorf("Method = %q, want GET", req.Method)
	}
	if req.Path != "/api/users?page=1&limit=10" {
		t.Errorf("Path = %q, want /api/users?page=1&limit=10", req.Path)
	}
	if findHeader(req.Headers, "Host") != "example.com" {
		t.Errorf("Host = %q, want example.com", findHeader(req.Headers, "Host"))
	}
}

func TestParseCurl_POSTwithJSONBody(t *testing.T) {
	cmd := `curl -X POST https://example.com/api/users ` +
		`-H "Content-Type: application/json" ` +
		`-d '{"name": "John Doe", "email": "john@example.com"}'`
	result := ParseCurl(cmd)
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	req := result.Request
	if req.Method != "POST" {
		t.Errorf("Method = %q, want POST", req.Method)
	}
	if req.Path != "/api/users" {
		t.Errorf("Path = %q, want /api/users", req.Path)
	}
	if findHeader(req.Headers, "Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q", findHeader(req.Headers, "Content-Type"))
	}
	if len(req.Body) == 0 {
		t.Error("expected non-empty body")
	}
	if !strings.Contains(string(req.Body), "John Doe") {
		t.Errorf("body does not contain expected data: %s", string(req.Body))
	}
	// Content-Length should be auto-set
	if findHeader(req.Headers, "Content-Length") == "" {
		t.Error("expected auto Content-Length header")
	}
}

func TestParseCurl_POSTDefaultMethod(t *testing.T) {
	// No -X flag but body present → method should default to POST
	result := ParseCurl(`curl https://example.com/api -d '{"x":1}'`)
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	if result.Request.Method != "POST" {
		t.Errorf("Method = %q, want POST when body present and no -X", result.Request.Method)
	}
}

func TestParseCurl_PUTwithAuth(t *testing.T) {
	cmd := "curl -X PUT https://example.com/api/users/123 \\\n" +
		`     -H "Authorization: Bearer abc123" ` + "\\\n" +
		`     -H "Content-Type: application/json" ` + "\\\n" +
		`     -d '{"name": "Jane Doe"}'`
	result := ParseCurl(cmd)
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	req := result.Request
	if req.Method != "PUT" {
		t.Errorf("Method = %q, want PUT", req.Method)
	}
	if req.Path != "/api/users/123" {
		t.Errorf("Path = %q, want /api/users/123", req.Path)
	}
	if findHeader(req.Headers, "Authorization") != "Bearer abc123" {
		t.Errorf("Authorization = %q", findHeader(req.Headers, "Authorization"))
	}
}

func TestParseCurl_BasicAuth(t *testing.T) {
	result := ParseCurl(`curl -u admin:secret https://example.com/api/protected`)
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	req := result.Request
	// "admin:secret" base64 → "YWRtaW46c2VjcmV0"
	want := "Basic YWRtaW46c2VjcmV0"
	if findHeader(req.Headers, "Authorization") != want {
		t.Errorf("Authorization = %q, want %q", findHeader(req.Headers, "Authorization"), want)
	}
	if req.Method != "GET" {
		t.Errorf("Method = %q, want GET", req.Method)
	}
}

func TestParseCurl_DELETE(t *testing.T) {
	result := ParseCurl(`curl -X DELETE https://example.com/api/users/123`)
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	if result.Request.Method != "DELETE" {
		t.Errorf("Method = %q, want DELETE", result.Request.Method)
	}
}

func TestParseCurl_MultilineBackslash(t *testing.T) {
	cmd := "curl -X POST \\\n" +
		"  https://example.com/api/users \\\n" +
		`  -H "Content-Type: application/json"` + " \\\n" +
		`  -d '{"name":"Alice"}'`
	result := ParseCurl(cmd)
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	req := result.Request
	if req.Method != "POST" {
		t.Errorf("Method = %q, want POST", req.Method)
	}
	if req.Path != "/api/users" {
		t.Errorf("Path = %q, want /api/users", req.Path)
	}
	if !strings.Contains(string(req.Body), "Alice") {
		t.Errorf("body = %q, want Alice", string(req.Body))
	}
}

func TestParseCurl_HTTP2(t *testing.T) {
	result := ParseCurl(`curl --http2 https://example.com/api`)
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	if result.Request.Version != "HTTP/2" {
		t.Errorf("Version = %q, want HTTP/2", result.Request.Version)
	}
}

func TestParseCurl_IgnoredFlags(t *testing.T) {
	// -v, -s, -k, -L should be silently ignored; result should be valid
	result := ParseCurl(`curl -v -s -k -L https://example.com/api`)
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	// No warnings expected for known-ignored flags
	for _, w := range result.Warnings {
		if strings.Contains(w, "unknown") {
			t.Errorf("unexpected 'unknown' warning for a known ignored flag: %s", w)
		}
	}
	if result.Request.Path != "/api" {
		t.Errorf("Path = %q, want /api", result.Request.Path)
	}
}

func TestParseCurl_UnknownFlag(t *testing.T) {
	result := ParseCurl(`curl --some-unknown-flag https://example.com/`)
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	hasWarn := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "unknown") || strings.Contains(w, "some-unknown-flag") {
			hasWarn = true
		}
	}
	if !hasWarn {
		t.Errorf("expected warning for unknown flag, got %v", result.Warnings)
	}
}

func TestParseCurl_FormData(t *testing.T) {
	result := ParseCurl(`curl -F "name=John" -F "email=john@example.com" https://example.com/upload`)
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	req := result.Request
	if req.Method != "POST" {
		t.Errorf("Method = %q, want POST", req.Method)
	}
	ct := findHeader(req.Headers, "Content-Type")
	if !strings.HasPrefix(ct, "multipart/form-data; boundary=") {
		t.Errorf("Content-Type = %q, want multipart/form-data", ct)
	}
	if !strings.Contains(string(req.Body), "John") {
		t.Errorf("body does not contain form value: %s", string(req.Body))
	}
	if !strings.Contains(string(req.Body), "john@example.com") {
		t.Errorf("body does not contain email: %s", string(req.Body))
	}
}

func TestParseCurl_DataUrlencode(t *testing.T) {
	result := ParseCurl(`curl --data-urlencode "name=John Doe" --data-urlencode "city=New York" https://example.com/search`)
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	req := result.Request
	if findHeader(req.Headers, "Content-Type") != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q", findHeader(req.Headers, "Content-Type"))
	}
	body := string(req.Body)
	// Spaces should be percent-encoded as %20
	if !strings.Contains(body, "John%20Doe") {
		t.Errorf("body %q does not contain encoded name", body)
	}
	if !strings.Contains(body, "New%20York") {
		t.Errorf("body %q does not contain encoded city", body)
	}
}

func TestParseCurl_AutoContentLength(t *testing.T) {
	result := ParseCurl(`curl -X POST https://example.com/api -d 'hello'`)
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	cl := findHeader(result.Request.Headers, "Content-Length")
	if cl != "5" { // len("hello") == 5
		t.Errorf("Content-Length = %q, want 5", cl)
	}
}

func TestParseCurl_NoAutoContentLengthWhenExplicit(t *testing.T) {
	result := ParseCurl(`curl -X POST https://example.com/api -H "Content-Length: 999" -d 'hello'`)
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	// Explicit header should not be overwritten
	if findHeader(result.Request.Headers, "Content-Length") != "999" {
		t.Errorf("Content-Length should preserve explicitly set value")
	}
}

func TestParseCurl_HostDerivedFromURL(t *testing.T) {
	result := ParseCurl(`curl https://api.example.com:8443/v1/data`)
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	if findHeader(result.Request.Headers, "Host") != "api.example.com:8443" {
		t.Errorf("Host = %q, want api.example.com:8443", findHeader(result.Request.Headers, "Host"))
	}
	if result.Request.Path != "/v1/data" {
		t.Errorf("Path = %q, want /v1/data", result.Request.Path)
	}
	if result.Request.Scheme != "https" {
		t.Errorf("Scheme = %q, want https", result.Request.Scheme)
	}
}

func TestParseCurl_ExplicitHostHeaderNotOverwritten(t *testing.T) {
	result := ParseCurl(`curl -H "Host: custom.host.com" https://api.example.com/v1`)
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	// Explicit Host header must win over URL-derived host
	if findHeader(result.Request.Headers, "Host") != "custom.host.com" {
		t.Errorf("Host = %q, want custom.host.com", findHeader(result.Request.Headers, "Host"))
	}
}

func TestParseCurl_NoCurlPrefix(t *testing.T) {
	// User omits the "curl " prefix — should still work
	result := ParseCurl(`https://example.com/api`)
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	if result.Request.Method != "GET" {
		t.Errorf("Method = %q, want GET", result.Request.Method)
	}
}

func TestParseCurl_SchemePreserved(t *testing.T) {
	result := ParseCurl(`curl http://example.com/api`)
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	if result.Request.Scheme != "http" {
		t.Errorf("Scheme = %q, want http", result.Request.Scheme)
	}
}

func TestShellSplit_Basic(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{`curl -X POST https://example.com`, []string{"curl", "-X", "POST", "https://example.com"}},
		{`curl -H "Content-Type: application/json"`, []string{"curl", "-H", "Content-Type: application/json"}},
		{`curl -d '{"name":"John"}'`, []string{"curl", "-d", `{"name":"John"}`}},
		{`curl -d "it's fine"`, []string{"curl", "-d", "it's fine"}},
	}
	for _, tc := range cases {
		got, err := shellSplit(tc.in)
		if err != nil {
			t.Errorf("shellSplit(%q) error: %v", tc.in, err)
			continue
		}
		if len(got) != len(tc.want) {
			t.Errorf("shellSplit(%q) = %v, want %v", tc.in, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("shellSplit(%q)[%d] = %q, want %q", tc.in, i, got[i], tc.want[i])
			}
		}
	}
}

func TestShellSplit_UnclosedQuote(t *testing.T) {
	_, err := shellSplit(`curl -H "Content-Type: application/json`)
	if err == nil {
		t.Error("expected error for unclosed double quote")
	}
}

func TestParseCurlURL(t *testing.T) {
	cases := []struct {
		in     string
		scheme string
		host   string
		path   string
	}{
		{"https://example.com/api", "https", "example.com", "/api"},
		{"http://example.com/api?q=1", "http", "example.com", "/api?q=1"},
		{"https://example.com:8080/path", "https", "example.com:8080", "/path"},
		{"https://example.com", "https", "example.com", "/"},
		{"/just/a/path", "", "", "/just/a/path"},
	}
	for _, tc := range cases {
		scheme, host, path := parseCurlURL(tc.in)
		if scheme != tc.scheme || host != tc.host || path != tc.path {
			t.Errorf("parseCurlURL(%q) = (%q, %q, %q), want (%q, %q, %q)",
				tc.in, scheme, host, path, tc.scheme, tc.host, tc.path)
		}
	}
}

func TestPercentEncode(t *testing.T) {
	cases := []struct{ in, want string }{
		{"hello", "hello"},
		{"hello world", "hello%20world"},
		{"a=b&c=d", "a%3Db%26c%3Dd"},
		{"café", "caf%C3%A9"},
		{"~-._", "~-._"},
	}
	for _, tc := range cases {
		got := percentEncode(tc.in)
		if got != tc.want {
			t.Errorf("percentEncode(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
