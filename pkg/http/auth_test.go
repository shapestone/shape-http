package http

// auth_test.go — Authentication header tests for UnmarshalRequest (strict) and
// UnmarshalLenient. Authentication is carried entirely in standard HTTP headers
// (Authorization, X-API-Key, X-Session-Token, Cookie, etc.) and requires no
// special handling by the parser — these tests confirm that all common auth
// patterns round-trip correctly through both the strict and lenient paths.

import (
	"strings"
	"testing"
)

// ── Strict (UnmarshalRequest) ──────────────────────────────────────────────

func TestAuth_Strict_BasicAuth(t *testing.T) {
	raw := "POST /api/users HTTP/1.1\r\n" +
		"Host: example.com:8080\r\n" +
		"Authorization: Basic dXNlcm5hbWU6cGFzc3dvcmQ=\r\n" +
		"Content-Type: application/json\r\n" +
		"\r\n" +
		`{"name":"John Doe","email":"john@example.com"}`

	req, err := UnmarshalRequest([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Headers.Get("Authorization") != "Basic dXNlcm5hbWU6cGFzc3dvcmQ=" {
		t.Errorf("Authorization = %q", req.Headers.Get("Authorization"))
	}
	if req.Headers.Get("Host") != "example.com:8080" {
		t.Errorf("Host = %q", req.Headers.Get("Host"))
	}
	if !strings.Contains(string(req.Body), "John Doe") {
		t.Errorf("body missing expected content: %s", req.Body)
	}
}

func TestAuth_Strict_BearerToken(t *testing.T) {
	raw := "GET /api/users HTTP/1.1\r\n" +
		"Host: api.example.com\r\n" +
		"Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U\r\n" +
		"\r\n"

	req, err := UnmarshalRequest([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	auth := req.Headers.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		t.Errorf("Authorization = %q, want Bearer prefix", auth)
	}
	if !strings.Contains(auth, "dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U") {
		t.Errorf("Authorization token truncated: %q", auth)
	}
}

func TestAuth_Strict_APIKeyHeader(t *testing.T) {
	raw := "GET /api/users HTTP/1.1\r\n" +
		"Host: 192.168.1.100:3000\r\n" +
		"X-API-Key: abc123def456\r\n" +
		"Content-Type: application/json\r\n" +
		"\r\n"

	req, err := UnmarshalRequest([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Headers.Get("X-API-Key") != "abc123def456" {
		t.Errorf("X-API-Key = %q", req.Headers.Get("X-API-Key"))
	}
}

func TestAuth_Strict_BearerWithPUT(t *testing.T) {
	raw := "PUT /api/users/1 HTTP/1.1\r\n" +
		"Host: localhost:8080\r\n" +
		"Authorization: Bearer mytoken123\r\n" +
		"Content-Type: application/json\r\n" +
		"\r\n" +
		`{"name":"Jane Doe","email":"jane@example.com"}`

	req, err := UnmarshalRequest([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Method != "PUT" {
		t.Errorf("Method = %q, want PUT", req.Method)
	}
	if req.Headers.Get("Authorization") != "Bearer mytoken123" {
		t.Errorf("Authorization = %q", req.Headers.Get("Authorization"))
	}
	if !strings.Contains(string(req.Body), "Jane Doe") {
		t.Errorf("body = %q", req.Body)
	}
}

func TestAuth_Strict_OAuthHeader(t *testing.T) {
	raw := "POST /api/users HTTP/1.1\r\n" +
		"Host: 10.0.0.5:9090\r\n" +
		`Authorization: OAuth oauth_consumer_key="key",oauth_token="token",oauth_signature="sig"` + "\r\n" +
		"Content-Type: application/json\r\n" +
		"\r\n" +
		`{"name":"Alice","email":"alice@example.com"}`

	req, err := UnmarshalRequest([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	auth := req.Headers.Get("Authorization")
	if !strings.HasPrefix(auth, "OAuth ") {
		t.Errorf("Authorization = %q, want OAuth prefix", auth)
	}
	if !strings.Contains(auth, "oauth_consumer_key") {
		t.Errorf("Authorization missing oauth fields: %q", auth)
	}
}

func TestAuth_Strict_APIKeyQueryParam(t *testing.T) {
	raw := "GET /api/users?api_key=abc123def456 HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"\r\n"

	req, err := UnmarshalRequest([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Path != "/api/users?api_key=abc123def456" {
		t.Errorf("Path = %q, want query string preserved", req.Path)
	}
}

func TestAuth_Strict_MultipleAuthHeaders(t *testing.T) {
	raw := "POST /api/users HTTP/1.1\r\n" +
		"Host: api.myservice.org:5000\r\n" +
		"X-API-Key: abc123def456\r\n" +
		"X-Session-Token: sess_xyz789\r\n" +
		"Content-Type: application/json\r\n" +
		"\r\n" +
		`{"name":"Bob","email":"bob@test.com"}`

	req, err := UnmarshalRequest([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Headers.Get("X-API-Key") != "abc123def456" {
		t.Errorf("X-API-Key = %q", req.Headers.Get("X-API-Key"))
	}
	if req.Headers.Get("X-Session-Token") != "sess_xyz789" {
		t.Errorf("X-Session-Token = %q", req.Headers.Get("X-Session-Token"))
	}
}

func TestAuth_Strict_BasicAuthDELETE(t *testing.T) {
	raw := "DELETE /api/users/42 HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Authorization: Basic am9objpzZWNyZXQ=\r\n" +
		"\r\n"

	req, err := UnmarshalRequest([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Method != "DELETE" {
		t.Errorf("Method = %q, want DELETE", req.Method)
	}
	if req.Headers.Get("Authorization") != "Basic am9objpzZWNyZXQ=" {
		t.Errorf("Authorization = %q", req.Headers.Get("Authorization"))
	}
}

func TestAuth_Strict_BearerWithIPAddress(t *testing.T) {
	raw := "GET /api/users HTTP/1.1\r\n" +
		"Host: 127.0.0.1:4000\r\n" +
		"Authorization: Bearer sk-proj-abc123xyz456\r\n" +
		"Accept: application/json\r\n" +
		"\r\n"

	req, err := UnmarshalRequest([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Headers.Get("Authorization") != "Bearer sk-proj-abc123xyz456" {
		t.Errorf("Authorization = %q", req.Headers.Get("Authorization"))
	}
	if req.Headers.Get("Host") != "127.0.0.1:4000" {
		t.Errorf("Host = %q", req.Headers.Get("Host"))
	}
}

// ── Lenient (UnmarshalLenient) ─────────────────────────────────────────────
// Lenient tests use the absolute-form request-target (method URL version) and
// also accept bare LF line endings, missing versions, and other deviations.

func TestAuth_Lenient_BasicAuth(t *testing.T) {
	raw := "POST https://example.com:8080/api/users HTTP/1.1\r\n" +
		"Authorization: Basic dXNlcm5hbWU6cGFzc3dvcmQ=\r\n" +
		"Content-Type: application/json\r\n" +
		"\r\n" +
		`{"name":"John Doe","email":"john@example.com"}`

	result := UnmarshalLenient([]byte(raw))
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	if result.Request.Headers.Get("Authorization") != "Basic dXNlcm5hbWU6cGFzc3dvcmQ=" {
		t.Errorf("Authorization = %q", result.Request.Headers.Get("Authorization"))
	}
	if result.Request.Scheme != "https" {
		t.Errorf("Scheme = %q, want https", result.Request.Scheme)
	}
	if result.Request.Headers.Get("Host") != "example.com:8080" {
		t.Errorf("Host = %q, want example.com:8080", result.Request.Headers.Get("Host"))
	}
}

func TestAuth_Lenient_BearerToken(t *testing.T) {
	raw := "GET https://api.example.com/api/users HTTP/1.1\n" +
		"Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U\n" +
		"\n"

	result := UnmarshalLenient([]byte(raw))
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	auth := result.Request.Headers.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		t.Errorf("Authorization = %q, want Bearer prefix", auth)
	}
}

func TestAuth_Lenient_APIKeyHeader(t *testing.T) {
	raw := "GET http://192.168.1.100:3000/api/users HTTP/1.1\r\n" +
		"X-API-Key: abc123def456\r\n" +
		"Content-Type: application/json\r\n" +
		"\r\n"

	result := UnmarshalLenient([]byte(raw))
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	if result.Request.Headers.Get("X-API-Key") != "abc123def456" {
		t.Errorf("X-API-Key = %q", result.Request.Headers.Get("X-API-Key"))
	}
}

func TestAuth_Lenient_BearerWithPUT(t *testing.T) {
	// Bare LF line endings and absolute-form target — lenient accepts both.
	raw := "PUT https://localhost:8080/api/users/1 HTTP/1.1\n" +
		"Authorization: Bearer mytoken123\n" +
		"Content-Type: application/json\n" +
		"\n" +
		`{"name":"Jane Doe","email":"jane@example.com"}`

	result := UnmarshalLenient([]byte(raw))
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	if result.Request.Method != "PUT" {
		t.Errorf("Method = %q, want PUT", result.Request.Method)
	}
	if result.Request.Headers.Get("Authorization") != "Bearer mytoken123" {
		t.Errorf("Authorization = %q", result.Request.Headers.Get("Authorization"))
	}
	if !strings.Contains(string(result.Request.Body), "Jane Doe") {
		t.Errorf("body = %q", result.Request.Body)
	}
}

func TestAuth_Lenient_OAuthHeader(t *testing.T) {
	raw := "POST http://10.0.0.5:9090/api/users HTTP/1.1\r\n" +
		`Authorization: OAuth oauth_consumer_key="key",oauth_token="token",oauth_signature="sig"` + "\r\n" +
		"Content-Type: application/json\r\n" +
		"\r\n" +
		`{"name":"Alice","email":"alice@example.com"}`

	result := UnmarshalLenient([]byte(raw))
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	auth := result.Request.Headers.Get("Authorization")
	if !strings.HasPrefix(auth, "OAuth ") {
		t.Errorf("Authorization = %q, want OAuth prefix", auth)
	}
}

func TestAuth_Lenient_APIKeyQueryParam(t *testing.T) {
	// Version omitted — lenient defaults to HTTP/1.1.
	raw := "GET https://example.com/api/users?api_key=abc123def456\r\n" +
		"\r\n"

	result := UnmarshalLenient([]byte(raw))
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	if result.Request.Path != "/api/users?api_key=abc123def456" {
		t.Errorf("Path = %q, want query string preserved", result.Request.Path)
	}
}

func TestAuth_Lenient_MultipleAuthHeaders(t *testing.T) {
	raw := "POST https://api.myservice.org:5000/api/users HTTP/1.1\r\n" +
		"X-API-Key: abc123def456\r\n" +
		"X-Session-Token: sess_xyz789\r\n" +
		"Content-Type: application/json\r\n" +
		"\r\n" +
		`{"name":"Bob","email":"bob@test.com"}`

	result := UnmarshalLenient([]byte(raw))
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	if result.Request.Headers.Get("X-API-Key") != "abc123def456" {
		t.Errorf("X-API-Key = %q", result.Request.Headers.Get("X-API-Key"))
	}
	if result.Request.Headers.Get("X-Session-Token") != "sess_xyz789" {
		t.Errorf("X-Session-Token = %q", result.Request.Headers.Get("X-Session-Token"))
	}
}

func TestAuth_Lenient_BasicAuthDELETE(t *testing.T) {
	raw := "DELETE http://example.com/api/users/42 HTTP/1.1\r\n" +
		"Authorization: Basic am9objpzZWNyZXQ=\r\n" +
		"\r\n"

	result := UnmarshalLenient([]byte(raw))
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	if result.Request.Method != "DELETE" {
		t.Errorf("Method = %q, want DELETE", result.Request.Method)
	}
	if result.Request.Headers.Get("Authorization") != "Basic am9objpzZWNyZXQ=" {
		t.Errorf("Authorization = %q", result.Request.Headers.Get("Authorization"))
	}
}

func TestAuth_Lenient_BearerWithIPAddress(t *testing.T) {
	raw := "GET http://127.0.0.1:4000/api/users HTTP/1.1\r\n" +
		"Authorization: Bearer sk-proj-abc123xyz456\r\n" +
		"Accept: application/json\r\n" +
		"\r\n"

	result := UnmarshalLenient([]byte(raw))
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	if result.Request.Headers.Get("Authorization") != "Bearer sk-proj-abc123xyz456" {
		t.Errorf("Authorization = %q", result.Request.Headers.Get("Authorization"))
	}
	if result.Request.Scheme != "http" {
		t.Errorf("Scheme = %q, want http", result.Request.Scheme)
	}
}
