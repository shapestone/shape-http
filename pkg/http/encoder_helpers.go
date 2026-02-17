package http

import "strconv"

// appendCRLF appends \r\n to buf.
func appendCRLF(buf []byte) []byte {
	return append(buf, '\r', '\n')
}

// appendRequestLine appends "METHOD PATH VERSION\r\n" to buf.
func appendRequestLine(buf []byte, method, path, version string) []byte {
	buf = append(buf, method...)
	buf = append(buf, ' ')
	buf = append(buf, path...)
	buf = append(buf, ' ')
	buf = append(buf, version...)
	return appendCRLF(buf)
}

// appendStatusLine appends "VERSION STATUS REASON\r\n" to buf.
func appendStatusLine(buf []byte, version string, statusCode int, reason string) []byte {
	buf = append(buf, version...)
	buf = append(buf, ' ')
	buf = strconv.AppendInt(buf, int64(statusCode), 10)
	buf = append(buf, ' ')
	buf = append(buf, reason...)
	return appendCRLF(buf)
}
