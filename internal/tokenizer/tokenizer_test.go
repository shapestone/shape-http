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
