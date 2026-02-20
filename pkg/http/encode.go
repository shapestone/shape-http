package http

import (
	"io"
)

// Encoder writes HTTP messages to an output stream in HTTP/1.1 wire format.
// A single Encoder is not safe for concurrent use; create one per goroutine
// or serialize access externally.
type Encoder struct {
	w io.Writer
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

// Encode writes the HTTP wire-format encoding of v to the stream.
// v must be a *Request or *Response.
func (enc *Encoder) Encode(v interface{}) error {
	data, err := Marshal(v)
	if err != nil {
		return err
	}
	_, err = enc.w.Write(data)
	return err
}
