package fastparser

import (
	"bytes"
	"encoding/hex"
	"fmt"
)

// Dechunk decodes a chunked transfer-encoded body.
//
// Format: hex-size CRLF data CRLF ... 0 CRLF [trailers] CRLF
// Chunk extensions after ';' are ignored.
func Dechunk(data []byte) ([]byte, error) {
	var result []byte
	pos := 0
	length := len(data)

	for {
		if pos >= length {
			return nil, fmt.Errorf("http: chunked encoding: unexpected end of data")
		}

		// Read chunk size line
		lineEnd := findLineEnd(data, pos)
		if lineEnd < 0 {
			return nil, fmt.Errorf("http: chunked encoding: unterminated chunk size line")
		}

		sizeLine := data[pos:lineEnd]
		// Advance past the line ending
		pos = skipLineEnding(data, lineEnd)

		// Strip chunk extension (everything after ';')
		if semi := bytes.IndexByte(sizeLine, ';'); semi >= 0 {
			sizeLine = sizeLine[:semi]
		}
		sizeLine = bytes.TrimSpace(sizeLine)

		// Parse hex size
		sizeStr := string(sizeLine)
		size, err := parseHexSize(sizeStr)
		if err != nil {
			return nil, fmt.Errorf("http: chunked encoding: invalid chunk size %q: %w", sizeStr, err)
		}

		// size 0 = last chunk
		if size == 0 {
			// Skip optional trailers and final CRLF
			break
		}

		// Read chunk data
		if pos+size > length {
			return nil, fmt.Errorf("http: chunked encoding: chunk data truncated (expected %d bytes, %d available)", size, length-pos)
		}
		result = append(result, data[pos:pos+size]...)
		pos += size

		// Expect CRLF after chunk data
		if pos >= length {
			return nil, fmt.Errorf("http: chunked encoding: missing CRLF after chunk data")
		}
		if data[pos] == '\r' && pos+1 < length && data[pos+1] == '\n' {
			pos += 2
		} else if data[pos] == '\n' {
			pos++
		} else {
			return nil, fmt.Errorf("http: chunked encoding: expected CRLF after chunk data, got %q", data[pos])
		}
	}

	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

// findLineEnd finds the position of \r\n or \n starting from pos.
// Returns the position of \r (or \n if bare), or -1 if not found.
func findLineEnd(data []byte, pos int) int {
	for i := pos; i < len(data); i++ {
		if data[i] == '\r' && i+1 < len(data) && data[i+1] == '\n' {
			return i
		}
		if data[i] == '\n' {
			return i
		}
	}
	return -1
}

// skipLineEnding advances past CRLF or LF at the given position.
func skipLineEnding(data []byte, pos int) int {
	if pos < len(data) && data[pos] == '\r' && pos+1 < len(data) && data[pos+1] == '\n' {
		return pos + 2
	}
	if pos < len(data) && data[pos] == '\n' {
		return pos + 1
	}
	return pos
}

// parseHexSize parses a hex string into an integer.
func parseHexSize(s string) (int, error) {
	if len(s) == 0 {
		return 0, fmt.Errorf("empty hex string")
	}
	// Fast path for small hex values
	if len(s) <= 8 {
		var n int
		for _, c := range s {
			n <<= 4
			switch {
			case c >= '0' && c <= '9':
				n |= int(c - '0')
			case c >= 'a' && c <= 'f':
				n |= int(c-'a') + 10
			case c >= 'A' && c <= 'F':
				n |= int(c-'A') + 10
			default:
				return 0, hex.InvalidByteError(byte(c))
			}
		}
		return n, nil
	}

	b, err := hex.DecodeString(padHex(s))
	if err != nil {
		return 0, err
	}
	n := 0
	for _, v := range b {
		n = (n << 8) | int(v)
	}
	return n, nil
}

// padHex pads a hex string to even length.
func padHex(s string) string {
	if len(s)%2 != 0 {
		return "0" + s
	}
	return s
}
