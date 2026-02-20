package http

import (
	"bytes"
	"io"

	"github.com/shapestone/shape-http/internal/fastparser"
)

// Validate checks that input is a syntactically valid HTTP/1.1 message per
// RFC 9112. It parses both the start line and all headers; it does not
// evaluate body semantics.
// Returns nil if valid, or a descriptive error identifying the problem.
func Validate(input string) error {
	return fastparser.Validate([]byte(input))
}

// ValidateReader reads all data from r and validates it as an HTTP/1.1 message.
// See Validate for the validation semantics.
func ValidateReader(r io.Reader) error {
	data, err := readAll(r)
	if err != nil {
		return err
	}
	return fastparser.Validate(data)
}

// readAll reads all data from r. This is a simple helper to avoid
// importing io (which is already imported) for io.ReadAll.
func readAll(r io.Reader) ([]byte, error) {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
