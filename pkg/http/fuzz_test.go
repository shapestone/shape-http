package http

import (
	"bytes"
	"strings"
	"testing"
)

// Seed corpora for requests and responses used across multiple fuzz targets.

var requestSeeds = [][]byte{
	[]byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"),
	[]byte("POST /api/users HTTP/1.1\r\nHost: api.example.com\r\nContent-Type: application/json\r\nContent-Length: 15\r\n\r\n{\"name\":\"alice\"}"),
	[]byte("PUT /resource/1 HTTP/1.1\r\nHost: example.com\r\nAuthorization: Bearer token123\r\nContent-Length: 4\r\n\r\ndata"),
	[]byte("DELETE /item/42 HTTP/1.1\r\nHost: example.com\r\n\r\n"),
	[]byte("HEAD /status HTTP/1.1\r\nHost: example.com\r\n\r\n"),
	[]byte("OPTIONS * HTTP/1.1\r\nHost: example.com\r\n\r\n"),
	[]byte("GET /path?q=hello+world&page=2 HTTP/1.1\r\nHost: example.com\r\nAccept: text/html,application/json\r\nAccept-Encoding: gzip, deflate\r\nConnection: keep-alive\r\n\r\n"),
	[]byte("POST /upload HTTP/1.1\r\nHost: example.com\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nhello\r\n6\r\nworld!\r\n0\r\n\r\n"),
	// Edge cases
	[]byte("GET / HTTP/1.0\r\n\r\n"),
	[]byte("GET / HTTP/1.1\r\nHost: example.com\r\nX-Empty:\r\n\r\n"),
	[]byte("GET / HTTP/1.1\r\nHost: example.com\r\nCookie: a=1; b=2; c=3\r\n\r\n"),
	[]byte("POST / HTTP/1.1\r\nHost: example.com\r\nContent-Length: 0\r\n\r\n"),
}

var responseSeeds = [][]byte{
	[]byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 5\r\n\r\nhello"),
	[]byte("HTTP/1.1 404 Not Found\r\nContent-Type: application/json\r\nContent-Length: 14\r\n\r\n{\"error\":\"gone\"}"),
	[]byte("HTTP/1.1 204 No Content\r\n\r\n"),
	[]byte("HTTP/1.1 301 Moved Permanently\r\nLocation: https://example.com/\r\nContent-Length: 0\r\n\r\n"),
	[]byte("HTTP/1.1 500 Internal Server Error\r\nContent-Type: text/plain\r\nContent-Length: 5\r\n\r\noops!"),
	[]byte("HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nhello\r\n6\r\nworld!\r\n0\r\n\r\n"),
	[]byte("HTTP/1.1 200 OK\r\nSet-Cookie: session=abc123; Path=/; HttpOnly\r\nContent-Length: 2\r\n\r\nok"),
	[]byte("HTTP/1.1 100 Continue\r\n\r\n"),
	// Edge cases
	[]byte("HTTP/1.0 200 OK\r\nContent-Length: 0\r\n\r\n"),
	[]byte("HTTP/1.1 200 OK\r\n\r\n"),
	[]byte("HTTP/1.1 200 OK\r\nContent-Type: text/html\r\nX-Custom-Header: value with spaces\r\nContent-Length: 6\r\n\r\n<html>"),
}

// FuzzUnmarshalRequest fuzzes the request parser.
// The invariant: never panic regardless of input.
func FuzzUnmarshalRequest(f *testing.F) {
	for _, seed := range requestSeeds {
		f.Add(seed)
	}
	// Pathological inputs
	f.Add([]byte(""))
	f.Add([]byte("\r\n\r\n"))
	f.Add([]byte("GET"))
	f.Add([]byte("GET / HTTP/1.1"))
	f.Add([]byte("GET / HTTP/1.1\r\n"))
	f.Add([]byte("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"))
	f.Add(bytes.Repeat([]byte("X-Header: value\r\n"), 100))

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("UnmarshalRequest panicked on input %q: %v", data, r)
			}
		}()
		_, _ = UnmarshalRequest(data)
	})
}

// FuzzUnmarshalResponse fuzzes the response parser.
// The invariant: never panic regardless of input.
func FuzzUnmarshalResponse(f *testing.F) {
	for _, seed := range responseSeeds {
		f.Add(seed)
	}
	// Pathological inputs
	f.Add([]byte(""))
	f.Add([]byte("HTTP/1.1"))
	f.Add([]byte("HTTP/1.1 200"))
	f.Add([]byte("HTTP/1.1 200 OK\r\n"))
	f.Add([]byte("HTTP/1.1 99999 Status\r\n\r\n"))
	f.Add([]byte("HTTP/1.1 -1 Bad\r\n\r\n"))
	f.Add([]byte("HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\nFFFFFFFF\r\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("UnmarshalResponse panicked on input %q: %v", data, r)
			}
		}()
		_, _ = UnmarshalResponse(data)
	})
}

// FuzzUnmarshal fuzzes the auto-detecting Unmarshal function.
func FuzzUnmarshal(f *testing.F) {
	for _, seed := range requestSeeds {
		f.Add(seed)
	}
	for _, seed := range responseSeeds {
		f.Add(seed)
	}
	f.Add([]byte(""))
	f.Add([]byte("\x00\x01\x02\x03"))

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Unmarshal panicked on input %q: %v", data, r)
			}
		}()

		if bytes.HasPrefix(data, []byte("HTTP/")) {
			var resp Response
			_ = Unmarshal(data, &resp)
		} else {
			var req Request
			_ = Unmarshal(data, &req)
		}
	})
}

// FuzzLenient fuzzes the lenient parser which must never return an error.
// Invariants: never panic, never return nil result.
func FuzzLenient(f *testing.F) {
	for _, seed := range requestSeeds {
		f.Add(seed)
	}
	for _, seed := range responseSeeds {
		f.Add(seed)
	}
	f.Add([]byte(""))
	f.Add([]byte("\r\n"))
	f.Add([]byte("not http at all"))
	f.Add([]byte("GET\r\n\r\n"))
	f.Add([]byte("HTTP/1.1 \r\n\r\n"))
	f.Add(bytes.Repeat([]byte("\r\n"), 50))
	f.Add([]byte("GET / HTTP/1.1\nHost: example.com\n\n")) // LF-only endings

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("UnmarshalLenient panicked on input %q: %v", data, r)
			}
		}()

		result := UnmarshalLenient(data)
		if result == nil {
			t.Error("UnmarshalLenient returned nil result")
		}
	})
}

// FuzzMarshalRequest fuzzes that Marshal never panics on a *Request.
func FuzzMarshalRequest(f *testing.F) {
	f.Add("GET", "/", "HTTP/1.1", "Host", "example.com", []byte(nil))
	f.Add("POST", "/api", "HTTP/1.1", "Content-Type", "application/json", []byte(`{"x":1}`))
	f.Add("", "", "", "", "", []byte(nil))
	f.Add("GET", "/", "", "", "", []byte(nil))
	f.Add("CUSTOM-METHOD", "/path with spaces", "HTTP/1.1", "X-Key", "val", []byte("body"))

	f.Fuzz(func(t *testing.T, method, path, version, headerKey, headerVal string, body []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Marshal(*Request) panicked: %v", r)
			}
		}()

		req := &Request{
			Method:  method,
			Path:    path,
			Version: version,
			Body:    body,
		}
		if headerKey != "" {
			req.Headers = Headers{{Key: headerKey, Value: headerVal}}
		}
		_, _ = Marshal(req)
	})
}

// FuzzMarshalResponse fuzzes that Marshal never panics on a *Response.
func FuzzMarshalResponse(f *testing.F) {
	f.Add("HTTP/1.1", 200, "OK", "Content-Type", "text/plain", []byte("hello"))
	f.Add("HTTP/1.1", 404, "Not Found", "", "", []byte(nil))
	f.Add("", 0, "", "", "", []byte(nil))
	f.Add("HTTP/1.1", -1, "", "", "", []byte(nil))
	f.Add("HTTP/1.1", 99999, "Unknown", "X-Key", "val", []byte("body"))

	f.Fuzz(func(t *testing.T, version string, statusCode int, reason, headerKey, headerVal string, body []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Marshal(*Response) panicked: %v", r)
			}
		}()

		resp := &Response{
			Version:    version,
			StatusCode: statusCode,
			Reason:     reason,
			Body:       body,
		}
		if headerKey != "" {
			resp.Headers = Headers{{Key: headerKey, Value: headerVal}}
		}
		_, _ = Marshal(resp)
	})
}

// FuzzRoundTripRequest verifies that a successfully parsed request can be
// marshaled and re-parsed to produce the same result.
func FuzzRoundTripRequest(f *testing.F) {
	for _, seed := range requestSeeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("round-trip panicked on input %q: %v", data, r)
			}
		}()

		req1, err := UnmarshalRequest(data)
		if err != nil {
			return // invalid input, skip
		}

		wire, err := Marshal(req1)
		if err != nil {
			t.Errorf("Marshal failed after successful UnmarshalRequest: %v", err)
			return
		}

		req2, err := UnmarshalRequest(wire)
		if err != nil {
			t.Errorf("UnmarshalRequest failed on re-serialized data: %v\noriginal wire: %q\nre-serialized: %q", err, data, wire)
			return
		}

		if req1.Method != req2.Method {
			t.Errorf("Method mismatch: %q != %q", req1.Method, req2.Method)
		}
		if req1.Path != req2.Path {
			t.Errorf("Path mismatch: %q != %q", req1.Path, req2.Path)
		}
		if len(req1.Headers) != len(req2.Headers) {
			t.Errorf("Header count mismatch: %d != %d", len(req1.Headers), len(req2.Headers))
		}
	})
}

// FuzzRoundTripResponse verifies that a successfully parsed response can be
// marshaled and re-parsed to produce the same result.
func FuzzRoundTripResponse(f *testing.F) {
	for _, seed := range responseSeeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("round-trip panicked on input %q: %v", data, r)
			}
		}()

		resp1, err := UnmarshalResponse(data)
		if err != nil {
			return // invalid input, skip
		}

		wire, err := Marshal(resp1)
		if err != nil {
			t.Errorf("Marshal failed after successful UnmarshalResponse: %v", err)
			return
		}

		resp2, err := UnmarshalResponse(wire)
		if err != nil {
			t.Errorf("UnmarshalResponse failed on re-serialized data: %v\noriginal wire: %q\nre-serialized: %q", err, data, wire)
			return
		}

		if resp1.StatusCode != resp2.StatusCode {
			t.Errorf("StatusCode mismatch: %d != %d", resp1.StatusCode, resp2.StatusCode)
		}
		if resp1.Reason != resp2.Reason {
			t.Errorf("Reason mismatch: %q != %q", resp1.Reason, resp2.Reason)
		}
		if len(resp1.Headers) != len(resp2.Headers) {
			t.Errorf("Header count mismatch: %d != %d", len(resp1.Headers), len(resp2.Headers))
		}
	})
}

// FuzzParse fuzzes the AST-based Parse path.
func FuzzParse(f *testing.F) {
	for _, seed := range requestSeeds {
		f.Add(string(seed))
	}
	for _, seed := range responseSeeds {
		f.Add(string(seed))
	}
	f.Add("")
	f.Add("not http")
	f.Add("\x00\x01\x02")

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Parse panicked on input %q: %v", input, r)
			}
		}()
		_, _ = Parse(input)
	})
}

// FuzzParseCurl fuzzes the curl command parser.
//
// Invariants:
//   - ParseCurl never panics on any input
//   - ParseCurl never returns nil
//   - When result.Partial is false, result.Request must be non-nil
func FuzzParseCurl(f *testing.F) {
	// Seed corpus: valid curl commands covering all flag types
	f.Add(`curl https://api.example.com/users`)
	f.Add(`curl -X POST https://api.example.com/users -H "Content-Type: application/json" -d '{"name":"Alice"}'`)
	f.Add(`curl -X PUT https://api.example.com/users/1 -H "Authorization: Bearer tok" -d '{"active":false}'`)
	f.Add(`curl -X DELETE https://api.example.com/users/42`)
	f.Add(`curl -X PATCH https://api.example.com/users/7 -d '{"email":"new@x.com"}'`)
	f.Add(`curl -I https://api.example.com/health`)
	f.Add(`curl -u admin:secret https://api.example.com/admin`)
	f.Add(`curl -H "Authorization: Bearer eyJ.payload.sig" https://api.example.com/me`)
	f.Add(`curl -v -s -k -L --compressed https://api.example.com/`)
	f.Add(`curl --http2 https://api.example.com/h2`)
	f.Add(`curl --http3 https://api.example.com/h3`)
	f.Add(`curl -F "name=Alice" -F "role=admin" https://api.example.com/profile`)
	f.Add(`curl --data-urlencode "q=hello world" https://api.example.com/search`)
	f.Add(`curl -X POST https://api.example.com/form -d 'a=1' -d 'b=2'`)
	f.Add(`curl -o /tmp/out.json https://api.example.com/export`)
	f.Add(`curl -A "TestAgent/1.0" https://api.example.com/`)
	f.Add("curl -X POST \\\n  https://api.example.com/users \\\n  -H \"Content-Type: application/json\" \\\n  -d '{\"name\":\"Bob\"}'")
	f.Add(`curl --data-raw '{"query":"{ me { id } }"}' https://api.example.com/graphql`)
	f.Add(`curl --data-binary '{"b":true}' https://api.example.com/raw`)
	f.Add(`https://api.example.com/no-curl-prefix`)
	f.Add(`curl http://localhost:3000/api/health`)
	f.Add(`curl http://127.0.0.1:8080/v1/data`)
	f.Add(`curl -X POST https://api.example.com/ -H "Content-Type: application/x-www-form-urlencoded" -d 'grant_type=cc&client_id=x'`)
	// Edge / pathological inputs
	f.Add(``)
	f.Add(`curl`)
	f.Add(`curl -X`)
	f.Add(`curl -H`)
	f.Add(`curl -d`)
	f.Add(`curl -X POST`)
	f.Add(`curl -u`)
	f.Add(`curl -F`)
	f.Add(`curl --unknown-flag https://x.com/`)
	f.Add(`curl -H "bad header no colon" https://x.com/`)
	f.Add(`curl -d @/etc/passwd https://x.com/`)
	f.Add(`curl -F "file=@/tmp/big.bin" https://x.com/`)
	f.Add(`curl "https://x.com/path with spaces"`)
	f.Add("curl -H \"unclosed")
	f.Add(`curl 'unclosed`)
	f.Add(`curl -X POST -X GET https://x.com/`) // conflicting -X flags
	f.Add(`curl ` + strings.Repeat("-H \"X-H: v\" ", 50) + `https://x.com/`)
	f.Add(`curl https://x.com/` + strings.Repeat("a", 4096))
	f.Add(`curl -d '` + strings.Repeat("x", 65536) + `' https://x.com/`)
	f.Add("curl \x00 https://x.com/")
	f.Add("curl \xff\xfe https://x.com/")
	f.Add(`curl --http2 --http3 https://x.com/`) // conflicting version flags

	f.Fuzz(func(t *testing.T, cmd string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ParseCurl panicked on input %q: %v", cmd, r)
			}
		}()

		result := ParseCurl(cmd)

		// Invariant: result is never nil
		if result == nil {
			t.Errorf("ParseCurl(%q) returned nil", cmd)
			return
		}

		// Invariant: if not partial, Request must be set
		if !result.Partial && result.Request == nil {
			t.Errorf("ParseCurl(%q): Partial=false but Request=nil; warnings=%v", cmd, result.Warnings)
		}

		// Invariant: if Request is set, method must be non-empty
		if result.Request != nil && result.Request.Method == "" {
			t.Errorf("ParseCurl(%q): Request.Method is empty; warnings=%v", cmd, result.Warnings)
		}

		// Invariant: if Request is set, path must start with /
		if result.Request != nil && result.Request.Path != "" && result.Request.Path[0] != '/' {
			t.Errorf("ParseCurl(%q): Request.Path %q does not start with /", cmd, result.Request.Path)
		}
	})
}
