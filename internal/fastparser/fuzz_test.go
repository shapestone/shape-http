package fastparser

import (
	"testing"
)

// FuzzParseRequest fuzzes the core fast-path request parser with arbitrary input.
// The invariant: never panic regardless of input.
func FuzzParseRequest(f *testing.F) {
	f.Add([]byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"))
	f.Add([]byte("POST /api HTTP/1.1\r\nHost: example.com\r\nContent-Length: 4\r\n\r\ndata"))
	f.Add([]byte("DELETE /resource/1 HTTP/1.1\r\nHost: example.com\r\nAuthorization: Bearer tok\r\n\r\n"))
	f.Add([]byte("GET /path?a=1&b=2 HTTP/1.1\r\nHost: example.com\r\nAccept: */*\r\n\r\n"))
	f.Add([]byte(""))
	f.Add([]byte("\r\n\r\n"))
	f.Add([]byte("GET"))
	f.Add([]byte("GET / HTTP/1.1\r\n"))
	f.Add([]byte("HTTP/1.1 200 OK\r\nX-A: \r\n\r\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ParseRequest panicked on input %q: %v", data, r)
			}
		}()

		p := NewParser(data)
		_, _ = p.ParseRequest()
	})
}

// FuzzParseResponse fuzzes the core fast-path response parser with arbitrary input.
func FuzzParseResponse(f *testing.F) {
	f.Add([]byte("HTTP/1.1 200 OK\r\nContent-Length: 5\r\n\r\nhello"))
	f.Add([]byte("HTTP/1.1 404 Not Found\r\nContent-Type: text/plain\r\nContent-Length: 9\r\n\r\nnot found"))
	f.Add([]byte("HTTP/1.1 204 No Content\r\n\r\n"))
	f.Add([]byte("HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nhello\r\n0\r\n\r\n"))
	f.Add([]byte(""))
	f.Add([]byte("HTTP/1.1 200\r\n\r\n"))
	f.Add([]byte("HTTP/1.1 abc OK\r\n\r\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ParseResponse panicked on input %q: %v", data, r)
			}
		}()

		p := NewParser(data)
		_, _ = p.ParseResponse()
	})
}

// FuzzUnmarshalRequest fuzzes the fast-path request unmarshaler directly.
func FuzzUnmarshalRequest(f *testing.F) {
	f.Add([]byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"))
	f.Add([]byte("POST /submit HTTP/1.1\r\nHost: example.com\r\nContent-Length: 5\r\n\r\nhello"))
	f.Add([]byte("GET / HTTP/1.1\r\nHost: example.com\r\nCookie: a=1; b=2\r\n\r\n"))
	f.Add([]byte("OPTIONS * HTTP/1.1\r\nHost: example.com\r\n\r\n"))
	f.Add([]byte("POST / HTTP/1.1\r\nHost: example.com\r\nTransfer-Encoding: chunked\r\n\r\n3\r\nfoo\r\n0\r\n\r\n"))
	// Malformed
	f.Add([]byte(""))
	f.Add([]byte(" "))
	f.Add([]byte("G\x00T / HTTP/1.1\r\n\r\n"))
	f.Add([]byte("GET / HTTP/1.1\r\nBad Header\r\n\r\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("UnmarshalRequest panicked on input %q: %v", data, r)
			}
		}()
		_, _ = UnmarshalRequest(data)
	})
}

// FuzzUnmarshalResponse fuzzes the fast-path response unmarshaler directly.
func FuzzUnmarshalResponse(f *testing.F) {
	f.Add([]byte("HTTP/1.1 200 OK\r\nContent-Length: 5\r\n\r\nhello"))
	f.Add([]byte("HTTP/1.1 301 Moved Permanently\r\nLocation: /new\r\nContent-Length: 0\r\n\r\n"))
	f.Add([]byte("HTTP/1.1 200 OK\r\nSet-Cookie: id=123; Path=/\r\nContent-Length: 2\r\n\r\nok"))
	f.Add([]byte("HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\na\r\n0123456789\r\n0\r\n\r\n"))
	// Malformed
	f.Add([]byte(""))
	f.Add([]byte("HTTP"))
	f.Add([]byte("HTTP/1.1\r\n\r\n"))
	f.Add([]byte("HTTP/1.1 200\r\n\r\n"))
	f.Add([]byte("HTTP/1.1 abc OK\r\n\r\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("UnmarshalResponse panicked on input %q: %v", data, r)
			}
		}()
		_, _ = UnmarshalResponse(data)
	})
}

// FuzzDechunk fuzzes the chunked transfer encoding decoder.
// This is a high-value target: hex integer parsing + arbitrary lengths.
func FuzzDechunk(f *testing.F) {
	// Valid chunked bodies (just the body after headers)
	f.Add([]byte("5\r\nhello\r\n0\r\n\r\n"))
	f.Add([]byte("a\r\n0123456789\r\n0\r\n\r\n"))
	f.Add([]byte("5\r\nhello\r\n6\r\nworld!\r\n0\r\n\r\n"))
	f.Add([]byte("0\r\n\r\n"))
	f.Add([]byte("1\r\nX\r\n0\r\n\r\n"))
	// Chunk with extension
	f.Add([]byte("5;ext=val\r\nhello\r\n0\r\n\r\n"))
	// LF-only endings
	f.Add([]byte("5\nhello\n0\n\n"))
	// Edge cases
	f.Add([]byte(""))
	f.Add([]byte("0\r\n"))
	f.Add([]byte("FFFFFFFF\r\n")) // huge chunk size
	f.Add([]byte("g\r\n"))        // invalid hex
	f.Add([]byte(";ext\r\n0\r\n\r\n"))
	f.Add([]byte("0000\r\n\r\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Dechunk panicked on input %q: %v", data, r)
			}
		}()
		_, _ = Dechunk(data)
	})
}

// FuzzLenientParser fuzzes the lenient parser which must never panic
// and must always return a non-nil result.
func FuzzLenientParser(f *testing.F) {
	f.Add([]byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"))
	f.Add([]byte("HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nok"))
	f.Add([]byte("GET / HTTP/1.1\nHost: example.com\n\n")) // LF only
	f.Add([]byte("get / http/1.1\r\n\r\n"))                // lowercase
	f.Add([]byte("GET\r\n\r\n"))                           // missing path and version
	f.Add([]byte("HTTP/1.1\r\n\r\n"))                      // missing status
	f.Add([]byte(""))
	f.Add([]byte("garbage"))
	f.Add([]byte("\x00\x01\x02\xff"))
	f.Add([]byte("GET / HTTP/1.1\r\nMalformed\r\n\r\n")) // header without colon

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("LenientParser panicked on input %q: %v", data, r)
			}
		}()

		lp := NewLenientParser(data)
		result := lp.Parse()
		if result == nil {
			t.Error("LenientParser.Parse() returned nil")
		}
	})
}
