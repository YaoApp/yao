//go:build integration

package context_test

import (
	stdContext "context"
	"testing"

	"github.com/stretchr/testify/assert"
	v8 "github.com/yaoapp/gou/runtime/v8"
	agentctx "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestJsValueSendWithBlockID(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Accept = agentctx.AcceptStandard
	cxt.Locale = "en"
	cxt.Writer = newTestMockResponseWriter()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const blockId = ctx.BlockID();
				
				const msg1 = ctx.Send("Message 1", blockId);
				const msg2 = ctx.Send("Message 2", blockId);
				const msg3 = ctx.Send("Message 3", blockId);
				
				const msg4 = ctx.Send({
					type: "text",
					props: { content: "Message 4" },
					block_id: "B_custom"
				}, blockId);
				
				return { 
					success: true,
					msg1: msg1,
					msg2: msg2,
					msg3: msg3,
					msg4: msg4,
					blockId: blockId
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}
	assert.Equal(t, true, result["success"], "Send with blockId should succeed")
}
