package taiid

import (
	"testing"
)

func TestGenerate_Deterministic(t *testing.T) {
	id1, err := Generate("machine-abc", "9100")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	id2, err := Generate("machine-abc", "9100")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if id1 != id2 {
		t.Errorf("same inputs produced different results: %q vs %q", id1, id2)
	}
	if len(id1) < 5 || id1[:4] != "tai-" {
		t.Errorf("result should start with 'tai-', got %q", id1)
	}
}

func TestGenerate_DifferentInputs(t *testing.T) {
	id1, _ := Generate("machine-abc", "9100")
	id2, _ := Generate("machine-abc", "9200")
	id3, _ := Generate("machine-xyz", "9100")

	if id1 == id2 {
		t.Errorf("different nodeID should produce different results: %q == %q", id1, id2)
	}
	if id1 == id3 {
		t.Errorf("different machineID should produce different results: %q == %q", id1, id3)
	}
}

func TestGenerate_EmptyInputs(t *testing.T) {
	if _, err := Generate("", "9100"); err == nil {
		t.Error("empty machineID should return error")
	}
	if _, err := Generate("machine-abc", ""); err == nil {
		t.Error("empty nodeID should return error")
	}
	if _, err := Generate("", ""); err == nil {
		t.Error("both empty should return error")
	}
}
