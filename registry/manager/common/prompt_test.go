package common

import (
	"testing"
)

func TestMockPrompter(t *testing.T) {
	m := &MockPrompter{
		ConfirmResponses: []bool{true, false},
		ChooseResponses:  []int{1, 2},
	}

	if !m.Confirm("install?") {
		t.Error("expected true")
	}
	if m.Confirm("upgrade?") {
		t.Error("expected false")
	}
	if m.Confirm("extra?") != true {
		t.Error("expected default true when responses exhausted")
	}

	if len(m.ConfirmCalls) != 3 {
		t.Errorf("expected 3 confirm calls, got %d", len(m.ConfirmCalls))
	}

	if m.Choose("pick", []string{"a", "b"}) != 1 {
		t.Error("expected 1")
	}
	if m.Choose("pick2", []string{"a", "b", "c"}) != 2 {
		t.Error("expected 2")
	}
	if m.Choose("pick3", nil) != 0 {
		t.Error("expected default 0 when responses exhausted")
	}

	if len(m.ChooseCalls) != 3 {
		t.Errorf("expected 3 choose calls, got %d", len(m.ChooseCalls))
	}
}

func TestAutoConfirmPrompter(t *testing.T) {
	p := &AutoConfirmPrompter{}
	if !p.Confirm("anything") {
		t.Error("expected always true")
	}
	if p.Choose("anything", []string{"a", "b"}) != 0 {
		t.Error("expected always 0")
	}
}
