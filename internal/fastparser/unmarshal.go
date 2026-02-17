package fastparser

import (
	"bytes"
	"fmt"
)

// UnmarshalRequest parses data as an HTTP request.
// Uses stack-allocated Parser to avoid heap allocation.
func UnmarshalRequest(data []byte) (*Request, error) {
	var p Parser
	initParser(&p, data)
	return p.ParseRequest()
}

// UnmarshalResponse parses data as an HTTP response.
// Uses stack-allocated Parser to avoid heap allocation.
func UnmarshalResponse(data []byte) (*Response, error) {
	var p Parser
	initParser(&p, data)
	return p.ParseResponse()
}

// Unmarshal auto-detects whether data is a request or response and parses it.
// If data starts with "HTTP/" it is treated as a response; otherwise a request.
func Unmarshal(data []byte) (interface{}, error) {
	if bytes.HasPrefix(data, []byte("HTTP/")) {
		return UnmarshalResponse(data)
	}
	return UnmarshalRequest(data)
}

// DetectMessageType returns "request" or "response" based on the data prefix.
func DetectMessageType(data []byte) string {
	if bytes.HasPrefix(data, []byte("HTTP/")) {
		return "response"
	}
	return "request"
}

// Validate checks that data is a valid HTTP message without returning a result.
func Validate(data []byte) error {
	_, err := Unmarshal(data)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	return nil
}
