package random

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestGenerateReturnsValidJobID(t *testing.T) {
	generator := NewIDGenerator()

	id, err := generator.Generate()
	if err != nil {
		t.Fatalf("Generate() gagal: %v", err)
	}

	if !strings.HasPrefix(id, "job-") {
		t.Fatalf("ID tidak memiliki prefix job-: %q", id)
	}

	data, err := hex.DecodeString(strings.TrimPrefix(id, "job-"))
	if err != nil {
		t.Fatalf("bagian acak bukan hexadecimal: %v", err)
	}

	if len(data) != 16 {
		t.Errorf("panjang data = %d byte, ingin 16", len(data))
	}
}

func TestGenerateReturnsDifferentIDs(t *testing.T) {
	generator := NewIDGenerator()

	first, err := generator.Generate()
	if err != nil {
		t.Fatal(err)
	}

	second, err := generator.Generate()
	if err != nil {
		t.Fatal(err)
	}

	if first == second {
		t.Error("dua ID yang dihasilkan sama")
	}
}
