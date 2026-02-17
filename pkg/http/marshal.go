package http

import (
	"fmt"
	"sync"
)

// bufPool pools []byte slices for the encoder fast path.
var bufPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 0, 2048)
		return &b
	},
}

// Marshal returns the HTTP/1.1 wire-format encoding of v.
//
// v must be a *Request or *Response. If body is present and Content-Length
// header is absent (and Transfer-Encoding is not chunked), Content-Length
// is automatically set.
//
// Marshal uses a sync.Pool buffer internally for zero-alloc serialization.
func Marshal(v interface{}) ([]byte, error) {
	if v == nil {
		return nil, fmt.Errorf("http: Marshal(nil)")
	}

	// Check for Marshaler interface
	if m, ok := v.(Marshaler); ok {
		return m.MarshalHTTP()
	}

	bp := bufPool.Get().(*[]byte)
	buf := (*bp)[:0]

	var err error
	switch msg := v.(type) {
	case *Request:
		buf, err = appendRequest(buf, msg)
	case *Response:
		buf, err = appendResponse(buf, msg)
	default:
		*bp = buf
		bufPool.Put(bp)
		return nil, fmt.Errorf("http: Marshal unsupported type %T (expected *Request or *Response)", v)
	}

	if err != nil {
		*bp = buf
		bufPool.Put(bp)
		return nil, err
	}

	result := make([]byte, len(buf))
	copy(result, buf)
	*bp = buf
	bufPool.Put(bp)
	return result, nil
}
