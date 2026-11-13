package computer_test

import (
	"regexp"
	"testing"

	"github.com/yaoapp/yao/modules/computer"
)

func TestNewComputerID(t *testing.T) {
	id := computer.NewComputerID()
	if len(id) != 32 {
		t.Fatalf("computer id length=%d, want 32", len(id))
	}
	if !regexp.MustCompile(`^cmp-[a-z0-9]{28}$`).MatchString(id) {
		t.Fatalf("unexpected computer id: %s", id)
	}
	if computer.NewComputerID() == id {
		t.Fatal("computer ids repeated")
	}
}

func TestNewNodeID(t *testing.T) {
	id := computer.NewNodeID()
	if !regexp.MustCompile(`^node-[a-z0-9]{8}$`).MatchString(id) {
		t.Fatalf("unexpected node id: %s", id)
	}
}
