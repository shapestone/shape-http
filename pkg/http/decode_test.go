package http

import (
	"bytes"
	"fmt"
	"testing"
)

func TestDecoder_Request(t *testing.T) {
	data := "GET /api HTTP/1.1\r\nHost: example.com\r\nContent-Length: 0\r\n\r\n"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	req := &Request{}
	err := dec.Decode(req)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if req.Method != "GET" {
		t.Errorf("Method = %q, want GET", req.Method)
	}
	if req.Path != "/api" {
		t.Errorf("Path = %q, want /api", req.Path)
	}
	if req.Version != "HTTP/1.1" {
		t.Errorf("Version = %q, want HTTP/1.1", req.Version)
	}
}

func TestDecoder_RequestWithBody(t *testing.T) {
	data := "POST /api HTTP/1.1\r\nHost: example.com\r\nContent-Length: 11\r\n\r\nhello world"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	req, err := dec.DecodeRequest()
	if err != nil {
		t.Fatalf("DecodeRequest() error = %v", err)
	}

	if string(req.Body) != "hello world" {
		t.Errorf("Body = %q, want hello world", string(req.Body))
	}
}

func TestDecoder_Response(t *testing.T) {
	data := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 5\r\n\r\nHello"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	resp := &Response{}
	err := dec.Decode(resp)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if resp.Version != "HTTP/1.1" {
		t.Errorf("Version = %q, want HTTP/1.1", resp.Version)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if resp.Reason != "OK" {
		t.Errorf("Reason = %q, want OK", resp.Reason)
	}
	if string(resp.Body) != "Hello" {
		t.Errorf("Body = %q, want Hello", string(resp.Body))
	}
}

func TestDecoder_ResponseWithChunkedBody(t *testing.T) {
	data := "HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nHello\r\n7\r\n, World\r\n0\r\n\r\n"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	resp, err := dec.DecodeResponse()
	if err != nil {
		t.Fatalf("DecodeResponse() error = %v", err)
	}

	if string(resp.Body) != "Hello, World" {
		t.Errorf("Body = %q, want Hello, World", string(resp.Body))
	}
}

func TestDecoder_TypeMismatch(t *testing.T) {
	data := "HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	req := &Request{}
	err := dec.Decode(req)
	if err == nil {
		t.Error("expected error for type mismatch")
	}
}

func TestDecoder_DecodeRequest_Convenience(t *testing.T) {
	data := "GET / HTTP/1.1\r\nHost: example.com\r\nContent-Length: 0\r\n\r\n"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	req, err := dec.DecodeRequest()
	if err != nil {
		t.Fatalf("DecodeRequest() error = %v", err)
	}

	if req.Method != "GET" {
		t.Errorf("Method = %q, want GET", req.Method)
	}
}

func TestDecoder_DecodeResponse_Convenience(t *testing.T) {
	data := "HTTP/1.1 404 Not Found\r\nContent-Length: 9\r\n\r\nNot Found"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	resp, err := dec.DecodeResponse()
	if err != nil {
		t.Fatalf("DecodeResponse() error = %v", err)
	}

	if resp.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", resp.StatusCode)
	}
	if string(resp.Body) != "Not Found" {
		t.Errorf("Body = %q, want 'Not Found'", string(resp.Body))
	}
}

func TestDecoder_UnsupportedType(t *testing.T) {
	data := "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	err := dec.Decode("not a request or response")
	if err == nil {
		t.Error("Decode() = nil, want error for unsupported type")
	}
}

func TestDecoder_ResponseTargetWithRequestData(t *testing.T) {
	// Request data decoded into *Response target should fail
	data := "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	resp := &Response{}
	err := dec.Decode(resp)
	if err == nil {
		t.Error("Decode() = nil, want error when decoding request into *Response")
	}
}

func TestDecoder_EmptyReader(t *testing.T) {
	dec := NewDecoder(bytes.NewReader([]byte{}))
	err := dec.Decode(&Request{})
	if err == nil {
		t.Error("Decode() = nil, want error for empty reader")
	}
}

func TestDecoder_MalformedRequestLine(t *testing.T) {
	// Request line without two spaces (parts < 3)
	data := "BADREQUEST\r\nHost: example.com\r\n\r\n"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	_, err := dec.DecodeRequest()
	if err == nil {
		t.Error("DecodeRequest() = nil, want error for malformed request line")
	}
}

func TestDecoder_MalformedStatusLine(t *testing.T) {
	// Status line without a space (parts < 2)
	data := "JUSTONEWORD\r\nContent-Length: 0\r\n\r\n"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	_, err := dec.DecodeResponse()
	if err == nil {
		t.Error("DecodeResponse() = nil, want error for malformed status line")
	}
}

func TestDecoder_MalformedHeader(t *testing.T) {
	// Header line without colon
	data := "GET / HTTP/1.1\r\nMalformedHeaderLine\r\n\r\n"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	_, err := dec.DecodeRequest()
	if err == nil {
		t.Error("DecodeRequest() = nil, want error for malformed header")
	}
}

func TestDecoder_ResponseWithNoBody(t *testing.T) {
	// Response with Content-Length: 0 should have nil body
	data := "HTTP/1.1 204 No Content\r\nContent-Length: 0\r\n\r\n"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	resp, err := dec.DecodeResponse()
	if err != nil {
		t.Fatalf("DecodeResponse() error = %v", err)
	}
	if resp.StatusCode != 204 {
		t.Errorf("StatusCode = %d, want 204", resp.StatusCode)
	}
	if resp.Body != nil {
		t.Errorf("Body = %v, want nil for Content-Length: 0", resp.Body)
	}
}

func TestDecoder_RequestChunkedBody(t *testing.T) {
	data := "POST / HTTP/1.1\r\nHost: example.com\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nhello\r\n0\r\n\r\n"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	req, err := dec.DecodeRequest()
	if err != nil {
		t.Fatalf("DecodeRequest() error = %v", err)
	}
	if string(req.Body) != "hello" {
		t.Errorf("Body = %q, want hello", string(req.Body))
	}
}

func TestDecoder_ChunkedBodyTruncated(t *testing.T) {
	// Chunked body that is truncated mid-stream
	data := "HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nhel"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	_, err := dec.DecodeResponse()
	if err == nil {
		t.Error("DecodeResponse() = nil, want error for truncated chunked body")
	}
}

func TestDecoder_ResponseStatusOnly(t *testing.T) {
	// Status line with version and code but no reason phrase
	data := "HTTP/1.1 201\r\nContent-Length: 0\r\n\r\n"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	resp, err := dec.DecodeResponse()
	if err != nil {
		t.Fatalf("DecodeResponse() error = %v", err)
	}
	if resp.StatusCode != 201 {
		t.Errorf("StatusCode = %d, want 201", resp.StatusCode)
	}
}

func TestDecoder_RequestWithLargeBody(t *testing.T) {
	body := bytes.Repeat([]byte("x"), 1024)
	data := "POST / HTTP/1.1\r\nContent-Length: 1024\r\n\r\n" + string(body)
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	req, err := dec.DecodeRequest()
	if err != nil {
		t.Fatalf("DecodeRequest() error = %v", err)
	}
	if len(req.Body) != 1024 {
		t.Errorf("Body length = %d, want 1024", len(req.Body))
	}
}

func TestDecoder_InvalidStatusCode(t *testing.T) {
	// Status code "abc" is not numeric — covers invalid status code error in decodeResponse
	data := "HTTP/1.1 abc OK\r\n\r\n"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	_, err := dec.DecodeResponse()
	if err == nil {
		t.Error("DecodeResponse() = nil, want error for non-numeric status code")
	}
}

func TestDecoder_ResponseBodyTruncated(t *testing.T) {
	// Content-Length: 100 but only a short body — io.ReadFull fails
	data := "HTTP/1.1 200 OK\r\nContent-Length: 100\r\n\r\nshort"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	_, err := dec.DecodeResponse()
	if err == nil {
		t.Error("DecodeResponse() = nil, want error for truncated body")
	}
}

func TestDecoder_ChunkedWithExtension(t *testing.T) {
	// Chunk size line with extension — exercises the sizeLine[:idx] path
	data := "HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n5;ext=foo\r\nhello\r\n0\r\n\r\n"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	resp, err := dec.DecodeResponse()
	if err != nil {
		t.Fatalf("DecodeResponse() error = %v", err)
	}
	if string(resp.Body) != "hello" {
		t.Errorf("Body = %q, want hello", string(resp.Body))
	}
}

func TestDecoder_ChunkedInvalidSize(t *testing.T) {
	// Invalid hex chunk size — covers invalid chunk size error in readChunkedBody
	data := "HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\nZZZZ\r\nhello\r\n0\r\n\r\n"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	_, err := dec.DecodeResponse()
	if err == nil {
		t.Error("DecodeResponse() = nil, want error for invalid chunk size")
	}
}

func TestDecoder_ChunkedEmptyBody(t *testing.T) {
	// Chunked body with only the terminating chunk (0\r\n\r\n) — len(result)==0 → nil body
	data := "HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n0\r\n\r\n"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	resp, err := dec.DecodeResponse()
	if err != nil {
		t.Fatalf("DecodeResponse() error = %v", err)
	}
	if resp.Body != nil {
		t.Errorf("Body = %v, want nil for empty chunked body", resp.Body)
	}
}

func TestDecoder_NoBodyNoContentLength(t *testing.T) {
	// No Content-Length, not chunked — returns nil body (last return nil, nil in readBody)
	data := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\n"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	resp, err := dec.DecodeResponse()
	if err != nil {
		t.Fatalf("DecodeResponse() error = %v", err)
	}
	if resp.Body != nil {
		t.Errorf("Body = %v, want nil when no Content-Length", resp.Body)
	}
}

func TestDecoder_ResponseNoHeaders(t *testing.T) {
	// Status line only with no headers and no CRLF terminator —
	// readHeaders readLine gets EOF with no data → error propagates back.
	data := "HTTP/1.1 200 OK\r\n"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	_, err := dec.DecodeResponse()
	if err == nil {
		t.Error("DecodeResponse() = nil, want error when status line has no headers")
	}
}

func TestDecoder_RequestNoHeaders(t *testing.T) {
	// Request line only with no headers and no CRLF terminator —
	// readHeaders readLine gets EOF → error propagates back.
	data := "GET / HTTP/1.1\r\n"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	_, err := dec.DecodeRequest()
	if err == nil {
		t.Error("DecodeRequest() = nil, want error when request line has no headers")
	}
}

func TestDecoder_ChunkedEOFBeforeData(t *testing.T) {
	// Transfer-Encoding: chunked but no body follows the blank line —
	// readChunkedBody readLine gets EOF → error
	data := "HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	_, err := dec.DecodeResponse()
	if err == nil {
		t.Error("DecodeResponse() = nil, want error when chunked body is missing")
	}
}

// immediateErrorReader always returns an error immediately on Read.
// Used to trigger the readLine error path in decodeRequest/decodeResponse.
type immediateErrorReader struct{}

func (r *immediateErrorReader) Read(_ []byte) (int, error) {
	return 0, errImmediateFail
}

// errImmediateFail is a sentinel error for immediateErrorReader.
var errImmediateFail = fmt.Errorf("immediate read failure")

func TestDecodeRequest_ReadLineError(t *testing.T) {
	// Calling decodeRequest directly with a reader that fails immediately exercises
	// the readLine → ("", err) → return fmt.Errorf("http: decode request: ...") path.
	dec := NewDecoder(&immediateErrorReader{})
	req := &Request{}
	err := dec.decodeRequest(req)
	if err == nil {
		t.Error("decodeRequest() = nil, want error when reader fails immediately")
	}
}

func TestDecoder_RequestBodyTruncated(t *testing.T) {
	// Content-Length: 100 but body is much shorter — io.ReadFull error in readBody
	// This covers the readBody error path in decodeRequest.
	data := "POST / HTTP/1.1\r\nContent-Length: 100\r\n\r\nshort"
	dec := NewDecoder(bytes.NewReader([]byte(data)))

	_, err := dec.DecodeRequest()
	if err == nil {
		t.Error("DecodeRequest() = nil, want error for truncated body")
	}
}

func TestDecodeResponse_ReadLineError(t *testing.T) {
	// Same as above but for decodeResponse.
	dec := NewDecoder(&immediateErrorReader{})
	resp := &Response{}
	err := dec.decodeResponse(resp)
	if err == nil {
		t.Error("decodeResponse() = nil, want error when reader fails immediately")
	}
}
