# Changelog

All notable changes to shape-http will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-02-17

### Added
- Initial release of shape-http
- High-performance HTTP/1.1 request and response parser
- Fast-path marshal for `*http.Request` and `*http.Response` (22x faster than net/http for writes)
- Fast-path unmarshal parser (3x faster than net/http for reads)
- Chunked transfer encoding support
- Lenient parsing mode for malformed or non-RFC-compliant HTTP messages
- `Marshal` / `Unmarshal` API compatible with standard library conventions
- `Encode` / `Decode` streaming API
- HTTP validation against RFC 7230 / RFC 7231
- Round-trip fidelity: parse → marshal → parse produces identical results
- Shape AST integration via shape-core
- Zero-allocation fast-path encoder using `sync.Pool` buffer reuse
- `go.mod` module: `github.com/shapestone/shape-http`

[Unreleased]: https://github.com/shapestone/shape-http/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/shapestone/shape-http/releases/tag/v0.1.0
