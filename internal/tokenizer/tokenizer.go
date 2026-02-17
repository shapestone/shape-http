package tokenizer

import (
	"github.com/shapestone/shape-core/pkg/tokenizer"
)

// NewTokenizer creates a tokenizer for HTTP format.
// HTTP is line-oriented, so the tokenizer uses matchers that work at the line level:
// 1. CRLF (line endings)
// 2. SP (space separator)
// 3. Colon (header separator)
// 4. HTTP version string
// 5. Generic text (method, path, header names/values, etc.)
//
// Note: Unlike JSON, HTTP doesn't use the default whitespace skipper because
// spaces and line endings are semantically significant.
func NewTokenizer() tokenizer.Tokenizer {
	return tokenizer.NewTokenizerWithoutWhitespace(
		// CRLF first (highest priority - line endings are structural)
		CRLFMatcher(),

		// Space separator
		SPMatcher(),

		// Colon (header separator)
		tokenizer.StringMatcherFunc(TokenHeaderColon, ":"),

		// HTTP version (before generic text)
		VersionMatcher(),

		// Generic text token (everything else until SP, CRLF, or colon)
		TextMatcher(),
	)
}

// NewTokenizerWithStream creates a tokenizer for HTTP format using a pre-configured stream.
func NewTokenizerWithStream(stream tokenizer.Stream) tokenizer.Tokenizer {
	tok := NewTokenizer()
	tok.InitializeFromStream(stream)
	return tok
}

// CRLFMatcher matches \r\n or bare \n.
func CRLFMatcher() tokenizer.Matcher {
	return func(stream tokenizer.Stream) *tokenizer.Token {
		r, ok := stream.PeekChar()
		if !ok {
			return nil
		}

		if r == '\r' {
			value := []rune{'\r'}
			stream.NextChar()
			// Check for \n after \r
			r2, ok := stream.PeekChar()
			if ok && r2 == '\n' {
				stream.NextChar()
				value = append(value, '\n')
			}
			return tokenizer.NewToken(TokenCRLF, value)
		}
		if r == '\n' {
			stream.NextChar()
			return tokenizer.NewToken(TokenCRLF, []rune{'\n'})
		}
		return nil
	}
}

// SPMatcher matches a single space character.
func SPMatcher() tokenizer.Matcher {
	return func(stream tokenizer.Stream) *tokenizer.Token {
		r, ok := stream.PeekChar()
		if !ok {
			return nil
		}
		if r == ' ' {
			stream.NextChar()
			return tokenizer.NewToken(TokenSP, []rune{' '})
		}
		return nil
	}
}

// VersionMatcher matches "HTTP/" followed by digits and dot.
func VersionMatcher() tokenizer.Matcher {
	return func(stream tokenizer.Stream) *tokenizer.Token {
		// Try to match "HTTP/"
		prefix := []rune("HTTP/")
		var value []rune

		for _, expected := range prefix {
			r, ok := stream.PeekChar()
			if !ok || r != expected {
				return nil
			}
			stream.NextChar()
			value = append(value, r)
		}

		// Match version number (digits and dots)
		for {
			r, ok := stream.PeekChar()
			if !ok {
				break
			}
			if (r >= '0' && r <= '9') || r == '.' {
				stream.NextChar()
				value = append(value, r)
			} else {
				break
			}
		}

		return tokenizer.NewToken(TokenVersion, value)
	}
}

// TextMatcher matches any sequence of characters until SP, CRLF, colon, or EOS.
// This is used for methods, paths, header names, header values, etc.
func TextMatcher() tokenizer.Matcher {
	return func(stream tokenizer.Stream) *tokenizer.Token {
		var value []rune

		for {
			r, ok := stream.PeekChar()
			if !ok {
				break
			}
			if r == ' ' || r == '\r' || r == '\n' || r == ':' {
				break
			}
			stream.NextChar()
			value = append(value, r)
		}

		if len(value) == 0 {
			return nil
		}

		return tokenizer.NewToken("Text", value)
	}
}

// HeaderValueMatcher matches everything after the colon until CRLF.
// This includes spaces and colons within the value.
func HeaderValueMatcher() tokenizer.Matcher {
	return func(stream tokenizer.Stream) *tokenizer.Token {
		var value []rune

		for {
			r, ok := stream.PeekChar()
			if !ok {
				break
			}
			if r == '\r' || r == '\n' {
				break
			}
			stream.NextChar()
			value = append(value, r)
		}

		if len(value) == 0 {
			return nil
		}

		return tokenizer.NewToken(TokenHeaderValue, value)
	}
}
