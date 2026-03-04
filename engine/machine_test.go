package engine

import (
	"strings"
	"testing"
)

func TestGetMachineID_Deterministic(t *testing.T) {
	info1, err := GetMachineID()
	if err != nil {
		t.Fatalf("GetMachineID() returned error: %v", err)
	}

	info2, err := GetMachineID()
	if err != nil {
		t.Fatalf("GetMachineID() second call returned error: %v", err)
	}

	if info1.ID != info2.ID {
		t.Errorf("GetMachineID() not deterministic: %q != %q", info1.ID, info2.ID)
	}
}

func TestGetMachineID_Format(t *testing.T) {
	info, err := GetMachineID()
	if err != nil {
		t.Fatalf("GetMachineID() returned error: %v", err)
	}

	if !strings.HasPrefix(info.ID, "yao-cli-") {
		t.Errorf("ID should have prefix 'yao-cli-', got %q", info.ID)
	}

	// "yao-cli-" (8) + 32 hex chars = 40
	if len(info.ID) != 40 {
		t.Errorf("ID should be 40 chars, got %d: %q", len(info.ID), info.ID)
	}

	if info.Hostname == "" {
		t.Error("Hostname should not be empty")
	}

	if info.Platform == "" {
		t.Error("Platform should not be empty")
	}
}

func TestGetMachineID_NonEmpty(t *testing.T) {
	info, err := GetMachineID()
	if err != nil {
		t.Fatalf("GetMachineID() returned error: %v", err)
	}

	if info.ID == "" {
		t.Error("ID should not be empty")
	}
}
