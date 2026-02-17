package http

import (
	"bufio"
	"bytes"
	nethttp "net/http"
	"strings"
	"testing"
)

// stdlib_bench_test.go — net/http comparison benchmarks
//
// These benchmarks compare shape-http against Go's standard library net/http
// for equivalent HTTP parsing and serialization operations.

// --- Marshal (Write) comparisons ---

func BenchmarkStdlib_WriteRequest_Simple(b *testing.B) {
	req, _ := nethttp.NewRequest("GET", "http://example.com/api/users", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "shape-http/1.0")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		req.Write(&buf)
	}
}

func BenchmarkStdlib_WriteRequest_WithBody(b *testing.B) {
	body := `{"name":"John Doe","email":"john@example.com","age":30}`
	req, _ := nethttp.NewRequest("POST", "http://example.com/api/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Save the body for reuse
	bodyBytes := []byte(body)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req.Body = nopReadCloser{bytes.NewReader(bodyBytes)}
		req.ContentLength = int64(len(bodyBytes))
		var buf bytes.Buffer
		req.Write(&buf)
	}
}

func BenchmarkStdlib_WriteResponse_Simple(b *testing.B) {
	body := `{"status":"ok","count":42}`
	resp := &nethttp.Response{
		Status:        "200 OK",
		StatusCode:    200,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        nethttp.Header{"Content-Type": {"application/json"}, "Server": {"shape-http/1.0"}},
		Body:          nopReadCloser{bytes.NewReader([]byte(body))},
		ContentLength: int64(len(body)),
	}
	bodyBytes := []byte(body)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp.Body = nopReadCloser{bytes.NewReader(bodyBytes)}
		var buf bytes.Buffer
		resp.Write(&buf)
	}
}

// --- Unmarshal (Read) comparisons ---

func BenchmarkStdlib_ReadRequest_Simple(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bufio.NewReader(bytes.NewReader(simpleRequest))
		req, err := nethttp.ReadRequest(r)
		if err != nil {
			b.Fatal(err)
		}
		req.Body.Close()
	}
}

func BenchmarkStdlib_ReadRequest_WithBody(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bufio.NewReader(bytes.NewReader(requestWithBody))
		req, err := nethttp.ReadRequest(r)
		if err != nil {
			b.Fatal(err)
		}
		req.Body.Close()
	}
}

func BenchmarkStdlib_ReadResponse_Simple(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bufio.NewReader(bytes.NewReader(simpleResponse))
		resp, err := nethttp.ReadResponse(r, nil)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func BenchmarkStdlib_ReadResponse_Chunked(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bufio.NewReader(bytes.NewReader(chunkedResponse))
		resp, err := nethttp.ReadResponse(r, nil)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// --- Round-trip comparison ---

func BenchmarkStdlib_RoundTrip_Request(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Read
		r := bufio.NewReader(bytes.NewReader(requestWithBody))
		req, err := nethttp.ReadRequest(r)
		if err != nil {
			b.Fatal(err)
		}
		req.Body.Close()

		// Write
		var buf bytes.Buffer
		req.Write(&buf)
	}
}

// nopReadCloser wraps a reader with a no-op Close method.
type nopReadCloser struct {
	*bytes.Reader
}

func (nopReadCloser) Close() error { return nil }
