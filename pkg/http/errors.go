package http

import "fmt"

// ParseError represents an error that occurred during HTTP message parsing.
type ParseError struct {
	Message  string // human-readable error message
	Line     int    // 1-indexed line number where error occurred (0 if unknown)
	Position int    // byte offset in input (0 if unknown)
}

// Error implements the error interface.
func (e *ParseError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("http: parse error at line %d: %s", e.Line, e.Message)
	}
	if e.Position > 0 {
		return fmt.Sprintf("http: parse error at position %d: %s", e.Position, e.Message)
	}
	return fmt.Sprintf("http: %s", e.Message)
}

func newParseError(msg string, line int) *ParseError {
	return &ParseError{Message: msg, Line: line}
}

func newParseErrorAtPos(msg string, pos int) *ParseError {
	return &ParseError{Message: msg, Position: pos}
}
