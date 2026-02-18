package http

import (
	"bytes"
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
