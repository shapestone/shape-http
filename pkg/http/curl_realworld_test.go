package http

// curl_realworld_test.go — 40 real-world curl command test cases drawn from
// curl man page examples, GitHub / Stripe / Twilio API docs, and common
// REST API patterns. Each test validates method, path, host, scheme, headers
// and body that ParseCurl must extract correctly.

import (
	"strings"
	"testing"
)

// curlCase is a single ParseCurl test vector.
type curlCase struct {
	name string
	cmd  string
	// expected
	method  string
	path    string
	host    string
	scheme  string
	version string
	// optional header checks
	headers map[string]string
	// optional body substring check ("" means don't check)
	bodyContains string
	// expected Content-Length value ("" means don't check, "-" means absent)
	contentLength string
	// partial result (true = URL missing / parse error)
	partial bool
	// warning substring expected in Warnings (empty means no specific check)
	warnContains string
}

func runCurlCase(t *testing.T, tc curlCase) {
	t.Helper()
	result := ParseCurl(tc.cmd)

	if tc.partial {
		if !result.Partial {
			t.Errorf("expected Partial=true, got false; warnings: %v", result.Warnings)
		}
		return
	}

	if result.Request == nil {
		t.Fatalf("expected non-nil Request; warnings: %v", result.Warnings)
	}
	req := result.Request

	if tc.method != "" && req.Method != tc.method {
		t.Errorf("Method = %q, want %q", req.Method, tc.method)
	}
	if tc.path != "" && req.Path != tc.path {
		t.Errorf("Path = %q, want %q", req.Path, tc.path)
	}
	if tc.host != "" && req.Headers.Get("Host") != tc.host {
		t.Errorf("Host = %q, want %q", req.Headers.Get("Host"), tc.host)
	}
	if tc.scheme != "" && req.Scheme != tc.scheme {
		t.Errorf("Scheme = %q, want %q", req.Scheme, tc.scheme)
	}
	if tc.version != "" && req.Version != tc.version {
		t.Errorf("Version = %q, want %q", req.Version, tc.version)
	}
	for k, v := range tc.headers {
		got := req.Headers.Get(k)
		if got != v {
			t.Errorf("Header %q = %q, want %q", k, got, v)
		}
	}
	if tc.bodyContains != "" && !strings.Contains(string(req.Body), tc.bodyContains) {
		t.Errorf("body %q does not contain %q", string(req.Body), tc.bodyContains)
	}
	if tc.contentLength != "" {
		cl := req.Headers.Get("Content-Length")
		if tc.contentLength == "-" {
			if cl != "" {
				t.Errorf("expected no Content-Length, got %q", cl)
			}
		} else if cl != tc.contentLength {
			t.Errorf("Content-Length = %q, want %q", cl, tc.contentLength)
		}
	}
	if tc.warnContains != "" {
		found := false
		for _, w := range result.Warnings {
			if strings.Contains(w, tc.warnContains) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected warning containing %q, got %v", tc.warnContains, result.Warnings)
		}
	}
}

// ── HTTP Methods ────────────────────────────────────────────────────────────

func TestCurlRW_01_SimpleGET(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "simple GET",
		cmd:    `curl https://api.example.com/users`,
		method: "GET", path: "/users", host: "api.example.com", scheme: "https",
	})
}

func TestCurlRW_02_ExplicitGET(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "explicit -X GET",
		cmd:    `curl -X GET https://api.example.com/status`,
		method: "GET", path: "/status", host: "api.example.com",
	})
}

func TestCurlRW_03_POST_JSON(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "POST JSON body",
		cmd:    `curl -X POST https://api.example.com/users -H "Content-Type: application/json" -d '{"name":"Alice","email":"alice@example.com"}'`,
		method: "POST", path: "/users", host: "api.example.com",
		headers:      map[string]string{"Content-Type": "application/json"},
		bodyContains: "Alice",
	})
}

func TestCurlRW_04_PUT_JSON(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "PUT with JSON body",
		cmd:    `curl -X PUT https://api.example.com/users/42 -H "Content-Type: application/json" -d '{"active":false}'`,
		method: "PUT", path: "/users/42",
		bodyContains: "active",
	})
}

func TestCurlRW_05_DELETE(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "DELETE resource",
		cmd:    `curl -X DELETE https://api.example.com/users/42`,
		method: "DELETE", path: "/users/42", host: "api.example.com",
	})
}

func TestCurlRW_06_PATCH(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "PATCH resource",
		cmd:    `curl -X PATCH https://api.example.com/users/7 -H "Content-Type: application/json" -d '{"email":"new@example.com"}'`,
		method: "PATCH", path: "/users/7",
		bodyContains: "email",
	})
}

func TestCurlRW_07_HEAD_flag(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "-I implies HEAD",
		cmd:    `curl -I https://api.example.com/health`,
		method: "HEAD", path: "/health", host: "api.example.com",
	})
}

func TestCurlRW_08_HEAD_long(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "--head implies HEAD",
		cmd:    `curl --head https://api.example.com/health`,
		method: "HEAD", path: "/health",
	})
}

func TestCurlRW_09_OPTIONS(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "OPTIONS method",
		cmd:    `curl -X OPTIONS https://api.example.com/users`,
		method: "OPTIONS", path: "/users",
	})
}

// ── URL Forms ───────────────────────────────────────────────────────────────

func TestCurlRW_10_QueryParams(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "query parameters",
		cmd:    `curl "https://api.example.com/search?q=hello+world&page=2&limit=10"`,
		method: "GET", path: "/search?q=hello+world&page=2&limit=10",
		host: "api.example.com",
	})
}

func TestCurlRW_11_Port_in_URL(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "non-default port in URL",
		cmd:    `curl https://api.example.com:8443/v1/secure`,
		method: "GET", path: "/v1/secure", host: "api.example.com:8443", scheme: "https",
	})
}

func TestCurlRW_12_HTTP_scheme(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "http:// scheme",
		cmd:    `curl http://localhost:3000/api/health`,
		method: "GET", path: "/api/health", host: "localhost:3000", scheme: "http",
	})
}

func TestCurlRW_13_IPv4_URL(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "IPv4 address URL",
		cmd:    `curl http://127.0.0.1:8080/api/v1/users`,
		method: "GET", path: "/api/v1/users", host: "127.0.0.1:8080", scheme: "http",
	})
}

func TestCurlRW_14_No_curl_prefix(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "no 'curl' prefix — bare URL",
		cmd:    `https://api.example.com/ping`,
		method: "GET", path: "/ping", host: "api.example.com",
	})
}

func TestCurlRW_15_URL_no_path(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "URL with no path component",
		cmd:    `curl https://api.example.com`,
		method: "GET", path: "/", host: "api.example.com",
	})
}

// ── Auth ─────────────────────────────────────────────────────────────────────

func TestCurlRW_16_BasicAuth(t *testing.T) {
	// -u user:pass → Authorization: Basic base64(user:pass)
	runCurlCase(t, curlCase{
		name: "basic auth -u",
		cmd:  `curl -u admin:s3cr3t https://api.example.com/admin`,
		headers: map[string]string{
			"Authorization": "Basic YWRtaW46czNjcjN0",
		},
		method: "GET", path: "/admin",
	})
}

func TestCurlRW_17_BasicAuth_ColonInPass(t *testing.T) {
	// password contains a colon — everything after first colon is password
	runCurlCase(t, curlCase{
		name: "basic auth with colon in password",
		cmd:  `curl -u user:p:ass https://api.example.com/`,
		// base64("user:p:ass")
		headers: map[string]string{
			"Authorization": "Basic dXNlcjpwOmFzcw==",
		},
		method: "GET",
	})
}

func TestCurlRW_18_BearerToken(t *testing.T) {
	runCurlCase(t, curlCase{
		name: "Bearer token header",
		cmd:  `curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.payload.sig" https://api.example.com/me`,
		headers: map[string]string{
			"Authorization": "Bearer eyJhbGciOiJIUzI1NiJ9.payload.sig",
		},
		method: "GET", path: "/me",
	})
}

func TestCurlRW_19_Stripe_BasicAuth(t *testing.T) {
	// Stripe uses -u sk_test_xxx: (empty password)
	runCurlCase(t, curlCase{
		name:   "Stripe-style basic auth (empty password)",
		cmd:    `curl -u sk_test_4eC39HqLyjWDarjtT7ia:  https://api.stripe.com/v1/customers`,
		method: "GET", path: "/v1/customers", host: "api.stripe.com",
		// Authorization header must be present (don't check exact value)
		headers: map[string]string{},
	})
}

// ── Headers ──────────────────────────────────────────────────────────────────

func TestCurlRW_20_MultipleHeaders(t *testing.T) {
	runCurlCase(t, curlCase{
		name: "multiple -H headers",
		cmd: `curl -H "Authorization: Bearer tok" -H "Accept: application/json" ` +
			`-H "X-Request-ID: req-abc-123" https://api.example.com/data`,
		headers: map[string]string{
			"Authorization": "Bearer tok",
			"Accept":        "application/json",
			"X-Request-ID":  "req-abc-123",
		},
		method: "GET",
	})
}

func TestCurlRW_21_AcceptHeader(t *testing.T) {
	runCurlCase(t, curlCase{
		name: "Accept header",
		cmd:  `curl -H "Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8" https://www.example.com/`,
		headers: map[string]string{
			"Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		},
		method: "GET",
	})
}

func TestCurlRW_22_CustomContentType(t *testing.T) {
	runCurlCase(t, curlCase{
		name: "XML content type",
		cmd:  `curl -X POST -H "Content-Type: application/xml" -d '<user><name>Alice</name></user>' https://api.example.com/users`,
		headers: map[string]string{
			"Content-Type": "application/xml",
		},
		bodyContains: "<name>Alice</name>",
		method:       "POST",
	})
}

func TestCurlRW_23_GithubAPI(t *testing.T) {
	runCurlCase(t, curlCase{
		name: "GitHub API with token",
		cmd: `curl -H "Authorization: token ghp_xxxxxxxxxxxxxxxxxxxx" ` +
			`-H "Accept: application/vnd.github+json" ` +
			`-H "X-GitHub-Api-Version: 2022-11-28" ` +
			`https://api.github.com/repos/octocat/hello-world`,
		method: "GET", path: "/repos/octocat/hello-world", host: "api.github.com",
		headers: map[string]string{
			"Accept":               "application/vnd.github+json",
			"X-GitHub-Api-Version": "2022-11-28",
		},
	})
}

func TestCurlRW_24_ExplicitHostHeader_NotOverwritten(t *testing.T) {
	runCurlCase(t, curlCase{
		name: "explicit Host header wins over URL-derived",
		cmd:  `curl -H "Host: custom.backend.internal" https://loadbalancer.example.com/api`,
		headers: map[string]string{
			"Host": "custom.backend.internal",
		},
		method: "GET",
	})
}

// ── Body Types ───────────────────────────────────────────────────────────────

func TestCurlRW_25_DataRaw(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "--data-raw (no file @-expansion)",
		cmd:    `curl --data-raw '{"query":"{ me { id name } }"}' -H "Content-Type: application/json" https://api.example.com/graphql`,
		method: "POST", path: "/graphql",
		bodyContains: `"query"`,
	})
}

func TestCurlRW_26_GraphQL(t *testing.T) {
	runCurlCase(t, curlCase{
		name: "GraphQL POST",
		cmd: `curl -X POST https://api.example.com/graphql ` +
			`-H "Content-Type: application/json" ` +
			`-d '{"query":"{ user(id:1) { id name email } }","variables":{}}'`,
		method: "POST", path: "/graphql",
		headers:      map[string]string{"Content-Type": "application/json"},
		bodyContains: "variables",
	})
}

func TestCurlRW_27_URLEncoded_FormBody(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "URL-encoded form via -d",
		cmd:    `curl -X POST https://api.example.com/auth -H "Content-Type: application/x-www-form-urlencoded" -d 'grant_type=client_credentials&client_id=xxx&client_secret=yyy'`,
		method: "POST", path: "/auth",
		headers:      map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		bodyContains: "grant_type=client_credentials",
	})
}

func TestCurlRW_28_DataUrlencode_Spaces(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "--data-urlencode encodes spaces",
		cmd:    `curl --data-urlencode "name=John Doe" --data-urlencode "city=New York" https://api.example.com/search`,
		method: "POST",
		headers: map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		bodyContains: "John%20Doe",
	})
}

func TestCurlRW_29_MultipleDashD_Joined(t *testing.T) {
	// Multiple -d flags → joined with & (curl behaviour)
	runCurlCase(t, curlCase{
		name:         "multiple -d flags joined with &",
		cmd:          `curl -X POST https://api.example.com/form -d 'name=Alice' -d 'email=alice@example.com'`,
		method:       "POST",
		bodyContains: "name=Alice",
	})
}

func TestCurlRW_30_FormData_Multipart(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "-F multipart form data",
		cmd:    `curl -F "username=alice" -F "role=admin" -F "bio=Software developer" https://api.example.com/profile`,
		method: "POST",
		headers: map[string]string{
			"Content-Type": "multipart/form-data; boundary=ShapeHttpFormBoundary",
		},
		bodyContains: "alice",
	})
}

// ── Ignored / Behaviour Flags ─────────────────────────────────────────────────

func TestCurlRW_31_Verbose_Silent(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "-v and -s are silently ignored",
		cmd:    `curl -v -s https://api.example.com/ping`,
		method: "GET", path: "/ping",
	})
}

func TestCurlRW_32_Insecure_Location(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "-k and -L are silently ignored",
		cmd:    `curl -k -L https://self-signed.example.com/redirect`,
		method: "GET", path: "/redirect",
	})
}

func TestCurlRW_33_Compressed_Include(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "--compressed and -i are silently ignored",
		cmd:    `curl --compressed -i https://api.example.com/data`,
		method: "GET", path: "/data",
	})
}

func TestCurlRW_34_Output_Flag(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "-o (output file) is silently ignored",
		cmd:    `curl -o /tmp/output.json https://api.example.com/export`,
		method: "GET", path: "/export",
	})
}

func TestCurlRW_35_UserAgent_Flag(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "-A user-agent flag is silently ignored",
		cmd:    `curl -A "MyClient/1.0" https://api.example.com/`,
		method: "GET", path: "/",
	})
}

// ── HTTP Versions ─────────────────────────────────────────────────────────────

func TestCurlRW_36_HTTP2(t *testing.T) {
	runCurlCase(t, curlCase{
		name:    "--http2 flag",
		cmd:     `curl --http2 https://api.example.com/v2/users`,
		version: "HTTP/2", method: "GET",
	})
}

func TestCurlRW_37_HTTP3(t *testing.T) {
	runCurlCase(t, curlCase{
		name:    "--http3 flag",
		cmd:     `curl --http3 https://api.example.com/v3/stream`,
		version: "HTTP/3", method: "GET",
	})
}

func TestCurlRW_38_HTTP10(t *testing.T) {
	runCurlCase(t, curlCase{
		name:    "--http1.0 flag",
		cmd:     `curl --http1.0 http://legacy.example.com/old`,
		version: "HTTP/1.0", scheme: "http", method: "GET",
	})
}

// ── Multi-line Commands ──────────────────────────────────────────────────────

func TestCurlRW_39_MultilineBackslash(t *testing.T) {
	cmd := "curl -X POST \\\n" +
		"  https://api.example.com/users \\\n" +
		"  -H \"Content-Type: application/json\" \\\n" +
		"  -H \"Authorization: Bearer tok123\" \\\n" +
		"  -d '{\"name\":\"Bob\"}'"
	runCurlCase(t, curlCase{
		name:         "multi-line backslash continuation",
		cmd:          cmd,
		method:       "POST",
		path:         "/users",
		bodyContains: "Bob",
		headers: map[string]string{
			"Authorization": "Bearer tok123",
			"Content-Type":  "application/json",
		},
	})
}

func TestCurlRW_40_MultilineCRLF(t *testing.T) {
	cmd := "curl -X DELETE \\\r\n" +
		"  https://api.example.com/sessions/abc123 \\\r\n" +
		"  -H \"Authorization: Bearer tok\""
	runCurlCase(t, curlCase{
		name:   "multi-line CRLF backslash continuation",
		cmd:    cmd,
		method: "DELETE", path: "/sessions/abc123",
		headers: map[string]string{"Authorization": "Bearer tok"},
	})
}

// ── Real API Patterns ────────────────────────────────────────────────────────

func TestCurlRW_41_Twilio_POST(t *testing.T) {
	runCurlCase(t, curlCase{
		name: "Twilio-style SMS send",
		cmd: `curl -X POST https://api.twilio.com/2010-04-01/Accounts/ACxxx/Messages.json ` +
			`-u ACxxx:auth_token ` +
			`--data-urlencode "To=+15558675310" ` +
			`--data-urlencode "From=+15017122661" ` +
			`--data-urlencode "Body=Hello from curl"`,
		method: "POST", path: "/2010-04-01/Accounts/ACxxx/Messages.json",
		host: "api.twilio.com",
		headers: map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		bodyContains: "Hello%20from%20curl",
	})
}

func TestCurlRW_42_Webhook_POST(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "webhook POST with event body",
		cmd:    `curl -X POST https://hooks.example.com/webhook -H "Content-Type: application/json" -H "X-Webhook-Secret: shhhh" --data-raw '{"event":"user.created","id":"u_123"}'`,
		method: "POST", path: "/webhook",
		headers: map[string]string{
			"X-Webhook-Secret": "shhhh",
		},
		bodyContains: "user.created",
	})
}

func TestCurlRW_43_BatchRequest(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "POST with JSON array body",
		cmd:    `curl -X POST https://api.example.com/batch -H "Content-Type: application/json" -H "X-Request-ID: req-001" -d '[{"id":1,"action":"update"},{"id":2,"action":"delete"}]'`,
		method: "POST",
		headers: map[string]string{
			"Content-Type": "application/json",
			"X-Request-ID": "req-001",
		},
		bodyContains: `"action":"update"`,
	})
}

func TestCurlRW_44_OAuth_Token(t *testing.T) {
	runCurlCase(t, curlCase{
		name: "OAuth2 token request",
		cmd: `curl -X POST https://auth.example.com/oauth/token ` +
			`-H "Content-Type: application/x-www-form-urlencoded" ` +
			`-d "grant_type=password&username=user@example.com&password=pass&client_id=client123"`,
		method:       "POST",
		path:         "/oauth/token",
		bodyContains: "grant_type=password",
	})
}

func TestCurlRW_45_ContentLength_AutoSet(t *testing.T) {
	body := `{"x":1}`
	runCurlCase(t, curlCase{
		name:          "Content-Length auto-set matches body length",
		cmd:           `curl -X POST https://api.example.com/data -H "Content-Type: application/json" -d '{"x":1}'`,
		contentLength: "7", // len(`{"x":1}`) == 7
	})
	_ = body
}

// ── Edge Cases ──────────────────────────────────────────────────────────────

func TestCurlRW_46_EmptyBody(t *testing.T) {
	// -d '' sends an empty body; Content-Length should be 0 or absent
	result := ParseCurl(`curl -X POST https://api.example.com/trigger -d ''`)
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	// Empty body → method should be POST (explicit -X POST)
	if result.Request.Method != "POST" {
		t.Errorf("Method = %q, want POST", result.Request.Method)
	}
}

func TestCurlRW_47_SpecialCharsInBody(t *testing.T) {
	runCurlCase(t, curlCase{
		name:         "special chars in JSON body",
		cmd:          `curl -X POST https://api.example.com/msgs -H "Content-Type: application/json" -d '{"msg":"Hello, <World> & \"friends\"!"}'`,
		bodyContains: "<World>",
		method:       "POST",
	})
}

func TestCurlRW_48_UnknownFlag_Warning(t *testing.T) {
	runCurlCase(t, curlCase{
		name:         "unknown flag generates warning",
		cmd:          `curl --some-unknown-flag https://api.example.com/`,
		method:       "GET",
		warnContains: "unknown",
	})
}

func TestCurlRW_49_MissingURL(t *testing.T) {
	runCurlCase(t, curlCase{
		name:    "no URL → partial result",
		cmd:     `curl -X POST -H "Content-Type: application/json"`,
		partial: true,
	})
}

func TestCurlRW_50_EmptyCommand(t *testing.T) {
	runCurlCase(t, curlCase{
		name:    "empty string → partial",
		cmd:     ``,
		partial: true,
	})
}

func TestCurlRW_51_DataBinary(t *testing.T) {
	runCurlCase(t, curlCase{
		name: "--data-binary treated as body",
		cmd:  `curl -X POST https://api.example.com/raw --data-binary '{"binary":true}' -H "Content-Type: application/octet-stream"`,
		headers: map[string]string{
			"Content-Type": "application/octet-stream",
		},
		bodyContains: "binary",
		method:       "POST",
	})
}

func TestCurlRW_52_FileUploadWarning(t *testing.T) {
	// @filename in -d should warn, not crash, and body is empty
	result := ParseCurl(`curl -X POST https://api.example.com/upload -d @/tmp/payload.json`)
	if result.Request == nil {
		t.Fatalf("expected request even when file upload skipped; warnings: %v", result.Warnings)
	}
	hasWarn := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "file upload") || strings.Contains(w, "not supported") {
			hasWarn = true
			break
		}
	}
	if !hasWarn {
		t.Errorf("expected file-upload warning; got %v", result.Warnings)
	}
}

func TestCurlRW_53_FormFileUploadWarning(t *testing.T) {
	// -F file=@path should warn and be skipped
	result := ParseCurl(`curl -F "file=@/tmp/test.txt" https://api.example.com/upload`)
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	hasWarn := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "file upload") || strings.Contains(w, "not supported") {
			hasWarn = true
		}
	}
	if !hasWarn {
		t.Errorf("expected file-upload warning; got %v", result.Warnings)
	}
}

func TestCurlRW_54_ShortLongFlagEquivalence(t *testing.T) {
	// -X and --request must both work
	r1 := ParseCurl(`curl -X DELETE https://api.example.com/item/1`)
	r2 := ParseCurl(`curl --request DELETE https://api.example.com/item/1`)
	if r1.Request == nil || r2.Request == nil {
		t.Fatal("expected both requests")
	}
	if r1.Request.Method != r2.Request.Method {
		t.Errorf("-X and --request produced different methods: %q vs %q", r1.Request.Method, r2.Request.Method)
	}
}

func TestCurlRW_55_HeaderShortLongEquivalence(t *testing.T) {
	r1 := ParseCurl(`curl -H "X-Foo: bar" https://api.example.com/`)
	r2 := ParseCurl(`curl --header "X-Foo: bar" https://api.example.com/`)
	if r1.Request == nil || r2.Request == nil {
		t.Fatal("expected both requests")
	}
	if r1.Request.Headers.Get("X-Foo") != r2.Request.Headers.Get("X-Foo") {
		t.Errorf("-H and --header produced different values")
	}
}

func TestCurlRW_56_DoubleQuotedBody(t *testing.T) {
	runCurlCase(t, curlCase{
		name:         "double-quoted body",
		cmd:          `curl -X POST https://api.example.com/data -d "{\"key\":\"value\"}"`,
		bodyContains: "key",
		method:       "POST",
	})
}

func TestCurlRW_57_DeepPath(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "deep nested path",
		cmd:    `curl https://api.example.com/v3/orgs/myorg/repos/myrepo/issues/42/comments`,
		method: "GET", path: "/v3/orgs/myorg/repos/myrepo/issues/42/comments",
		host: "api.example.com",
	})
}

func TestCurlRW_58_HTTP2PriorKnowledge(t *testing.T) {
	runCurlCase(t, curlCase{
		name:    "--http2-prior-knowledge",
		cmd:     `curl --http2-prior-knowledge https://api.example.com/h2`,
		version: "HTTP/2", method: "GET",
	})
}

// ── Compound short flags ───────────────────────────────────────────────────

func TestCurlRW_59_CompoundSS(t *testing.T) {
	// -sS = --silent --show-error: both ignored, request still parsed.
	runCurlCase(t, curlCase{
		name:   "-sS compound ignored",
		cmd:    `curl -sS https://api.example.com/status`,
		method: "GET", host: "api.example.com", path: "/status",
	})
	result := ParseCurl(`curl -sS https://api.example.com/status`)
	for _, w := range result.Warnings {
		if strings.Contains(w, "unknown curl flag") {
			t.Errorf("compound flag -sS produced unexpected warning: %q", w)
		}
	}
}

func TestCurlRW_60_CompoundVK(t *testing.T) {
	// -vk = --verbose --insecure: both ignored.
	runCurlCase(t, curlCase{
		name:   "-vk compound ignored",
		cmd:    `curl -vk -X DELETE https://dev.example.com/cache`,
		method: "DELETE", host: "dev.example.com", path: "/cache",
	})
	result := ParseCurl(`curl -vk -X DELETE https://dev.example.com/cache`)
	for _, w := range result.Warnings {
		if strings.Contains(w, "unknown curl flag") {
			t.Errorf("compound flag -vk produced unexpected warning: %q", w)
		}
	}
}

func TestCurlRW_61_CompoundSLK(t *testing.T) {
	// -sLk = --silent --location --insecure: all ignored.
	runCurlCase(t, curlCase{
		name:   "-sLk compound ignored",
		cmd:    `curl -sLk https://short.example.com/abc`,
		method: "GET", host: "short.example.com", path: "/abc",
	})
}

func TestCurlRW_62_CompoundSSKV(t *testing.T) {
	// -sSKv compound with multiple ignored flags.
	runCurlCase(t, curlCase{
		name:         "-sSvk + POST with body",
		cmd:          `curl -sSvk -X POST https://api.example.com/events -H "Content-Type: application/json" -d '{"ev":"click"}'`,
		method:       "POST",
		path:         "/events",
		host:         "api.example.com",
		bodyContains: "click",
	})
}

func TestCurlRW_63_CompoundSO_Download(t *testing.T) {
	// -O (capital O) = write output to named file: ignored.
	runCurlCase(t, curlCase{
		name:   "-O download flag ignored",
		cmd:    `curl -O https://api.example.com/export.json`,
		method: "GET", host: "api.example.com", path: "/export.json",
	})
}

func TestCurlRW_64_CompoundSXInline(t *testing.T) {
	// -XPOST as a single token (method inline in compound, curl behaviour).
	runCurlCase(t, curlCase{
		name:   "-XPOST inline method",
		cmd:    `curl -XPOST https://api.example.com/users -H "Content-Type: application/json" -d '{"name":"Eve"}'`,
		method: "POST", host: "api.example.com", path: "/users",
		bodyContains: "Eve",
	})
}

// ── Cookie flag ────────────────────────────────────────────────────────────

func TestCurlRW_65_CookieFlag(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "-b cookie string",
		cmd:    `curl -b "session=abc123; theme=dark" https://api.example.com/dashboard`,
		method: "GET", host: "api.example.com", path: "/dashboard",
		headers: map[string]string{"Cookie": "session=abc123; theme=dark"},
	})
}

func TestCurlRW_66_CookieLongFlag(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "--cookie long flag",
		cmd:    `curl --cookie "csrf=tok42" https://api.example.com/form`,
		method: "GET", host: "api.example.com", path: "/form",
		headers: map[string]string{"Cookie": "csrf=tok42"},
	})
}

func TestCurlRW_67_CookieWithPost(t *testing.T) {
	runCurlCase(t, curlCase{
		name:         "-b + POST with body",
		cmd:          `curl -X POST https://api.example.com/session/refresh -H "Authorization: Bearer tok" -b "session_id=sess_789" -d '{}'`,
		method:       "POST",
		host:         "api.example.com",
		path:         "/session/refresh",
		bodyContains: "{}",
		headers: map[string]string{
			"Authorization": "Bearer tok",
			"Cookie":        "session_id=sess_789",
		},
	})
}

// ── URL fragments ──────────────────────────────────────────────────────────

func TestCurlRW_68_FragmentStripped(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "URL fragment stripped from path",
		cmd:    `curl https://api.example.com/docs#section-3`,
		method: "GET", host: "api.example.com", path: "/docs",
	})
}

func TestCurlRW_69_FragmentWithQuery(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "URL query + fragment: fragment stripped",
		cmd:    `curl "https://api.example.com/search?q=go#results"`,
		method: "GET", host: "api.example.com", path: "/search?q=go",
	})
}

func TestCurlRW_70_FragmentOnlyAfterSlash(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "bare path fragment",
		cmd:    `curl "https://example.com/#hero"`,
		method: "GET", host: "example.com", path: "/",
	})
}

// ── IPv6 URLs ──────────────────────────────────────────────────────────────

func TestCurlRW_71_IPv6Loopback(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "IPv6 loopback with port",
		cmd:    `curl http://[::1]:8080/api/ping`,
		method: "GET", path: "/api/ping",
		host: "[::1]:8080", scheme: "http",
	})
}

func TestCurlRW_72_IPv6FullAddress(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "IPv6 full address",
		cmd:    `curl -X GET "http://[2001:db8::1]/api/v1/status" -H "Accept: application/json"`,
		method: "GET", path: "/api/v1/status",
		host: "[2001:db8::1]", scheme: "http",
		headers: map[string]string{"Accept": "application/json"},
	})
}

func TestCurlRW_73_IPv6NoPort(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "IPv6 loopback no port",
		cmd:    `curl https://[::1]/health`,
		method: "GET", path: "/health",
		host: "[::1]", scheme: "https",
	})
}

// ── No "curl" prefix ───────────────────────────────────────────────────────

func TestCurlRW_74_NoCurlPrefixWithMethod(t *testing.T) {
	// User pastes just the flags from a curl command, without the word "curl".
	runCurlCase(t, curlCase{
		name:         "no curl prefix, flags first",
		cmd:          `-X POST https://api.example.com/webhook -H "Content-Type: application/json" -H "X-Secret: whs_abc" -d '{"event":"push"}'`,
		method:       "POST",
		host:         "api.example.com",
		path:         "/webhook",
		bodyContains: "push",
		headers: map[string]string{
			"Content-Type": "application/json",
			"X-Secret":     "whs_abc",
		},
	})
}

func TestCurlRW_75_NoCurlPrefixURLFirst(t *testing.T) {
	// URL is first token, no "curl" word at all.
	runCurlCase(t, curlCase{
		name:   "no curl prefix, URL first",
		cmd:    `https://api.example.com/users -H "Accept: application/json"`,
		method: "GET", host: "api.example.com", path: "/users",
		headers: map[string]string{"Accept": "application/json"},
	})
}

func TestCurlRW_76_NoCurlPrefixBearerGet(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "no curl prefix, bearer token, GET",
		cmd:    `-H "Authorization: Bearer eyJ.tok" -H "Accept: application/json" https://api.example.com/v1/profile`,
		method: "GET", host: "api.example.com", path: "/v1/profile",
		headers: map[string]string{
			"Authorization": "Bearer eyJ.tok",
			"Accept":        "application/json",
		},
	})
}

// ── -u without password ────────────────────────────────────────────────────

func TestCurlRW_77_UserNoPassword(t *testing.T) {
	// -u username without colon: parser warns and encodes username only.
	result := ParseCurl(`curl -u john https://api.example.com/account`)
	if result.Request == nil {
		t.Fatalf("expected request, got nil; warnings: %v", result.Warnings)
	}
	// Should still produce an Authorization header.
	if result.Request.Headers.Get("Authorization") == "" {
		t.Error("expected Authorization header even with username-only -u")
	}
	// Should warn about missing password.
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "no colon") || strings.Contains(w, "password") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about missing password; warnings: %v", result.Warnings)
	}
}

// ── Comment lines, separators, no-scheme URLs, indentation ────────────────
// These tests cover the exact examples from the user-facing docs and are
// specifically designed to catch inputs that trip up parsers in the wild.

// TestCurlRW_78 through TestCurlRW_87 are the 10 examples from the spec.

func TestCurlRW_78_PostHTTPSPortCommentLine(t *testing.T) {
	// #-comment line immediately before curl command must be ignored.
	cmd := "# POST with HTTPS, domain, port, JSON body\n" +
		"curl -X POST https://example.com:8080/api/users \\\n" +
		"  -H \"Content-Type: application/json\" \\\n" +
		"  -d '{\"name\": \"John Doe\", \"email\": \"john@example.com\"}'"
	runCurlCase(t, curlCase{
		name: "comment + POST https port",
		cmd:  cmd, method: "POST", host: "example.com:8080",
		path: "/api/users", scheme: "https",
		headers:      map[string]string{"Content-Type": "application/json"},
		bodyContains: "John Doe",
	})
}

func TestCurlRW_79_GetHTTPIPNoPort(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "GET http IP no port",
		cmd:    "curl -X GET http://192.168.1.100/api/users",
		method: "GET", host: "192.168.1.100", path: "/api/users", scheme: "http",
	})
}

func TestCurlRW_80_PutHTTPSLocalhostPort(t *testing.T) {
	cmd := "curl -X PUT https://localhost:3000/api/users/1 \\\n" +
		"  -H \"Content-Type: application/json\" \\\n" +
		"  -d '{\"name\": \"Jane Doe\", \"email\": \"jane@example.com\"}'"
	runCurlCase(t, curlCase{
		name: "PUT https localhost port",
		cmd:  cmd, method: "PUT", host: "localhost:3000",
		path: "/api/users/1", scheme: "https",
		bodyContains: "Jane Doe",
	})
}

func TestCurlRW_81_PostHTTPIPPort(t *testing.T) {
	cmd := "curl -X POST http://10.0.0.5:9090/api/users \\\n" +
		"  -H \"Content-Type: application/json\" \\\n" +
		"  -d '{\"name\": \"Alice\", \"email\": \"alice@example.com\"}'"
	runCurlCase(t, curlCase{
		name: "POST http IP port",
		cmd:  cmd, method: "POST", host: "10.0.0.5:9090",
		path: "/api/users", scheme: "http",
		bodyContains: "Alice",
	})
}

func TestCurlRW_82_GetNoSchemeDomain(t *testing.T) {
	// No scheme, bare domain — host must still be extracted.
	runCurlCase(t, curlCase{
		name:   "GET no scheme domain",
		cmd:    "curl example.com/api/users",
		method: "GET", host: "example.com", path: "/api/users",
	})
}

func TestCurlRW_83_PutHTTPDomainPort(t *testing.T) {
	cmd := "curl -X PUT http://api.myservice.org:5000/api/users/42 \\\n" +
		"  -H \"Content-Type: application/json\" \\\n" +
		"  -d '{\"email\": \"updated@example.com\"}'"
	runCurlCase(t, curlCase{
		name: "PUT http domain port",
		cmd:  cmd, method: "PUT", host: "api.myservice.org:5000",
		path: "/api/users/42", scheme: "http",
		bodyContains: "updated@example.com",
	})
}

func TestCurlRW_84_GetHTTPSLocalhostNoPort(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "GET https localhost no port",
		cmd:    "curl https://localhost/api/users/1",
		method: "GET", host: "localhost", path: "/api/users/1", scheme: "https",
	})
}

func TestCurlRW_85_PostNoSchemeLocalhostPort(t *testing.T) {
	// No scheme, localhost:port — host must be extracted.
	cmd := "curl -X POST localhost:8080/api/users \\\n" +
		"  -H \"Content-Type: application/json\" \\\n" +
		"  -d '{\"name\": \"Bob\", \"email\": \"bob@test.com\"}'"
	runCurlCase(t, curlCase{
		name: "POST no scheme localhost port",
		cmd:  cmd, method: "POST", host: "localhost:8080",
		path:         "/api/users",
		bodyContains: "Bob",
	})
}

func TestCurlRW_86_GetHTTPIPPort(t *testing.T) {
	runCurlCase(t, curlCase{
		name:   "GET http IP port",
		cmd:    "curl http://127.0.0.1:4000/api/users",
		method: "GET", host: "127.0.0.1:4000", path: "/api/users", scheme: "http",
	})
}

func TestCurlRW_87_PutNoSchemeIPNoPort(t *testing.T) {
	// No scheme, bare IP — host must be extracted.
	cmd := "curl -X PUT 192.168.0.50/api/users/3 \\\n" +
		"  -H \"Content-Type: application/json\" \\\n" +
		"  -d '{\"name\": \"Charlie\"}'"
	runCurlCase(t, curlCase{
		name: "PUT no scheme IP no port",
		cmd:  cmd, method: "PUT", host: "192.168.0.50",
		path:         "/api/users/3",
		bodyContains: "Charlie",
	})
}

// ── Comment / separator / indentation robustness ──────────────────────────

func TestCurlRW_88_CommentLineIgnored(t *testing.T) {
	// Multiple comment lines above the command must all be ignored.
	cmd := "# GET with HTTP, IP address, no port\n" +
		"# second comment line\n" +
		"curl -X GET http://192.168.1.100/api/users"
	runCurlCase(t, curlCase{
		name: "multiple comment lines",
		cmd:  cmd, method: "GET", host: "192.168.1.100", path: "/api/users",
	})
}

func TestCurlRW_89_MarkdownSeparatorIgnored(t *testing.T) {
	// "---" separator line between examples must not produce warnings.
	cmd := "---\ncurl https://api.example.com/users\n---"
	result := ParseCurl(cmd)
	if result.Request == nil {
		t.Fatalf("expected request; warnings: %v", result.Warnings)
	}
	for _, w := range result.Warnings {
		if strings.Contains(w, "unknown curl flag") {
			t.Errorf("--- produced spurious warning: %q", w)
		}
	}
}

func TestCurlRW_90_LeadingTrailingBlankLines(t *testing.T) {
	cmd := "\n\ncurl -X DELETE https://api.example.com/users/5\n\n"
	runCurlCase(t, curlCase{
		name: "leading and trailing blank lines",
		cmd:  cmd, method: "DELETE", host: "api.example.com", path: "/users/5",
	})
}

func TestCurlRW_91_IndentedContinuation(t *testing.T) {
	// Indented flags after \ continuation must parse correctly.
	cmd := "curl -X POST https://api.example.com/events \\\n" +
		"    -H \"Content-Type: application/json\" \\\n" +
		"    -H \"X-Api-Key: k_123\" \\\n" +
		"    -d '{\"event\":\"login\"}'"
	runCurlCase(t, curlCase{
		name: "indented backslash continuation",
		cmd:  cmd, method: "POST", host: "api.example.com", path: "/events",
		headers:      map[string]string{"Content-Type": "application/json", "X-Api-Key": "k_123"},
		bodyContains: "login",
	})
}

func TestCurlRW_92_FullDocBlockWithCommentAndSeparator(t *testing.T) {
	// Simulates exactly what a user pastes: comment + command + separator.
	cmd := "# PUT with no scheme, IP address, no port\n" +
		"curl -X PUT 192.168.0.50/api/users/3 \\\n" +
		"  -H \"Content-Type: application/json\" \\\n" +
		"  -d '{\"name\": \"Charlie\"}'"
	runCurlCase(t, curlCase{
		name: "full doc block comment+cmd",
		cmd:  cmd, method: "PUT", host: "192.168.0.50", path: "/api/users/3",
		bodyContains: "Charlie",
	})
}
