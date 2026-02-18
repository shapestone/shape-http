# Lenient Parsing

`UnmarshalLenient` and `ParseLenient` provide best-effort extraction from
malformed or non-RFC-compliant HTTP messages. They never return an error for
bad input — instead they extract what they can and report issues as warnings.

## When to use lenient parsing

| Use case | API |
|----------|-----|
| Debug tooling, log analysis | `UnmarshalLenient` |
| Proxy / middleware that must survive upstream bugs | `UnmarshalLenient` |
| Text editors, HTTP scratch-pad inputs | `UnmarshalLenient` |
| AST inspection of malformed messages | `ParseLenient` |
| Strict RFC 9112 compliance required | `UnmarshalRequest` / `UnmarshalResponse` |

## Result type

```go
type ParseResult struct {
    Request  *Request   // non-nil when a request was detected
    Response *Response  // non-nil when a response was detected
    Warnings []string   // human-readable descriptions of every issue found
    Partial  bool       // true if the message was truncated or incomplete
}
```

Only one of `Request` / `Response` is ever set. Auto-detection is based on
whether the input starts with `HTTP/` (response) or anything else (request).

## Request-line tolerances

| Deviation | Strict behaviour | Lenient behaviour |
|-----------|-----------------|-------------------|
| Missing HTTP version (`GET /path`) | Error | Default to `HTTP/1.1`, warn |
| Only method present (`GET`) | Error | Default path `/`, version `HTTP/1.1`, warn |
| Extra whitespace in request line | Error | Fields split, extra tokens ignored |

## Header tolerances

| Deviation | Strict behaviour | Lenient behaviour |
|-----------|-----------------|-------------------|
| Bare LF line endings (no CR) | Error | Accepted silently |
| Bare CR line endings | Error | Accepted silently |
| Whitespace before colon (`Key : value`) | Error | Trimmed, warn |
| Obs-fold continuation lines | Accepted | Accepted |
| No colon, looks like hostname (`example.com`) | Error | Treated as `Host: example.com`, warn |
| No colon, bare host:port (`example.com:8080`) — *CR-1* | Error | Treated as `Host: example.com:8080`, warn |
| No colon, bare word (`localhost`) | Error | Skipped, warn |
| Colon present, key is hostname, value is port — *CR-3* | Stored verbatim | Re-emitted as `Host: key:value`, warn |

### CR-1: bare hostname line

A line in the header section that has no colon but looks like a hostname
(contains a dot or an explicit port) is accepted as an implicit `Host` header:

```
POST /api/users HTTP/1.1
example.com              ← no "Host:" prefix
Content-Type: application/json
```

Result: `Host: example.com`, warning emitted.

### CR-3: host:port split on colon

A line like `example.com:8080` does have a colon, so the parser first splits
it into `key = "example.com"`, `value = "8080"`. CR-3 detects that the key
looks like a hostname (contains a dot, only hostname characters) and the value
is a bare port number (all digits), and re-emits the whole line as
`Host: example.com:8080` with a warning.

This is safe because legitimate HTTP header field-names never contain dots.

## Request-target (path) normalization

The lenient parser recognises six common malformed request forms and
normalises them into an origin-form path plus an injected `Host` header.

| Input form | Extracted `Path` | Injected `Host` |
|-----------|-----------------|----------------|
| `POST /api/users HTTP/1.1` | `/api/users` | — (normal) |
| `POST example.com:8080/api/users HTTP/1.1` | `/api/users` | `example.com:8080` |
| `POST https://example.com:8080/api/users HTTP/1.1` | `/api/users` | `example.com:8080` |
| `POST https://example.com:8080/api/users` *(no version)* | `/api/users` | `example.com:8080` |
| `POST https://example.com/api/users` *(no version)* | `/api/users` | `example.com` |
| `POST /api/users HTTP/1.1` + `example.com` bare header line | `/api/users` | `example.com` |
| `POST example.com/api/users HTTP/1.1` | `/api/users` | `example.com` |

**Precedence**: an explicit `Host` header supplied by the caller always wins
over the host extracted from the request-target.

### Absolute-form URLs

`https://example.com:8080/api/users` is technically valid HTTP (RFC 7230
§5.3.2 absolute-form, used by proxies), but most origin servers expect
origin-form paths. The lenient parser normalises the path and extracts the
host in all cases.

Userinfo (`user:pass@host`) is stripped when extracting the `Host` value.

## Body tolerances

| Deviation | Strict behaviour | Lenient behaviour |
|-----------|-----------------|-------------------|
| Body shorter than `Content-Length` | Error | Read all available bytes, `Partial = true`, warn |
| Body longer than `Content-Length` | Stops at declared length | Read all available bytes, warn |
| `Content-Length` absent | Remaining bytes are body | Same |
| Truncated chunked body | Error | Return decoded chunks so far, `Partial = true`, warn |

### CR-2: Content-Length is advisory

In lenient mode `Content-Length` is treated as a hint, not a hard limit. The
parser always reads every available byte:

- `actual < declared` → `Partial = true` (message was probably truncated in transit)
- `actual > declared` → `Partial = false` (header is wrong but all data is present)
- `actual == declared` → no warning, no `Partial`

## Warning format

Every warning is a plain string. Warnings that can be attributed to a specific
line carry a `"line N: "` prefix; body-level warnings do not.

```
line 1: missing HTTP version in request-line, defaulting to HTTP/1.1
line 1: absolute-form request-target: extracted Host "example.com:8080", using path "/api"
line 3: bare host:port "example.com:8080" treated as implicit Host header
line 4: whitespace before colon in header name "Content-Type ", accepted leniently
Content-Length declared 100, actual body is 42 bytes
chunked encoding error: unexpected EOF, returning available data
```

## Convenience helpers

The `Host` header value returned by `UnmarshalLenient` may include a port
(`example.com:8080`) or may be a bare hostname (`example.com`). Future
helpers will make it easy to work with these values:

```go
// Planned — not yet implemented.
// SplitHostPort splits a Host header value into hostname and port.
// Port is empty when no port is present.
//   "example.com:8080" → ("example.com", "8080")
//   "example.com"      → ("example.com", "")
func SplitHostPort(hostport string) (host, port string)
```

Until then, use `Headers.Get("Host")` to retrieve the raw value.
