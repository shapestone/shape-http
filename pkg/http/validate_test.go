package http

import (
	"io"
	"strings"
	"testing"
)

func TestValidate_ValidRequest(t *testing.T) {
	err := Validate("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")
	if err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestValidate_ValidResponse(t *testing.T) {
	err := Validate("HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n")
	if err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestValidate_ValidRequestWithBody(t *testing.T) {
	err := Validate("POST /api HTTP/1.1\r\nHost: example.com\r\nContent-Length: 11\r\n\r\nhello world")
	if err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestValidate_MalformedRequestLine(t *testing.T) {
	err := Validate("GETHTTP/1.1\r\n\r\n")
	if err == nil {
		t.Error("expected error for malformed request line")
	}
}

func TestValidate_WhitespaceBeforeColon(t *testing.T) {
	err := Validate("GET / HTTP/1.1\r\nHost : example.com\r\n\r\n")
	if err == nil {
		t.Error("expected error for whitespace before colon")
	}
}

func TestValidate_TruncatedBody(t *testing.T) {
	err := Validate("POST / HTTP/1.1\r\nContent-Length: 100\r\n\r\nshort")
	if err == nil {
		t.Error("expected error for truncated body")
	}
}

func TestValidateReader(t *testing.T) {
	r := strings.NewReader("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")
	err := ValidateReader(r)
	if err != nil {
		t.Errorf("ValidateReader() = %v, want nil", err)
	}
}

func TestValidateReader_Invalid(t *testing.T) {
	r := strings.NewReader("NOTHTTP\r\n\r\n")
	err := ValidateReader(r)
	if err == nil {
		t.Error("ValidateReader() = nil, want error for invalid HTTP")
	}
}

func TestValidateReader_Response(t *testing.T) {
	r := strings.NewReader("HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n")
	err := ValidateReader(r)
	if err != nil {
		t.Errorf("ValidateReader() = %v, want nil", err)
	}
}

func TestValidateReader_IOError(t *testing.T) {
	// errReader is defined in parser_test.go (same package)
	r := &errReader{err: io.ErrUnexpectedEOF}
	err := ValidateReader(r)
	if err == nil {
		t.Error("ValidateReader() = nil, want error for reader failure")
	}
}
