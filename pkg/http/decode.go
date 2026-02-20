package http

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Decoder reads HTTP messages from an input stream in HTTP/1.1 wire format.
// A single Decoder is not safe for concurrent use; create one per goroutine
// or serialize access externally.
type Decoder struct {
	r *bufio.Reader
}

// NewDecoder returns a new decoder that reads from r.
// The decoder uses buffered reading for efficient parsing.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: bufio.NewReader(r)}
}

// Decode reads the next HTTP message and stores it in v.
// v must be a *Request or *Response.
func (dec *Decoder) Decode(v interface{}) error {
	// Peek to determine message type
	prefix, err := dec.r.Peek(5)
	if err != nil {
		return fmt.Errorf("http: decode: %w", err)
	}

	isResponse := bytes.HasPrefix(prefix, []byte("HTTP/"))

	switch target := v.(type) {
	case *Request:
		if isResponse {
			return fmt.Errorf("http: data appears to be a response but target is *Request")
		}
		return dec.decodeRequest(target)
	case *Response:
		if !isResponse {
			return fmt.Errorf("http: data appears to be a request but target is *Response")
		}
		return dec.decodeResponse(target)
	default:
		return fmt.Errorf("http: Decode unsupported type %T", v)
	}
}

// DecodeRequest reads the next HTTP request from the stream.
func (dec *Decoder) DecodeRequest() (*Request, error) {
	req := &Request{}
	if err := dec.decodeRequest(req); err != nil {
		return nil, err
	}
	return req, nil
}

// DecodeResponse reads the next HTTP response from the stream.
func (dec *Decoder) DecodeResponse() (*Response, error) {
	resp := &Response{}
	if err := dec.decodeResponse(resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (dec *Decoder) decodeRequest(req *Request) error {
	// Read request line
	line, err := dec.readLine()
	if err != nil {
		return fmt.Errorf("http: decode request: %w", err)
	}

	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 3 {
		return fmt.Errorf("http: decode request: malformed request line: %q", line)
	}
	req.Method = parts[0]
	req.Path = parts[1]
	req.Version = parts[2]

	// Read headers
	headers, err := dec.readHeaders()
	if err != nil {
		return err
	}
	req.Headers = headers

	// Read body
	body, err := dec.readBody(headers)
	if err != nil {
		return err
	}
	req.Body = body

	return nil
}

func (dec *Decoder) decodeResponse(resp *Response) error {
	// Read status line
	line, err := dec.readLine()
	if err != nil {
		return fmt.Errorf("http: decode response: %w", err)
	}

	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 2 {
		return fmt.Errorf("http: decode response: malformed status line: %q", line)
	}

	resp.Version = parts[0]
	code, convErr := strconv.Atoi(parts[1])
	if convErr != nil {
		return fmt.Errorf("http: decode response: invalid status code: %q", parts[1])
	}
	resp.StatusCode = code
	if len(parts) >= 3 {
		resp.Reason = parts[2]
	}

	// Read headers
	headers, err := dec.readHeaders()
	if err != nil {
		return err
	}
	resp.Headers = headers

	// Read body
	body, err := dec.readBody(headers)
	if err != nil {
		return err
	}
	resp.Body = body

	return nil
}

// readLine reads a line from the buffered reader, stripping CRLF or LF.
func (dec *Decoder) readLine() (string, error) {
	line, err := dec.r.ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", err
	}
	// Strip trailing \r\n or \n
	line = strings.TrimRight(line, "\r\n")
	return line, nil
}

// readHeaders reads header lines until an empty line.
func (dec *Decoder) readHeaders() (Headers, error) {
	var headers Headers

	for {
		line, err := dec.readLine()
		if err != nil {
			return headers, err
		}

		// Empty line = end of headers
		if line == "" {
			return headers, nil
		}

		// Parse "Key: Value"
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			return nil, fmt.Errorf("http: decode: malformed header line: %q", line)
		}

		key := line[:colon]
		value := strings.TrimSpace(line[colon+1:])
		headers = append(headers, Header{Key: key, Value: value})
	}
}

// readBody reads the message body based on headers.
func (dec *Decoder) readBody(headers Headers) ([]byte, error) {
	// Check Content-Length
	cl := headers.ContentLength()
	if cl > 0 {
		body := make([]byte, cl)
		_, err := io.ReadFull(dec.r, body)
		if err != nil {
			return nil, fmt.Errorf("http: decode body: %w", err)
		}
		return body, nil
	}
	if cl == 0 {
		return nil, nil
	}

	// Check chunked
	if headers.IsChunked() {
		return dec.readChunkedBody()
	}

	// No Content-Length, not chunked — no body for streaming decoder
	return nil, nil
}

// readChunkedBody reads a chunked transfer-encoded body from the stream.
func (dec *Decoder) readChunkedBody() ([]byte, error) {
	var result []byte

	for {
		// Read chunk size line
		sizeLine, err := dec.readLine()
		if err != nil {
			return nil, fmt.Errorf("http: decode chunked: %w", err)
		}

		// Strip chunk extension
		if idx := strings.IndexByte(sizeLine, ';'); idx >= 0 {
			sizeLine = sizeLine[:idx]
		}
		sizeLine = strings.TrimSpace(sizeLine)

		size, err := strconv.ParseInt(sizeLine, 16, 64)
		if err != nil {
			return nil, fmt.Errorf("http: decode chunked: invalid chunk size %q: %w", sizeLine, err)
		}

		if size == 0 {
			// Read trailing CRLF
			dec.readLine()
			break
		}

		// Read chunk data
		chunk := make([]byte, size)
		_, err = io.ReadFull(dec.r, chunk)
		if err != nil {
			return nil, fmt.Errorf("http: decode chunked: %w", err)
		}
		result = append(result, chunk...)

		// Read trailing CRLF after chunk data
		dec.readLine()
	}

	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}
