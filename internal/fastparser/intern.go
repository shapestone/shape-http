package fastparser

// String interning for common HTTP tokens.
//
// The Go compiler optimizes map lookups with string([]byte) keys
// to avoid allocating the temporary string (the mapaccess optimization).
// This means internMethod(someBytes) is zero-alloc for known methods.

var methods = map[string]string{
	"GET": "GET", "HEAD": "HEAD", "POST": "POST",
	"PUT": "PUT", "DELETE": "DELETE", "CONNECT": "CONNECT",
	"OPTIONS": "OPTIONS", "TRACE": "TRACE", "PATCH": "PATCH",
}

var versions = map[string]string{
	"HTTP/1.0": "HTTP/1.0", "HTTP/1.1": "HTTP/1.1",
	"HTTP/2": "HTTP/2", "HTTP/2.0": "HTTP/2.0",
}

var headerNames = map[string]string{
	"Accept":              "Accept",
	"Accept-Charset":      "Accept-Charset",
	"Accept-Encoding":     "Accept-Encoding",
	"Accept-Language":     "Accept-Language",
	"Accept-Ranges":       "Accept-Ranges",
	"Age":                 "Age",
	"Allow":               "Allow",
	"Authorization":       "Authorization",
	"Cache-Control":       "Cache-Control",
	"Connection":          "Connection",
	"Content-Disposition": "Content-Disposition",
	"Content-Encoding":    "Content-Encoding",
	"Content-Language":    "Content-Language",
	"Content-Length":      "Content-Length",
	"Content-Location":    "Content-Location",
	"Content-Range":       "Content-Range",
	"Content-Type":        "Content-Type",
	"Cookie":              "Cookie",
	"Date":                "Date",
	"ETag":                "ETag",
	"Expect":              "Expect",
	"Expires":             "Expires",
	"From":                "From",
	"Host":                "Host",
	"If-Match":            "If-Match",
	"If-Modified-Since":   "If-Modified-Since",
	"If-None-Match":       "If-None-Match",
	"If-Range":            "If-Range",
	"If-Unmodified-Since": "If-Unmodified-Since",
	"Last-Modified":       "Last-Modified",
	"Location":            "Location",
	"Max-Forwards":        "Max-Forwards",
	"Origin":              "Origin",
	"Pragma":              "Pragma",
	"Proxy-Authenticate":  "Proxy-Authenticate",
	"Proxy-Authorization": "Proxy-Authorization",
	"Range":               "Range",
	"Referer":             "Referer",
	"Retry-After":         "Retry-After",
	"Server":              "Server",
	"Set-Cookie":          "Set-Cookie",
	"TE":                  "TE",
	"Trailer":             "Trailer",
	"Transfer-Encoding":   "Transfer-Encoding",
	"Upgrade":             "Upgrade",
	"User-Agent":          "User-Agent",
	"Vary":                "Vary",
	"Via":                 "Via",
	"Warning":             "Warning",
	"WWW-Authenticate":    "WWW-Authenticate",
	"X-Forwarded-For":     "X-Forwarded-For",
	"X-Forwarded-Host":    "X-Forwarded-Host",
	"X-Forwarded-Proto":   "X-Forwarded-Proto",
	"X-Request-ID":        "X-Request-ID",
	"X-Real-IP":           "X-Real-IP",
}

var reasons = map[string]string{
	"OK":                    "OK",
	"Created":               "Created",
	"Accepted":              "Accepted",
	"No Content":            "No Content",
	"Moved Permanently":     "Moved Permanently",
	"Found":                 "Found",
	"Not Modified":          "Not Modified",
	"Bad Request":           "Bad Request",
	"Unauthorized":          "Unauthorized",
	"Forbidden":             "Forbidden",
	"Not Found":             "Not Found",
	"Method Not Allowed":    "Method Not Allowed",
	"Conflict":              "Conflict",
	"Gone":                  "Gone",
	"Internal Server Error": "Internal Server Error",
	"Not Implemented":       "Not Implemented",
	"Bad Gateway":           "Bad Gateway",
	"Service Unavailable":   "Service Unavailable",
	"Gateway Timeout":       "Gateway Timeout",
}

// internMethod returns an interned string for known HTTP methods, avoiding allocation.
func internMethod(b []byte) string {
	if s, ok := methods[string(b)]; ok {
		return s
	}
	return string(b)
}

// internVersion returns an interned string for known HTTP versions, avoiding allocation.
func internVersion(b []byte) string {
	if s, ok := versions[string(b)]; ok {
		return s
	}
	return string(b)
}

// internHeaderName returns an interned string for known header names, avoiding allocation.
func internHeaderName(b []byte) string {
	if s, ok := headerNames[string(b)]; ok {
		return s
	}
	return string(b)
}

// internReason returns an interned string for known reason phrases, avoiding allocation.
func internReason(b []byte) string {
	if s, ok := reasons[string(b)]; ok {
		return s
	}
	return string(b)
}
