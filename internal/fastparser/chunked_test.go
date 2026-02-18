package fastparser

import (
	"testing"
)

func TestDechunk_Simple(t *testing.T) {
	data := []byte("5\r\nHello\r\n0\r\n\r\n")
	body, err := Dechunk(data)
	if err != nil {
		t.Fatalf("Dechunk() error = %v", err)
	}
	if string(body) != "Hello" {
		t.Errorf("Dechunk() = %q, want Hello", string(body))
	}
}

func TestDechunk_MultipleChunks(t *testing.T) {
	data := []byte("5\r\nHello\r\n7\r\n, World\r\n0\r\n\r\n")
	body, err := Dechunk(data)
	if err != nil {
		t.Fatalf("Dechunk() error = %v", err)
	}
	if string(body) != "Hello, World" {
		t.Errorf("Dechunk() = %q, want Hello, World", string(body))
	}
}

func TestDechunk_HexUpperCase(t *testing.T) {
	data := []byte("A\r\n0123456789\r\n0\r\n\r\n")
	body, err := Dechunk(data)
	if err != nil {
		t.Fatalf("Dechunk() error = %v", err)
	}
	if string(body) != "0123456789" {
		t.Errorf("Dechunk() = %q, want 0123456789", string(body))
	}
}

func TestDechunk_WithExtension(t *testing.T) {
	data := []byte("5;ext=val\r\nHello\r\n0\r\n\r\n")
	body, err := Dechunk(data)
	if err != nil {
		t.Fatalf("Dechunk() error = %v", err)
	}
	if string(body) != "Hello" {
		t.Errorf("Dechunk() = %q, want Hello", string(body))
	}
}

func TestDechunk_EmptyBody(t *testing.T) {
	data := []byte("0\r\n\r\n")
	body, err := Dechunk(data)
	if err != nil {
		t.Fatalf("Dechunk() error = %v", err)
	}
	if body != nil {
		t.Errorf("Dechunk() = %v, want nil", body)
	}
}

func TestDechunk_InvalidHex(t *testing.T) {
	data := []byte("XYZ\r\nHello\r\n0\r\n\r\n")
	_, err := Dechunk(data)
	if err == nil {
		t.Error("expected error for invalid hex")
	}
}

func TestDechunk_Truncated(t *testing.T) {
	data := []byte("5\r\nHe")
	_, err := Dechunk(data)
	if err == nil {
		t.Error("expected error for truncated chunk")
	}
}

func TestDechunk_BareLF(t *testing.T) {
	data := []byte("5\nHello\n0\n\n")
	body, err := Dechunk(data)
	if err != nil {
		t.Fatalf("Dechunk() error = %v", err)
	}
	if string(body) != "Hello" {
		t.Errorf("Dechunk() = %q, want Hello", string(body))
	}
}

// TestDechunk_LargeHexSize uses a chunk size string longer than 8 hex chars,
// which triggers the padHex path in parseHexSize.
func TestDechunk_LargeHexSize(t *testing.T) {
	// "000000001" is 9 hex chars (> 8), value = 1; body is "X"
	data := []byte("000000001\r\nX\r\n0\r\n\r\n")
	body, err := Dechunk(data)
	if err != nil {
		t.Fatalf("Dechunk() error = %v", err)
	}
	if string(body) != "X" {
		t.Errorf("Dechunk() = %q, want X", string(body))
	}
}

func TestPadHex(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"abc", "0abc"},  // odd length — gets padded
		{"ab", "ab"},     // even length — unchanged
		{"1", "01"},      // single char — padded
		{"12", "12"},     // two chars — unchanged
		{"123", "0123"},  // three chars — padded
		{"", ""},         // empty — even length, unchanged
		{"abcd", "abcd"}, // four chars — unchanged
	}
	for _, tt := range tests {
		got := padHex(tt.input)
		if got != tt.want {
			t.Errorf("padHex(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDechunk_MissingCRLFAfterData(t *testing.T) {
	// Chunk data is present but not followed by CRLF — just more data with no separator
	data := []byte("5\r\nHello!!") // no CRLF after "Hello", but data[9] == '!'
	_, err := Dechunk(data)
	if err == nil {
		t.Error("expected error when CRLF after chunk data is missing")
	}
}

func TestDechunk_UnterminatedSizeLine(t *testing.T) {
	// No line ending at all
	data := []byte("5")
	_, err := Dechunk(data)
	if err == nil {
		t.Error("expected error for unterminated chunk size line")
	}
}

// TestParseHexSize_LongInvalidHex exercises the hex.DecodeString error path in
// parseHexSize — triggered when len(s) > 8 and the string is not valid hex.
func TestParseHexSize_LongInvalidHex(t *testing.T) {
	// "GGGGGGGGGG" is 10 chars (> 8), 'G' is not a valid hex digit
	_, err := parseHexSize("GGGGGGGGGG")
	if err == nil {
		t.Error("expected error for long invalid hex string")
	}
}

// TestDechunk_EmptyData exercises the "unexpected end of data" path.
func TestDechunk_EmptyData(t *testing.T) {
	_, err := Dechunk([]byte{})
	if err == nil {
		t.Error("expected error for empty input")
	}
}
