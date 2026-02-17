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
