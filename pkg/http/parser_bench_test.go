package http

import (
	"testing"
)

var simpleRequest = []byte("GET /api/users HTTP/1.1\r\nHost: example.com\r\nAccept: application/json\r\nUser-Agent: shape-http/1.0\r\n\r\n")

var requestWithBody = []byte("POST /api/users HTTP/1.1\r\nHost: example.com\r\nContent-Type: application/json\r\nContent-Length: 55\r\n\r\n{\"name\":\"John Doe\",\"email\":\"john@example.com\",\"age\":30}")

var simpleResponse = []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: 25\r\nServer: shape-http/1.0\r\n\r\n{\"status\":\"ok\",\"count\":42}")

var chunkedResponse = []byte("HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nHello\r\n7\r\n, World\r\n1\r\n!\r\n0\r\n\r\n")

func BenchmarkUnmarshal_SimpleRequest(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := UnmarshalRequest(simpleRequest)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshal_RequestWithBody(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := UnmarshalRequest(requestWithBody)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshal_SimpleResponse(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := UnmarshalResponse(simpleResponse)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshal_ChunkedResponse(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := UnmarshalResponse(chunkedResponse)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParse_SimpleRequest(b *testing.B) {
	input := string(simpleRequest)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Parse(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRoundTrip_Request(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, err := UnmarshalRequest(requestWithBody)
		if err != nil {
			b.Fatal(err)
		}
		_, err = Marshal(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalLenient_SimpleRequest(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := UnmarshalLenient(simpleRequest)
		if result.Request == nil {
			b.Fatal("expected request")
		}
	}
}

func BenchmarkDetectMessageType(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectMessageType(simpleRequest)
		DetectMessageType(simpleResponse)
	}
}
