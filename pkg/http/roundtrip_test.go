package http

import (
	"testing"
)

// TestRoundTrip_Request tests that Marshal(Unmarshal(data)) == data for requests.
func TestRoundTrip_Request(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "simple GET",
			data: "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n",
		},
		{
			name: "POST with body",
			data: "POST /api/users HTTP/1.1\r\nHost: example.com\r\nContent-Type: application/json\r\nContent-Length: 19\r\n\r\n{\"name\":\"John Doe\"}",
		},
		{
			name: "multiple headers",
			data: "GET /search?q=test HTTP/1.1\r\nHost: example.com\r\nAccept: text/html\r\nAccept-Language: en-US\r\nUser-Agent: shape-http/1.0\r\n\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := UnmarshalRequest([]byte(tt.data))
			if err != nil {
				t.Fatalf("UnmarshalRequest() error = %v", err)
			}

			out, err := Marshal(req)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			if string(out) != tt.data {
				t.Errorf("round-trip mismatch:\ngot:  %q\nwant: %q", string(out), tt.data)
			}
		})
	}
}

// TestRoundTrip_Response tests that Marshal(Unmarshal(data)) == data for responses.
func TestRoundTrip_Response(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "simple 200",
			data: "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\n",
		},
		{
			name: "200 with body",
			data: "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 13\r\n\r\nHello, World!",
		},
		{
			name: "404",
			data: "HTTP/1.1 404 Not Found\r\nContent-Type: text/html\r\nContent-Length: 18\r\n\r\n<h1>Not Found</h1>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := UnmarshalResponse([]byte(tt.data))
			if err != nil {
				t.Fatalf("UnmarshalResponse() error = %v", err)
			}

			out, err := Marshal(resp)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			if string(out) != tt.data {
				t.Errorf("round-trip mismatch:\ngot:  %q\nwant: %q", string(out), tt.data)
			}
		})
	}
}
