// Package http provides HTTP/1.1 message parsing and serialization per RFC 9112.
//
// This package implements a complete HTTP/1.1 wire-format parser that can parse
// and serialize both requests and responses, including chunked transfer encoding.
//
// # Thread Safety
//
// All functions in this package are safe for concurrent use by multiple goroutines.
// Each function call creates its own parser instance with no shared mutable state.
//
// # Parsing APIs
//
// The package provides multiple parsing paths:
//
//   - Unmarshal/UnmarshalRequest/UnmarshalResponse - Fast direct parsing
//   - Parse/ParseReader - AST-based parsing via shape-core
//   - UnmarshalLenient - Best-effort parsing with warnings
//   - NewDecoder - Streaming io.Reader-based parsing
package http

import (
	"strconv"
	"strings"
)

// Request represents an HTTP/1.1 request message.
type Request struct {
	Method  string  // "GET", "POST", etc.
	Path    string  // request-target "/api/users?q=foo"
	Version string  // "HTTP/1.1"
	Headers Headers // ordered, repeatable headers
	Body    []byte  // raw body (nil if none)
}

// Response represents an HTTP/1.1 response message.
type Response struct {
	Version    string  // "HTTP/1.1"
	StatusCode int     // 200, 404, etc.
	Reason     string  // "OK", "Not Found"
	Headers    Headers // ordered, repeatable headers
	Body       []byte  // raw body (nil if none)
}

// Header represents a single HTTP header key-value pair.
type Header struct {
	Key   string
	Value string
}

// Headers is an ordered, repeatable list of HTTP headers.
// HTTP headers are case-insensitive by spec but we preserve original case.
type Headers []Header

// Get returns the first header value for the given key (case-insensitive).
// Returns empty string if not found.
func (h Headers) Get(key string) string {
	for _, hdr := range h {
		if strings.EqualFold(hdr.Key, key) {
			return hdr.Value
		}
	}
	return ""
}

// Values returns all header values for the given key (case-insensitive).
func (h Headers) Values(key string) []string {
	var vals []string
	for _, hdr := range h {
		if strings.EqualFold(hdr.Key, key) {
			vals = append(vals, hdr.Value)
		}
	}
	return vals
}

// Set replaces the first header with the given key (case-insensitive) or appends if not found.
func (h *Headers) Set(key, value string) {
	for i, hdr := range *h {
		if strings.EqualFold(hdr.Key, key) {
			(*h)[i].Value = value
			// Remove any subsequent headers with same key
			j := i + 1
			for j < len(*h) {
				if strings.EqualFold((*h)[j].Key, key) {
					*h = append((*h)[:j], (*h)[j+1:]...)
				} else {
					j++
				}
			}
			return
		}
	}
	*h = append(*h, Header{Key: key, Value: value})
}

// Add appends a header without replacing existing ones.
func (h *Headers) Add(key, value string) {
	*h = append(*h, Header{Key: key, Value: value})
}

// Del removes all headers with the given key (case-insensitive).
func (h *Headers) Del(key string) {
	j := 0
	for _, hdr := range *h {
		if !strings.EqualFold(hdr.Key, key) {
			(*h)[j] = hdr
			j++
		}
	}
	*h = (*h)[:j]
}

// Clone returns a deep copy of the headers.
func (h Headers) Clone() Headers {
	if h == nil {
		return nil
	}
	clone := make(Headers, len(h))
	copy(clone, h)
	return clone
}

// ContentLength returns the Content-Length header value, or -1 if absent or invalid.
func (h Headers) ContentLength() int64 {
	v := h.Get("Content-Length")
	if v == "" {
		return -1
	}
	n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	if err != nil {
		return -1
	}
	return n
}

// IsChunked returns true if Transfer-Encoding contains "chunked".
func (h Headers) IsChunked() bool {
	v := h.Get("Transfer-Encoding")
	return strings.Contains(strings.ToLower(v), "chunked")
}

// Message is the interface shared by Request and Response.
type Message interface {
	GetVersion() string
	GetHeaders() Headers
	GetBody() []byte
}

// GetVersion returns the HTTP version string.
func (r *Request) GetVersion() string { return r.Version }

// GetHeaders returns the headers.
func (r *Request) GetHeaders() Headers { return r.Headers }

// GetBody returns the body bytes.
func (r *Request) GetBody() []byte { return r.Body }

// GetVersion returns the HTTP version string.
func (r *Response) GetVersion() string { return r.Version }

// GetHeaders returns the headers.
func (r *Response) GetHeaders() Headers { return r.Headers }

// GetBody returns the body bytes.
func (r *Response) GetBody() []byte { return r.Body }

// Marshaler is the interface implemented by types that can marshal themselves
// into valid HTTP wire format.
type Marshaler interface {
	MarshalHTTP() ([]byte, error)
}

// Unmarshaler is the interface implemented by types that can unmarshal
// an HTTP wire-format description of themselves.
type Unmarshaler interface {
	UnmarshalHTTP([]byte) error
}

// ParseResult holds the result of lenient parsing.
type ParseResult struct {
	Request  *Request  // non-nil if request detected
	Response *Response // non-nil if response detected
	Warnings []string  // non-fatal issues
	Partial  bool      // true if message was incomplete/truncated
}
