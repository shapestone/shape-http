package http

import (
	"bytes"
	"fmt"

	"github.com/shapestone/shape-http/internal/fastparser"
)

// Unmarshal parses the HTTP wire-format data and stores the result in v.
//
// v must be a *Request or *Response. The function auto-detects the message type
// based on whether data starts with "HTTP/" (response) or not (request).
//
// This function uses a high-performance fast path that bypasses AST construction.
func Unmarshal(data []byte, v interface{}) error {
	if v == nil {
		return fmt.Errorf("http: Unmarshal(nil)")
	}

	// Check for Unmarshaler interface
	if u, ok := v.(Unmarshaler); ok {
		return u.UnmarshalHTTP(data)
	}

	isResp := bytes.HasPrefix(data, []byte("HTTP/"))

	switch target := v.(type) {
	case *Request:
		if isResp {
			return fmt.Errorf("http: data appears to be a response but target is *Request")
		}
		return unmarshalRequest(data, target)

	case *Response:
		if !isResp {
			return fmt.Errorf("http: data appears to be a request but target is *Response")
		}
		return unmarshalResponse(data, target)

	default:
		return fmt.Errorf("http: Unmarshal unsupported type %T (expected *Request or *Response)", v)
	}
}

// UnmarshalRequest parses HTTP wire-format data as a request.
func UnmarshalRequest(data []byte) (*Request, error) {
	req := &Request{}
	if err := unmarshalRequest(data, req); err != nil {
		return nil, err
	}
	return req, nil
}

// UnmarshalResponse parses HTTP wire-format data as a response.
func UnmarshalResponse(data []byte) (*Response, error) {
	resp := &Response{}
	if err := unmarshalResponse(data, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// DetectMessageType returns "request" or "response" based on the data prefix.
// Data starting with "HTTP/" is detected as a response; everything else as a request.
func DetectMessageType(data []byte) string {
	return fastparser.DetectMessageType(data)
}

func unmarshalRequest(data []byte, target *Request) error {
	req, err := fastparser.UnmarshalRequest(data)
	if err != nil {
		return err
	}
	target.Method = req.Method
	target.Path = req.Path
	target.Version = req.Version
	target.Headers = convertHeaders(req.Headers)
	target.Body = req.Body
	return nil
}

func unmarshalResponse(data []byte, target *Response) error {
	resp, err := fastparser.UnmarshalResponse(data)
	if err != nil {
		return err
	}
	target.Version = resp.Version
	target.StatusCode = resp.StatusCode
	target.Reason = resp.Reason
	target.Headers = convertHeaders(resp.Headers)
	target.Body = resp.Body
	return nil
}

func convertHeaders(internal []fastparser.Header) Headers {
	if len(internal) == 0 {
		return nil
	}
	headers := make(Headers, len(internal))
	for i, h := range internal {
		headers[i] = Header{Key: h.Key, Value: h.Value}
	}
	return headers
}
