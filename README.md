# shape-http

![Build Status](https://github.com/shapestone/shape-http/actions/workflows/ci.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/shapestone/shape-http)](https://goreportcard.com/report/github.com/shapestone/shape-http)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![codecov](https://codecov.io/gh/shapestone/shape-http/branch/main/graph/badge.svg)](https://codecov.io/gh/shapestone/shape-http)
![Go Version](https://img.shields.io/github/go-mod/go-version/shapestone/shape-http)
![Latest Release](https://img.shields.io/github/v/release/shapestone/shape-http)
[![GoDoc](https://pkg.go.dev/badge/github.com/shapestone/shape-http.svg)](https://pkg.go.dev/github.com/shapestone/shape-http)

[![CodeQL](https://github.com/shapestone/shape-http/actions/workflows/codeql.yml/badge.svg)](https://github.com/shapestone/shape-http/actions/workflows/codeql.yml)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/shapestone/shape-http/badge)](https://securityscorecards.dev/viewer/?uri=github.com/shapestone/shape-http)
[![Security Policy](https://img.shields.io/badge/Security-Policy-brightgreen)](SECURITY.md)

**Repository:** github.com/shapestone/shape-http

A high-performance HTTP/1.1 parser and marshaler for the [Shape Parser™](https://github.com/shapestone/shape) ecosystem.

Parses and serializes HTTP/1.1 messages (RFC 7230/7231) with a dual-path architecture: a fast-path encoder that is **22x faster** than `net/http` for writes, and a fast-path parser that is **3x faster** for reads.

## Installation

```bash
go get github.com/shapestone/shape-http
```

## Import

```go
import shaphttp "github.com/shapestone/shape-http/pkg/http"
```

> **Why the alias?** The package is named `http`, which collides with Go's standard
> library `net/http`. When both are needed in the same file, use an alias to
> distinguish them:
>
> ```go
> import (
>     "net/http"                                           // stdlib — ResponseWriter, ServeMux, etc.
>     shaphttp "github.com/shapestone/shape-http/pkg/http" // shape-http — wire format parsing
> )
> ```
>
> When only shape-http is needed (e.g. a standalone parser or proxy), no alias is
> required and you can import it directly as `http`.

## Usage

### Marshal: shape-http types → HTTP wire format

```go
import shaphttp "github.com/shapestone/shape-http/pkg/http"

req := &shaphttp.Request{
    Method:  "GET",
    Path:    "/api/users",
    Version: "HTTP/1.1",
    Headers: shaphttp.Headers{
        {Key: "Host", Value: "example.com"},
        {Key: "Accept", Value: "application/json"},
    },
}

data, err := shaphttp.Marshal(req)
// GET /api/users HTTP/1.1\r\n
// Host: example.com\r\n
// Accept: application/json\r\n
// \r\n

resp := &shaphttp.Response{
    Version:    "HTTP/1.1",
    StatusCode: 200,
    Reason:     "OK",
    Headers:    shaphttp.Headers{{Key: "Content-Type", Value: "application/json"}},
    Body:       []byte(`{"ok":true}`),
}
data, err = shaphttp.Marshal(resp)
// HTTP/1.1 200 OK\r\n
// Content-Type: application/json\r\n
// Content-Length: 11\r\n
// \r\n
// {"ok":true}
```

### Unmarshal: HTTP wire format → shape-http types

```go
import shaphttp "github.com/shapestone/shape-http/pkg/http"

// Parse an HTTP request
raw := []byte("GET /path HTTP/1.1\r\nHost: example.com\r\n\r\n")
req, err := shaphttp.UnmarshalRequest(raw)
// req is *shaphttp.Request{Method: "GET", Path: "/path", Version: "HTTP/1.1", ...}

// Parse an HTTP response
rawResp := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\nHello")
resp, err := shaphttp.UnmarshalResponse(rawResp)
// resp is *shaphttp.Response{StatusCode: 200, Reason: "OK", Body: []byte("Hello"), ...}
```

### Chunked Transfer Encoding

```go
// Chunked responses are decoded automatically
chunkedResp := []byte(
    "HTTP/1.1 200 OK\r\n" +
    "Transfer-Encoding: chunked\r\n\r\n" +
    "5\r\nHello\r\n6\r\nWorld!\r\n0\r\n\r\n",
)
resp, err := shaphttp.UnmarshalResponse(chunkedResp)
// resp.Body == []byte("HelloWorld!")
```

### Lenient Mode

```go
// Best-effort parsing of malformed or non-RFC-compliant HTTP.
// Never returns an error — extracts what it can and reports issues as warnings.
raw := []byte("GET /path\nHost: example.com\n\n") // bare LF, missing version
result := shaphttp.UnmarshalLenient(raw)
// result.Request.Method  == "GET"
// result.Request.Version == "HTTP/1.1"  (defaulted)
// result.Warnings        == ["line 1: missing HTTP version in request-line, defaulting to HTTP/1.1"]
// result.Partial         == false
```

### Streaming API

```go
// Encode to an io.Writer
encoder := shaphttp.NewEncoder(os.Stdout)
encoder.Encode(req)

// Decode from an io.Reader
decoder := shaphttp.NewDecoder(conn)
req, err := decoder.DecodeRequest()
resp, err := decoder.DecodeResponse()
```

### Validation

```go
// Validate raw bytes against RFC 9112
if err := shaphttp.Validate(string(raw)); err != nil {
    log.Printf("Invalid HTTP message: %v", err)
}
```

## Performance

Benchmarks run on Apple M1 Max (arm64), Go 1.23, `-count=5 -benchmem`:

### Write (Marshal)

| Benchmark | shape-http | net/http | Speedup |
|-----------|-----------|----------|---------|
| Simple request | **60 ns** / 1 alloc | 1336 ns / 8 allocs | **22x** |
| Request + body | **84 ns** / 1 alloc | 1701 ns / 10 allocs | **20x** |
| Simple response | **65 ns** / 1 alloc | 413 ns / 5 allocs | **6.3x** |

### Read (Unmarshal)

| Benchmark | shape-http | net/http | Speedup |
|-----------|-----------|----------|---------|
| Simple request | **433 ns** / 8 allocs | 1449 ns / 12 allocs | **3.3x** |
| Request + body | **487 ns** / 9 allocs | 1612 ns / 14 allocs | **3.3x** |
| Simple response | **465 ns** / 8 allocs | 1368 ns / 13 allocs | **2.9x** |
| Chunked response | **428 ns** / 7 allocs | 1393 ns / 13 allocs | **3.3x** |
| Round-trip req+body | **594 ns** / 10 allocs | 3313 ns / 24 allocs | **5.6x** |

Run benchmarks yourself:

```bash
go test -bench=. -benchmem -count=5 ./pkg/http/
```

## Architecture

shape-http uses a dual-path architecture:

- **Fast-path encoder**: Zero-copy buffer pool (`sync.Pool`) with direct byte appending. Pre-computes static header bytes. Single allocation per encode call.
- **Fast-path parser**: Hand-written state machine parser that avoids `bufio.Reader` overhead. Processes headers in a single pass.
- **Lenient parser**: Accepts common HTTP variations (LF-only endings, missing version, etc.) for debugging or proxy use cases.
- **Shape AST integration**: Full integration with [shape-core](https://github.com/shapestone/shape-core)'s universal AST for structured inspection.

## RFC Compliance

shape-http implements:

- [RFC 7230](https://datatracker.ietf.org/doc/html/rfc7230) — HTTP/1.1: Message Syntax and Routing
- [RFC 7231](https://datatracker.ietf.org/doc/html/rfc7231) — HTTP/1.1: Semantics and Content
- Chunked transfer encoding (RFC 7230 §4.1)
- Trailer fields (RFC 7230 §4.4)

## Related Projects

- **[shape](https://github.com/shapestone/shape)** — Multi-format parser ecosystem
- **[shape-core](https://github.com/shapestone/shape-core)** — Universal AST and tokenizer framework
- **[shape-json](https://github.com/shapestone/shape-json)** — JSON parser
- **[shape-yaml](https://github.com/shapestone/shape-yaml)** — YAML parser
- **[shape-xml](https://github.com/shapestone/shape-xml)** — XML parser
- **[shape-csv](https://github.com/shapestone/shape-csv)** — CSV parser

## License

Apache 2.0 — see [LICENSE](LICENSE) for details.
