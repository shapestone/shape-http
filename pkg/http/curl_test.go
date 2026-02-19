package http

import (
	"strings"
	"testing"
)

func TestParseCurl_PublicAPI_SimpleGET(t *testing.T) {
	result := ParseCurl(`curl https://example.com/api/users`)
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
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
	if req.Headers.Get("Host") != "example.com" {
		t.Errorf("Host = %q, want example.com", req.Headers.Get("Host"))
	}
}

func TestParseCurl_PublicAPI_POSTwithBody(t *testing.T) {
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
	if req.Headers.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q", req.Headers.Get("Content-Type"))
	}
	if !strings.Contains(string(req.Body), "John Doe") {
		t.Errorf("body missing expected content: %s", string(req.Body))
	}
	if req.Headers.Get("Content-Length") == "" {
		t.Error("expected auto Content-Length")
	}
}

func TestParseCurl_PublicAPI_BasicAuth(t *testing.T) {
	result := ParseCurl(`curl -u admin:secret https://example.com/api/protected`)
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	want := "Basic YWRtaW46c2VjcmV0"
	if result.Request.Headers.Get("Authorization") != want {
		t.Errorf("Authorization = %q, want %q", result.Request.Headers.Get("Authorization"), want)
	}
}

func TestParseCurl_PublicAPI_MultilineCommand(t *testing.T) {
	cmd := "curl -X PUT \\\n" +
		"  https://example.com/api/users/42 \\\n" +
		`  -H "Authorization: Bearer tok" ` + "\\\n" +
		`  -d '{"active":false}'`
	result := ParseCurl(cmd)
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	req := result.Request
	if req.Method != "PUT" {
		t.Errorf("Method = %q, want PUT", req.Method)
	}
	if req.Path != "/api/users/42" {
		t.Errorf("Path = %q, want /api/users/42", req.Path)
	}
	if req.Headers.Get("Authorization") != "Bearer tok" {
		t.Errorf("Authorization = %q", req.Headers.Get("Authorization"))
	}
}

func TestParseCurl_PublicAPI_EmptyInput(t *testing.T) {
	result := ParseCurl("")
	if result.Request != nil {
		t.Error("expected nil Request for empty input")
	}
	if !result.Partial {
		t.Error("expected Partial=true")
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warnings")
	}
}

func TestParseCurl_PublicAPI_NoURL(t *testing.T) {
	result := ParseCurl("curl -X DELETE")
	if result.Request != nil {
		t.Error("expected nil Request when URL missing")
	}
	if !result.Partial {
		t.Error("expected Partial=true")
	}
}

func TestParseCurl_PublicAPI_HTTP2(t *testing.T) {
	result := ParseCurl(`curl --http2 https://example.com/`)
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	if result.Request.Version != "HTTP/2" {
		t.Errorf("Version = %q, want HTTP/2", result.Request.Version)
	}
}

func TestParseCurl_PublicAPI_FormData(t *testing.T) {
	result := ParseCurl(`curl -F "username=alice" -F "role=admin" https://example.com/users`)
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	ct := result.Request.Headers.Get("Content-Type")
	if !strings.HasPrefix(ct, "multipart/form-data") {
		t.Errorf("Content-Type = %q, want multipart/form-data", ct)
	}
	if !strings.Contains(string(result.Request.Body), "alice") {
		t.Errorf("body does not contain form data: %s", string(result.Request.Body))
	}
}

func TestParseCurl_PublicAPI_URLEncoded(t *testing.T) {
	result := ParseCurl(`curl --data-urlencode "q=hello world" https://example.com/search`)
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	if result.Request.Headers.Get("Content-Type") != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q", result.Request.Headers.Get("Content-Type"))
	}
	if !strings.Contains(string(result.Request.Body), "hello%20world") {
		t.Errorf("body = %q, want percent-encoded spaces", string(result.Request.Body))
	}
}

// TestParseCurl_PublicAPI_MatchesLenient verifies that a curl command round-trips
// through ParseCurl and produces the same key fields as UnmarshalLenient on the
// equivalent HTTP wire format.
func TestParseCurl_PublicAPI_MatchesLenient(t *testing.T) {
	curlResult := ParseCurl(`curl -X POST https://example.com:8080/api/users -H "Content-Type: application/json" -d '{"name":"John"}'`)
	if curlResult.Request == nil {
		t.Fatalf("ParseCurl returned nil Request; warnings: %v", curlResult.Warnings)
	}

	wireFormat := "POST https://example.com:8080/api/users HTTP/1.1\r\n" +
		"Content-Type: application/json\r\n" +
		"\r\n" +
		`{"name":"John"}`
	lenientResult := UnmarshalLenient([]byte(wireFormat))
	if lenientResult.Request == nil {
		t.Fatalf("UnmarshalLenient returned nil Request; warnings: %v", lenientResult.Warnings)
	}

	cr := curlResult.Request
	lr := lenientResult.Request

	if cr.Method != lr.Method {
		t.Errorf("Method mismatch: ParseCurl=%q, Lenient=%q", cr.Method, lr.Method)
	}
	if cr.Path != lr.Path {
		t.Errorf("Path mismatch: ParseCurl=%q, Lenient=%q", cr.Path, lr.Path)
	}
	if cr.Scheme != lr.Scheme {
		t.Errorf("Scheme mismatch: ParseCurl=%q, Lenient=%q", cr.Scheme, lr.Scheme)
	}
	if cr.Headers.Get("Host") != lr.Headers.Get("Host") {
		t.Errorf("Host mismatch: ParseCurl=%q, Lenient=%q", cr.Headers.Get("Host"), lr.Headers.Get("Host"))
	}
	if cr.Headers.Get("Content-Type") != lr.Headers.Get("Content-Type") {
		t.Errorf("Content-Type mismatch: ParseCurl=%q, Lenient=%q", cr.Headers.Get("Content-Type"), lr.Headers.Get("Content-Type"))
	}
}
