package hook_test

import (
	"context"
	"testing"

	"github.com/yaoapp/yao/agent/assistant"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/testutils"
)

// TestCreate test the create hook
func TestCreate(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.create")
	if err != nil {
		t.Fatalf("Failed to get the tests.create assistant: %s", err.Error())
	}

	if agent.Script == nil {
		t.Fatalf("The tests.create assistant has no script")
	}

	ctx := &agentContext.Context{
		Context:     context.Background(),
		ChatID:      "chat-test-create-hook",
		AssistantID: "tests.create",
		Sid:         "test-session-create-hook",
	}

	agent.Script.Create(ctx, []agentContext.Message{{Role: "user", Content: "Hello, how are you?"}})
}
