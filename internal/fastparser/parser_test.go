package fastparser

import (
	"testing"
)

func TestParseRequest_Simple(t *testing.T) {
	data := []byte("GET /api/users HTTP/1.1\r\nHost: example.com\r\n\r\n")
	p := NewParser(data)
	req, err := p.ParseRequest()
	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}

	if req.Method != "GET" {
		t.Errorf("Method = %q, want GET", req.Method)
	}
	if req.Path != "/api/users" {
		t.Errorf("Path = %q, want /api/users", req.Path)
	}
	if req.Version != "HTTP/1.1" {
		t.Errorf("Version = %q, want HTTP/1.1", req.Version)
	}
	if len(req.Headers) != 1 {
		t.Fatalf("Headers count = %d, want 1", len(req.Headers))
	}
}

func TestParseResponse_Simple(t *testing.T) {
	data := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 5\r\n\r\nHello")
	p := NewParser(data)
	resp, err := p.ParseResponse()
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
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

func TestParseRequest_WithQueryString(t *testing.T) {
	data := []byte("GET /search?q=hello&page=1 HTTP/1.1\r\nHost: example.com\r\n\r\n")
	p := NewParser(data)
	req, err := p.ParseRequest()
	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}

	if req.Path != "/search?q=hello&page=1" {
		t.Errorf("Path = %q, want /search?q=hello&page=1", req.Path)
	}
}

func TestParseRequest_MalformedRequestLine(t *testing.T) {
	data := []byte("GETHTTP/1.1\r\n\r\n")
	p := NewParser(data)
	_, err := p.ParseRequest()
	if err == nil {
		t.Error("expected error for malformed request line")
	}
}

func TestParseResponse_MalformedStatusLine(t *testing.T) {
	data := []byte("HTTP/1.1 abc OK\r\n\r\n")
	p := NewParser(data)
	_, err := p.ParseResponse()
	if err == nil {
		t.Error("expected error for invalid status code")
	}
}

func TestEqFold(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"Content-Type", "content-type", true},
		{"HOST", "host", true},
		{"Host", "Host", true},
		{"Host", "Hos", false},
		{"", "", true},
	}
	for _, tt := range tests {
		got := eqFold(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("eqFold(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestSplitComma(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"a,b,c", []string{"a", "b", "c"}},
		{"single", []string{"single"}},
		{"a, b", []string{"a", " b"}}, // spaces not trimmed by splitComma itself
		{"", []string{""}},
		{"a,,b", []string{"a", "", "b"}},
		{"gzip, chunked", []string{"gzip", " chunked"}},
	}
	for _, tt := range tests {
		got := splitComma(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("splitComma(%q) = %v (len %d), want %v (len %d)", tt.input, got, len(got), tt.want, len(tt.want))
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("splitComma(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestTrimString(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"  hello  ", "hello"},
		{"\t  world \t", "world"},
		{"no-trim", "no-trim"},
		{"  ", ""},
		{"", ""},
		{" a ", "a"},
	}
	for _, tt := range tests {
		got := trimString(tt.input)
		if got != tt.want {
			t.Errorf("trimString(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestStripChunked(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"chunked", ""},
		{"CHUNKED", ""},
		{"gzip, chunked", "gzip"},
		{"gzip, deflate, chunked", "gzip, deflate"},
		{"gzip", "gzip"},       // no chunked present — returned as-is
		{"deflate", "deflate"}, // no chunked present
	}
	for _, tt := range tests {
		got := stripChunked(tt.input)
		if got != tt.want {
			t.Errorf("stripChunked(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeChunkedHeaders_WithExistingContentLength(t *testing.T) {
	// The hasContentLength=true branch: headers already have Content-Length
	headers := []Header{
		{Key: "Transfer-Encoding", Value: "chunked"},
		{Key: "Content-Length", Value: "999"}, // should be replaced with decoded length
		{Key: "Host", Value: "example.com"},
	}
	result := normalizeChunkedHeaders(headers, 42)

	foundTE := false
	foundCL := false
	for _, h := range result {
		if eqFold(h.Key, "Transfer-Encoding") {
			foundTE = true
		}
		if eqFold(h.Key, "Content-Length") {
			foundCL = true
			if h.Value != "42" {
				t.Errorf("Content-Length = %q, want 42", h.Value)
			}
		}
	}
	if foundTE {
		t.Error("Transfer-Encoding should be removed after chunked decode")
	}
	if !foundCL {
		t.Error("Content-Length should be present after chunked decode")
	}
}

func TestNormalizeChunkedHeaders_MultiValueTE(t *testing.T) {
	// Transfer-Encoding: gzip, chunked → gzip remains, Content-Length added
	headers := []Header{
		{Key: "Transfer-Encoding", Value: "gzip, chunked"},
	}
	result := normalizeChunkedHeaders(headers, 10)

	foundTE := false
	foundCL := false
	for _, h := range result {
		if eqFold(h.Key, "Transfer-Encoding") {
			foundTE = true
			if h.Value != "gzip" {
				t.Errorf("Transfer-Encoding = %q, want gzip", h.Value)
			}
		}
		if eqFold(h.Key, "Content-Length") {
			foundCL = true
			if h.Value != "10" {
				t.Errorf("Content-Length = %q, want 10", h.Value)
			}
		}
	}
	if !foundTE {
		t.Error("Transfer-Encoding: gzip should remain")
	}
	if !foundCL {
		t.Error("Content-Length should be added")
	}
}

func TestParseRequest_MultipleHeaders(t *testing.T) {
	data := []byte("GET / HTTP/1.1\r\nHost: example.com\r\nAccept: text/html\r\nX-Custom: value\r\n\r\n")
	p := NewParser(data)
	req, err := p.ParseRequest()
	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}
	if len(req.Headers) != 3 {
		t.Errorf("Headers count = %d, want 3", len(req.Headers))
	}
}

func TestParseResponse_NoReason(t *testing.T) {
	// Status line with no reason phrase — should parse with empty reason
	data := []byte("HTTP/1.1 200\r\n\r\n")
	p := NewParser(data)
	resp, err := p.ParseResponse()
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if resp.Reason != "" {
		t.Errorf("Reason = %q, want empty", resp.Reason)
	}
}

func TestParseRequest_EmptyPath(t *testing.T) {
	// Empty method/path detection
	data := []byte(" /path HTTP/1.1\r\n\r\n")
	p := NewParser(data)
	_, err := p.ParseRequest()
	if err == nil {
		t.Error("expected error for empty method")
	}
}

// Unmarshal, DetectMessageType, and Validate tests

func TestContainsFold_NeedleLongerThanHaystack(t *testing.T) {
	// When needle is longer than haystack, should return false
	got := containsFold("hi", "chunked")
	if got {
		t.Error("containsFold(short, long) should return false")
	}
}

func TestContainsFold_NoMatch(t *testing.T) {
	// needle length <= haystack length but no match found — returns false at loop exit
	got := containsFold("gzip, deflate", "chunked")
	if got {
		t.Error("containsFold(gzip, deflate, chunked) should return false")
	}
}

func TestTrimOWS_Tab(t *testing.T) {
	// Test tab trimming in trimOWS
	input := []byte("\t value \t")
	got := trimOWS(input)
	if string(got) != "value" {
		t.Errorf("trimOWS(tab-padded) = %q, want value", string(got))
	}
}

func TestSkipLineEnding_NoCRLFOrLF(t *testing.T) {
	// skipLineEnding when data[pos] is neither \r nor \n — returns pos unchanged
	data := []byte("hello")
	pos := 2 // points to 'l'
	got := skipLineEnding(data, pos)
	if got != pos {
		t.Errorf("skipLineEnding(non-newline) = %d, want %d (unchanged)", got, pos)
	}
}

func TestSkipLineEnding_BareCR(t *testing.T) {
	// skipLineEnding when data[pos] is '\r' but not followed by '\n' — returns pos+2? No, returns pos
	// Actually: data[pos] == '\r' && pos+1 < len(data) && data[pos+1] == '\n' is false for bare CR
	// Then data[pos] == '\n' is also false
	// So returns pos unchanged
	data := []byte("\rhello")
	pos := 0
	got := skipLineEnding(data, pos)
	// \r is present but not followed by \n — returns pos (unchanged)
	if got != pos {
		t.Errorf("skipLineEnding(bare CR) = %d, want %d (unchanged)", got, pos)
	}
}

func TestParseResponse_UnknownReason(t *testing.T) {
	// Reason phrase not in the intern table → triggers internReason fallback
	data := []byte("HTTP/1.1 418 I'm a Teapot\r\n\r\n")
	p := NewParser(data)
	resp, err := p.ParseResponse()
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if resp.Reason != "I'm a Teapot" {
		t.Errorf("Reason = %q, want I'm a Teapot", resp.Reason)
	}
}

func TestParseRequest_EmptyData(t *testing.T) {
	// Empty data: readLine hits p.pos >= p.length branch
	p := NewParser([]byte{})
	_, err := p.ParseRequest()
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestParseResponse_EmptyData(t *testing.T) {
	// Empty data: readLine hits p.pos >= p.length branch
	p := NewParser([]byte{})
	_, err := p.ParseResponse()
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestReadLine_BareLF(t *testing.T) {
	// readLine with bare LF line endings exercises the '\n' branch
	data := []byte("GET / HTTP/1.1\nHost: example.com\n\n")
	p := NewParser(data)
	req, err := p.ParseRequest()
	if err != nil {
		t.Fatalf("ParseRequest() with bare LF error = %v", err)
	}
	if req.Method != "GET" {
		t.Errorf("Method = %q, want GET", req.Method)
	}
	if len(req.Headers) != 1 {
		t.Errorf("Headers count = %d, want 1", len(req.Headers))
	}
}

func TestReadLine_NoLineEnding(t *testing.T) {
	// readLine on data with no line ending returns remaining data without error
	// We test through ParseRequest with a trailing header that has no newline
	data := []byte("GET / HTTP/1.1\r\nX-Header: value")
	p := NewParser(data)
	req, err := p.ParseRequest()
	// Should parse without error; header may or may not be included
	_ = err
	_ = req
	// Main goal: exercise the no-line-ending code path without panicking
}

func TestTrimOWSBytes_Tab(t *testing.T) {
	// Test tab trimming in trimOWSBytes
	b := []byte("\t value \t")
	got := trimOWSBytes(b)
	if string(got) != "value" {
		t.Errorf("trimOWSBytes(tab-padded) = %q, want value", string(got))
	}
}

func TestUnmarshal_Request(t *testing.T) {
	data := []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")
	result, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	req, ok := result.(*Request)
	if !ok {
		t.Fatalf("expected *Request, got %T", result)
	}
	if req.Method != "GET" {
		t.Errorf("Method = %q, want GET", req.Method)
	}
}

func TestUnmarshal_Response(t *testing.T) {
	data := []byte("HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n")
	result, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	resp, ok := result.(*Response)
	if !ok {
		t.Fatalf("expected *Response, got %T", result)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
}

func TestDetectMessageType(t *testing.T) {
	tests := []struct {
		data []byte
		want string
	}{
		{[]byte("HTTP/1.1 200 OK\r\n"), "response"},
		{[]byte("GET / HTTP/1.1\r\n"), "request"},
		{[]byte("POST /api HTTP/1.1\r\n"), "request"},
		{[]byte("HTTP/1.0 404 Not Found\r\n"), "response"},
		{[]byte(""), "request"},
	}
	for _, tt := range tests {
		got := DetectMessageType(tt.data)
		if got != tt.want {
			t.Errorf("DetectMessageType(%q) = %q, want %q", tt.data, got, tt.want)
		}
	}
}

func TestValidate_ValidRequest(t *testing.T) {
	err := Validate([]byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"))
	if err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestValidate_ValidResponse(t *testing.T) {
	err := Validate([]byte("HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n"))
	if err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestValidate_Invalid(t *testing.T) {
	err := Validate([]byte("NOTHTTP\r\n\r\n"))
	if err == nil {
		t.Error("Validate() = nil, want error for invalid HTTP")
	}
}

// Additional parser coverage tests

func TestParseRequest_ObsFold(t *testing.T) {
	// Obs-fold: continuation lines start with SP or HTAB
	data := []byte("GET / HTTP/1.1\r\nX-Folded: part1\r\n continued\r\n\r\n")
	p := NewParser(data)
	req, err := p.ParseRequest()
	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}
	if len(req.Headers) != 1 {
		t.Fatalf("Headers count = %d, want 1", len(req.Headers))
	}
	// obs-fold joins with single SP
	if req.Headers[0].Key != "X-Folded" {
		t.Errorf("Header key = %q, want X-Folded", req.Headers[0].Key)
	}
}

func TestParseResponse_TruncatedBody(t *testing.T) {
	// Content-Length says 100 but only 5 bytes provided
	data := []byte("HTTP/1.1 200 OK\r\nContent-Length: 100\r\n\r\nshort")
	p := NewParser(data)
	_, err := p.ParseResponse()
	if err == nil {
		t.Error("expected error for truncated body")
	}
}

func TestParseRequest_RemainingBytesBody(t *testing.T) {
	// No Content-Length, not chunked — remaining bytes become body
	data := []byte("POST / HTTP/1.1\r\nHost: example.com\r\n\r\nsome body here")
	p := NewParser(data)
	req, err := p.ParseRequest()
	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}
	if string(req.Body) != "some body here" {
		t.Errorf("Body = %q, want 'some body here'", string(req.Body))
	}
}

func TestParseRequest_WhitespaceBeforeColon(t *testing.T) {
	// RFC 9112: whitespace before colon is not allowed
	data := []byte("GET / HTTP/1.1\r\nHost : example.com\r\n\r\n")
	p := NewParser(data)
	_, err := p.ParseRequest()
	if err == nil {
		t.Error("expected error for whitespace before colon")
	}
}

func TestParseChunkedRequestError(t *testing.T) {
	// Malformed chunked body → ParseRequest should return error
	data := []byte("POST / HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\nXXX\r\n")
	p := NewParser(data)
	_, err := p.ParseRequest()
	if err == nil {
		t.Error("expected error for malformed chunked body")
	}
}

func TestGetContentLength_InvalidValue(t *testing.T) {
	// Content-Length with non-numeric value returns -1
	headers := []Header{
		{Key: "Content-Length", Value: "not-a-number"},
	}
	got := getContentLength(headers)
	if got != -1 {
		t.Errorf("getContentLength(invalid) = %d, want -1", got)
	}
}

func TestParseRequest_TabOWSHeader(t *testing.T) {
	// Header value with leading/trailing tab OWS
	data := []byte("GET / HTTP/1.1\r\nHost:\texample.com\t\r\n\r\n")
	p := NewParser(data)
	req, err := p.ParseRequest()
	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}
	if len(req.Headers) != 1 {
		t.Fatalf("Headers count = %d, want 1", len(req.Headers))
	}
	if req.Headers[0].Value != "example.com" {
		t.Errorf("Header value = %q, want example.com", req.Headers[0].Value)
	}
}

func TestParseRequest_ChunkedNormalized(t *testing.T) {
	// After parsing chunked request, Transfer-Encoding should be removed and
	// Content-Length should be set to decoded body length.
	data := []byte("POST / HTTP/1.1\r\nHost: example.com\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nhello\r\n0\r\n\r\n")
	p := NewParser(data)
	req, err := p.ParseRequest()
	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}
	if string(req.Body) != "hello" {
		t.Errorf("Body = %q, want hello", string(req.Body))
	}
	for _, h := range req.Headers {
		if eqFold(h.Key, "Transfer-Encoding") {
			t.Error("Transfer-Encoding should be removed after chunked decode")
		}
	}
	found := false
	for _, h := range req.Headers {
		if eqFold(h.Key, "Content-Length") && h.Value == "5" {
			found = true
		}
	}
	if !found {
		t.Error("Content-Length: 5 should be added after chunked decode")
	}
}

// TestParseRequest_EmptyPath2 tests the "empty request path" error in parseRequestLine.
// "GET  HTTP/1.1" has two spaces, so method="GET", rest=" HTTP/1.1", sp2=0, path="".
func TestParseRequest_EmptyPath2(t *testing.T) {
	data := []byte("GET  HTTP/1.1\r\n\r\n")
	p := NewParser(data)
	_, err := p.ParseRequest()
	if err == nil {
		t.Error("expected error for empty request path (double-space request line)")
	}
}

// TestParseResponse_InvalidStatusCodeNoReason tests the error path in parseStatusLine
// when there is no reason phrase (sp2 < 0) and the status code is not numeric.
func TestParseResponse_InvalidStatusCodeNoReason(t *testing.T) {
	data := []byte("HTTP/1.1 abc\r\n\r\n")
	p := NewParser(data)
	_, err := p.ParseResponse()
	if err == nil {
		t.Error("expected error for non-numeric status code with no reason phrase")
	}
}

// TestParseResponse_ParseBodyError exercises parseBody returning an error through
// ParseResponse — e.g., chunked decode failure propagating back.
func TestParseResponse_ParseBodyError(t *testing.T) {
	// Malformed chunked body (invalid hex) should cause ParseResponse to return error.
	data := []byte("HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\nZZZ\r\n")
	p := NewParser(data)
	_, err := p.ParseResponse()
	if err == nil {
		t.Error("expected error for malformed chunked body in ParseResponse")
	}
}

// TestParseResponse_MalformedHeader exercises the parseHeaders error path in ParseResponse.
// A header with whitespace before the colon causes parseHeaders to return an error.
func TestParseResponse_MalformedHeader(t *testing.T) {
	data := []byte("HTTP/1.1 200 OK\r\nBad-Header : value\r\n\r\n")
	p := NewParser(data)
	_, err := p.ParseResponse()
	if err == nil {
		t.Error("expected error for response header with whitespace before colon")
	}
}

// TestParseHeaders_TruncatedAtEnd exercises parseHeaders when data ends without
// an empty line — the "pos >= length" early return in the header loop.
func TestParseHeaders_TruncatedAtEnd(t *testing.T) {
	// No terminating CRLF after headers
	data := []byte("GET / HTTP/1.1\r\nHost: example.com")
	p := NewParser(data)
	req, err := p.ParseRequest()
	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}
	if req.Headers[0].Key != "Host" {
		t.Errorf("Header key = %q, want Host", req.Headers[0].Key)
	}
}
