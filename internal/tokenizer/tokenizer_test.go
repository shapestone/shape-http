package tokenizer

import (
	"testing"

	coretok "github.com/shapestone/shape-core/pkg/tokenizer"
)

func TestTokenize_RequestLine(t *testing.T) {
	tok := NewTokenizer()
	tok.Initialize("GET /api HTTP/1.1\r\n")

	tokens, eos := tok.Tokenize()
	if !eos {
		t.Error("expected EOS")
	}

	// Expect: Text("GET"), SP, Text("/api"), SP, Version("HTTP/1.1"), CRLF
	expected := []struct {
		kind  string
		value string
	}{
		{"Text", "GET"},
		{TokenSP, " "},
		{"Text", "/api"},
		{TokenSP, " "},
		{TokenVersion, "HTTP/1.1"},
		{TokenCRLF, "\r\n"},
	}

	if len(tokens) != len(expected) {
		t.Fatalf("token count = %d, want %d. tokens = %v", len(tokens), len(expected), formatTokens(tokens))
	}

	for i, exp := range expected {
		if tokens[i].Kind() != exp.kind {
			t.Errorf("token[%d].Kind() = %q, want %q", i, tokens[i].Kind(), exp.kind)
		}
		if tokens[i].ValueString() != exp.value {
			t.Errorf("token[%d].Value() = %q, want %q", i, tokens[i].ValueString(), exp.value)
		}
	}
}

func TestTokenize_HeaderLine(t *testing.T) {
	tok := NewTokenizer()
	tok.Initialize("Host: example.com\r\n")

	tokens, eos := tok.Tokenize()
	if !eos {
		t.Error("expected EOS")
	}

	// Expect: Text("Host"), Colon(":"), SP, Text("example.com"), CRLF
	if len(tokens) < 4 {
		t.Fatalf("token count = %d, want >= 4. tokens = %v", len(tokens), formatTokens(tokens))
	}

	if tokens[0].Kind() != "Text" || tokens[0].ValueString() != "Host" {
		t.Errorf("token[0] = %v, want Text('Host')", tokens[0])
	}
	if tokens[1].Kind() != TokenHeaderColon {
		t.Errorf("token[1] = %v, want Colon", tokens[1])
	}
}

func TestTokenize_BareLF(t *testing.T) {
	tok := NewTokenizer()
	tok.Initialize("GET /\n")

	tokens, eos := tok.Tokenize()
	if !eos {
		t.Error("expected EOS")
	}

	// Last token should be CRLF with bare LF
	found := false
	for _, tok := range tokens {
		if tok.Kind() == TokenCRLF {
			found = true
		}
	}
	if !found {
		t.Error("expected CRLF token for bare LF")
	}
}

func TestNewTokenizerWithStream(t *testing.T) {
	stream := coretok.NewStream("GET /api HTTP/1.1\r\n")
	tok := NewTokenizerWithStream(stream)

	tokens, eos := tok.Tokenize()
	if !eos {
		t.Error("expected EOS")
	}
	if len(tokens) == 0 {
		t.Fatal("expected tokens, got none")
	}

	// First token should be Text("GET")
	if tokens[0].Kind() != "Text" || tokens[0].ValueString() != "GET" {
		t.Errorf("tokens[0] = %v, want Text('GET')", tokens[0])
	}
}

func TestHeaderValueMatcher_Basic(t *testing.T) {
	matcher := HeaderValueMatcher()
	stream := coretok.NewStream("application/json\r\n")

	tok := matcher(stream)
	if tok == nil {
		t.Fatal("expected token, got nil")
	}
	if tok.Kind() != TokenHeaderValue {
		t.Errorf("Kind = %q, want %q", tok.Kind(), TokenHeaderValue)
	}
	if tok.ValueString() != "application/json" {
		t.Errorf("Value = %q, want application/json", tok.ValueString())
	}
}

func TestHeaderValueMatcher_WithColonInValue(t *testing.T) {
	// Header values can contain colons (e.g., URLs)
	matcher := HeaderValueMatcher()
	stream := coretok.NewStream("https://example.com/path\r\n")

	tok := matcher(stream)
	if tok == nil {
		t.Fatal("expected token, got nil")
	}
	if tok.ValueString() != "https://example.com/path" {
		t.Errorf("Value = %q, want https://example.com/path", tok.ValueString())
	}
}

func TestHeaderValueMatcher_Empty(t *testing.T) {
	// Empty value (immediately hits \r\n) → nil token
	matcher := HeaderValueMatcher()
	stream := coretok.NewStream("\r\n")

	tok := matcher(stream)
	if tok != nil {
		t.Errorf("expected nil token for empty value, got %v", tok)
	}
}

func TestHeaderValueMatcher_BareLF(t *testing.T) {
	// Bare LF also terminates the value
	matcher := HeaderValueMatcher()
	stream := coretok.NewStream("text/html\n")

	tok := matcher(stream)
	if tok == nil {
		t.Fatal("expected token, got nil")
	}
	if tok.ValueString() != "text/html" {
		t.Errorf("Value = %q, want text/html", tok.ValueString())
	}
}

func TestHeaderValueMatcher_EOS(t *testing.T) {
	// Stream at EOS — PeekChar returns false → break → len(value)==0 → return nil
	matcher := HeaderValueMatcher()
	stream := coretok.NewStream("")
	tok := matcher(stream)
	if tok != nil {
		t.Errorf("expected nil for EOS stream, got %v", tok)
	}
}

func TestCRLFMatcher_EOS(t *testing.T) {
	// Stream at EOS — PeekChar returns false → return nil
	matcher := CRLFMatcher()
	stream := coretok.NewStream("")
	tok := matcher(stream)
	if tok != nil {
		t.Errorf("expected nil for EOS stream, got %v", tok)
	}
}

func TestCRLFMatcher_NonCRLF(t *testing.T) {
	// First char is not \r or \n — return nil
	matcher := CRLFMatcher()
	stream := coretok.NewStream("GET /")
	tok := matcher(stream)
	if tok != nil {
		t.Errorf("expected nil for non-CRLF char, got %v", tok)
	}
}

func TestCRLFMatcher_BareCR(t *testing.T) {
	// Bare CR (\r not followed by \n) — returns token with just \r
	matcher := CRLFMatcher()
	stream := coretok.NewStream("\rGET")
	tok := matcher(stream)
	if tok == nil {
		t.Fatal("expected token for bare CR, got nil")
	}
	if tok.Kind() != TokenCRLF {
		t.Errorf("Kind = %q, want %q", tok.Kind(), TokenCRLF)
	}
}

func TestSPMatcher_EOS(t *testing.T) {
	// Stream at EOS — PeekChar returns false → return nil
	matcher := SPMatcher()
	stream := coretok.NewStream("")
	tok := matcher(stream)
	if tok != nil {
		t.Errorf("expected nil for EOS stream, got %v", tok)
	}
}

func TestSPMatcher_NonSP(t *testing.T) {
	// First char is not a space — return nil
	matcher := SPMatcher()
	stream := coretok.NewStream("X")
	tok := matcher(stream)
	if tok != nil {
		t.Errorf("expected nil for non-SP char, got %v", tok)
	}
}

func TestVersionMatcher_EOS(t *testing.T) {
	// Stream at EOS — PeekChar returns false → return nil
	matcher := VersionMatcher()
	stream := coretok.NewStream("")
	tok := matcher(stream)
	if tok != nil {
		t.Errorf("expected nil for EOS stream, got %v", tok)
	}
}

func TestVersionMatcher_NonHTTP(t *testing.T) {
	// Doesn't start with HTTP/ — return nil
	matcher := VersionMatcher()
	stream := coretok.NewStream("GET /")
	tok := matcher(stream)
	if tok != nil {
		t.Errorf("expected nil for non-HTTP/ prefix, got %v", tok)
	}
}

func TestVersionMatcher_VersionNumberEOS(t *testing.T) {
	// "HTTP/" followed by EOS — returns token with just "HTTP/"
	matcher := VersionMatcher()
	stream := coretok.NewStream("HTTP/")
	tok := matcher(stream)
	if tok == nil {
		t.Fatal("expected token for HTTP/ prefix, got nil")
	}
	if tok.Kind() != TokenVersion {
		t.Errorf("Kind = %q, want %q", tok.Kind(), TokenVersion)
	}
}

func TestTextMatcher_EOS(t *testing.T) {
	// Empty stream — PeekChar returns false → len(value)==0 → return nil
	matcher := TextMatcher()
	stream := coretok.NewStream("")
	tok := matcher(stream)
	if tok != nil {
		t.Errorf("expected nil for EOS stream, got %v", tok)
	}
}

func TestTextMatcher_StartWithStopChar(t *testing.T) {
	// First char is a stop char (colon) — len(value)==0 → return nil
	matcher := TextMatcher()
	stream := coretok.NewStream(": value")
	tok := matcher(stream)
	if tok != nil {
		t.Errorf("expected nil when starting with colon, got %v", tok)
	}
}

func formatTokens(tokens []coretok.Token) string {
	s := "["
	for i, t := range tokens {
		if i > 0 {
			s += ", "
		}
		s += t.String()
	}
	s += "]"
	return s
}
