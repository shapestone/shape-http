package http

import (
	"testing"
)

func TestMarshal_Request_Simple(t *testing.T) {
	req := &Request{
		Method:  "GET",
		Path:    "/api/users",
		Version: "HTTP/1.1",
		Headers: Headers{
			{Key: "Host", Value: "example.com"},
			{Key: "Accept", Value: "application/json"},
		},
	}

	data, err := Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	want := "GET /api/users HTTP/1.1\r\nHost: example.com\r\nAccept: application/json\r\n\r\n"
	if string(data) != want {
		t.Errorf("Marshal() =\n%q\nwant:\n%q", string(data), want)
	}
}

func TestMarshal_Request_WithBody(t *testing.T) {
	req := &Request{
		Method:  "POST",
		Path:    "/api/users",
		Version: "HTTP/1.1",
		Headers: Headers{
			{Key: "Host", Value: "example.com"},
			{Key: "Content-Type", Value: "application/json"},
		},
		Body: []byte(`{"name":"John Doe"}`),
	}

	data, err := Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	want := "POST /api/users HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Content-Type: application/json\r\n" +
		"Content-Length: 19\r\n" +
		"\r\n" +
		`{"name":"John Doe"}`
	if string(data) != want {
		t.Errorf("Marshal() =\n%q\nwant:\n%q", string(data), want)
	}
}

func TestMarshal_Request_WithExplicitContentLength(t *testing.T) {
	req := &Request{
		Method:  "POST",
		Path:    "/api/users",
		Version: "HTTP/1.1",
		Headers: Headers{
			{Key: "Host", Value: "example.com"},
			{Key: "Content-Length", Value: "19"},
		},
		Body: []byte(`{"name":"John Doe"}`),
	}

	data, err := Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Should not add a second Content-Length
	want := "POST /api/users HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Content-Length: 19\r\n" +
		"\r\n" +
		`{"name":"John Doe"}`
	if string(data) != want {
		t.Errorf("Marshal() =\n%q\nwant:\n%q", string(data), want)
	}
}

func TestMarshal_Request_DefaultVersion(t *testing.T) {
	req := &Request{
		Method: "GET",
		Path:   "/",
		Headers: Headers{
			{Key: "Host", Value: "example.com"},
		},
	}

	data, err := Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	want := "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"
	if string(data) != want {
		t.Errorf("Marshal() =\n%q\nwant:\n%q", string(data), want)
	}
}

func TestMarshal_Request_EmptyMethod(t *testing.T) {
	req := &Request{Path: "/"}
	_, err := Marshal(req)
	if err == nil {
		t.Error("Marshal() expected error for empty method")
	}
}

func TestMarshal_Request_EmptyPath(t *testing.T) {
	req := &Request{Method: "GET"}
	_, err := Marshal(req)
	if err == nil {
		t.Error("Marshal() expected error for empty path")
	}
}

func TestMarshal_Response_Simple(t *testing.T) {
	resp := &Response{
		Version:    "HTTP/1.1",
		StatusCode: 200,
		Reason:     "OK",
		Headers: Headers{
			{Key: "Content-Type", Value: "text/plain"},
		},
	}

	data, err := Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	want := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\n"
	if string(data) != want {
		t.Errorf("Marshal() =\n%q\nwant:\n%q", string(data), want)
	}
}

func TestMarshal_Response_WithBody(t *testing.T) {
	resp := &Response{
		Version:    "HTTP/1.1",
		StatusCode: 200,
		Reason:     "OK",
		Headers: Headers{
			{Key: "Content-Type", Value: "text/plain"},
		},
		Body: []byte("Hello, World!"),
	}

	data, err := Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	want := "HTTP/1.1 200 OK\r\n" +
		"Content-Type: text/plain\r\n" +
		"Content-Length: 13\r\n" +
		"\r\n" +
		"Hello, World!"
	if string(data) != want {
		t.Errorf("Marshal() =\n%q\nwant:\n%q", string(data), want)
	}
}

func TestMarshal_Response_404(t *testing.T) {
	resp := &Response{
		Version:    "HTTP/1.1",
		StatusCode: 404,
		Reason:     "Not Found",
		Headers: Headers{
			{Key: "Content-Type", Value: "text/html"},
		},
		Body: []byte("<h1>Not Found</h1>"),
	}

	data, err := Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	want := "HTTP/1.1 404 Not Found\r\n" +
		"Content-Type: text/html\r\n" +
		"Content-Length: 18\r\n" +
		"\r\n" +
		"<h1>Not Found</h1>"
	if string(data) != want {
		t.Errorf("Marshal() =\n%q\nwant:\n%q", string(data), want)
	}
}

func TestMarshal_Response_Chunked_NoAutoContentLength(t *testing.T) {
	resp := &Response{
		Version:    "HTTP/1.1",
		StatusCode: 200,
		Reason:     "OK",
		Headers: Headers{
			{Key: "Transfer-Encoding", Value: "chunked"},
		},
		Body: []byte("Hello"),
	}

	data, err := Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Should NOT add Content-Length when Transfer-Encoding: chunked
	want := "HTTP/1.1 200 OK\r\n" +
		"Transfer-Encoding: chunked\r\n" +
		"\r\n" +
		"Hello"
	if string(data) != want {
		t.Errorf("Marshal() =\n%q\nwant:\n%q", string(data), want)
	}
}

func TestMarshal_Nil(t *testing.T) {
	_, err := Marshal(nil)
	if err == nil {
		t.Error("Marshal(nil) expected error")
	}
}

func TestMarshal_UnsupportedType(t *testing.T) {
	_, err := Marshal("not a request or response")
	if err == nil {
		t.Error("Marshal(string) expected error")
	}
}

func TestMarshal_Marshaler_Interface(t *testing.T) {
	m := &mockMarshaler{data: []byte("custom output")}
	data, err := Marshal(m)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(data) != "custom output" {
		t.Errorf("Marshal() = %q, want %q", string(data), "custom output")
	}
}

type mockMarshaler struct {
	data []byte
}

func (m *mockMarshaler) MarshalHTTP() ([]byte, error) {
	return m.data, nil
}
