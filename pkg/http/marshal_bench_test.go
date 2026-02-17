package http

import (
	"testing"
)

func BenchmarkMarshal_SimpleRequest(b *testing.B) {
	req := &Request{
		Method:  "GET",
		Path:    "/api/users",
		Version: "HTTP/1.1",
		Headers: Headers{
			{Key: "Host", Value: "example.com"},
			{Key: "Accept", Value: "application/json"},
			{Key: "User-Agent", Value: "shape-http/1.0"},
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Marshal(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshal_RequestWithBody(b *testing.B) {
	body := []byte(`{"name":"John Doe","email":"john@example.com","age":30}`)
	req := &Request{
		Method:  "POST",
		Path:    "/api/users",
		Version: "HTTP/1.1",
		Headers: Headers{
			{Key: "Host", Value: "example.com"},
			{Key: "Content-Type", Value: "application/json"},
		},
		Body: body,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Marshal(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshal_SimpleResponse(b *testing.B) {
	resp := &Response{
		Version:    "HTTP/1.1",
		StatusCode: 200,
		Reason:     "OK",
		Headers: Headers{
			{Key: "Content-Type", Value: "application/json"},
			{Key: "Content-Length", Value: "27"},
			{Key: "Server", Value: "shape-http/1.0"},
		},
		Body: []byte(`{"status":"ok","count":42}`),
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Marshal(resp)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshal_LargeHeaders(b *testing.B) {
	headers := make(Headers, 20)
	for i := 0; i < 20; i++ {
		headers[i] = Header{
			Key:   "X-Custom-Header-" + string(rune('A'+i)),
			Value: "some-value-that-is-reasonably-long-for-benchmarking",
		}
	}

	req := &Request{
		Method:  "GET",
		Path:    "/api/resource",
		Version: "HTTP/1.1",
		Headers: headers,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Marshal(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}
