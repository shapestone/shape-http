# Contributing to shape-http

Thank you for your interest in contributing to shape-http! This document provides guidelines for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [How to Contribute](#how-to-contribute)
- [Development Setup](#development-setup)
- [Pull Request Process](#pull-request-process)
- [Testing Guidelines](#testing-guidelines)

## Code of Conduct

This project adheres to the Contributor Covenant [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report unacceptable behavior to conduct@shapestone.com.

## How to Contribute

### Types of Contributions Welcome

1. **Bug Fixes:** Fix parsing errors, incorrect behavior, RFC non-compliance
2. **Performance Improvements:** Optimize parsing, reduce allocations
3. **Documentation:** Improve guides, examples, API docs
4. **Test Coverage:** Add tests for edge cases, malformed inputs
5. **Tooling:** Improve CI/CD, development tools
6. **Examples:** Add usage examples

### Types of Contributions We Generally Don't Accept

1. **Breaking API Changes:** For v1.x releases (semver)
2. **Scope Creep:** Features outside shape-http's core mission (HTTP/1.1 parsing and marshaling)
3. **New Protocol Support:** HTTP/2 or HTTP/3 support (separate projects)

## Development Setup

### Quick Setup

```bash
# Clone repository
git clone https://github.com/shapestone/shape-http.git
cd shape-http

# Run tests
go test -race ./...

# Run linter
golangci-lint run

# Check coverage
go test -cover ./...

# Run benchmarks
go test -bench=. -benchmem ./pkg/http/
```

### Dependencies

shape-http uses a go.work workspace for local development with shape-core. If you are working on shape-core simultaneously, set up a workspace:

```bash
go work init
go work use .
go work use ../shape-core  # if cloned alongside
```

## Pull Request Process

1. **Fork the repository** and create a feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes:**
   - Write clean, documented code
   - Add tests for new functionality
   - Update documentation as needed

3. **Run tests and linting:**
   ```bash
   go test -race ./...
   golangci-lint run
   ```

4. **Commit with clear messages** using [Conventional Commits](https://www.conventionalcommits.org/):
   ```bash
   git commit -m "feat: add support for X"
   git commit -m "fix: resolve issue with Y"
   git commit -m "docs: update parser guide"
   git commit -m "perf: reduce allocations in header parsing"
   ```

   Prefixes:
   - `feat:` - New feature
   - `fix:` - Bug fix
   - `docs:` - Documentation changes
   - `test:` - Test additions/changes
   - `refactor:` - Code refactoring
   - `perf:` - Performance improvements
   - `chore:` - Build process, tooling

5. **Push and create PR:**
   ```bash
   git push origin feature/your-feature-name
   ```

   Then create a pull request on GitHub with:
   - Clear title and description
   - Reference any related issues
   - Benchmark output if performance-related

6. **Code Review:**
   - Maintainers will review your PR
   - Address feedback and make requested changes
   - Once approved, maintainers will merge

## Testing Guidelines

### Test Coverage Requirements

- **New Code:** Must have tests
- **Bug Fixes:** Add test that reproduces the bug
- **Target Coverage:** Maintain 90%+ coverage

### Running Tests

```bash
# All tests with race detector
go test -race ./...

# Specific package
go test -race ./pkg/http/

# With coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# With verbose output
go test -v -race ./...

# Benchmarks
go test -bench=. -benchmem ./pkg/http/

# Compare with net/http stdlib
go test -bench=BenchmarkStdlib -benchmem ./pkg/http/
```

### Writing Good Tests

```go
func TestParseRequest(t *testing.T) {
    // Arrange
    input := "GET /path HTTP/1.1\r\nHost: example.com\r\n\r\n"

    // Act
    req, err := http.ParseRequest([]byte(input))

    // Assert
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }
    if req.Method != "GET" {
        t.Errorf("Expected GET, got %q", req.Method)
    }
}
```

### Performance Expectations

shape-http should be meaningfully faster than net/http. When adding new functionality, run benchmarks before and after to verify there is no regression:

```bash
go test -bench=BenchmarkMarshal -benchmem -count=5 ./pkg/http/
```

## Questions?

- **Issues:** [GitHub Issues](https://github.com/shapestone/shape-http/issues)
- **Discussions:** [GitHub Discussions](https://github.com/shapestone/shape-http/discussions)

Thank you for contributing to shape-http!
