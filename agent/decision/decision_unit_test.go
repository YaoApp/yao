//go:build unit

package decision_test

import (
	"testing"

	"github.com/yaoapp/gou/llm"
	"github.com/yaoapp/yao/agent/decision"
	"github.com/yaoapp/yao/llmprovider"
)

func TestCall_NilGlobal(t *testing.T) {
	saved := llmprovider.Global
	llmprovider.Global = nil
	defer func() { llmprovider.Global = saved }()

	_, err := decision.Call(&llm.DecisionRequest{State: "test"})
	if err == nil {
		t.Fatal("expected error when Global is nil")
	}
}

func TestCallByUser_NilGlobal(t *testing.T) {
	saved := llmprovider.Global
	llmprovider.Global = nil
	defer func() { llmprovider.Global = saved }()

	_, err := decision.CallByUser("user1", &llm.DecisionRequest{State: "test"})
	if err == nil {
		t.Fatal("expected error when Global is nil")
	}
}
