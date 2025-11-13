package context

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
	"rogchap.com/v8go"
)

// TestJsValue test the JsValue function
func TestJsValue(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	cxt := &Context{
		ChatID:      "ChatID-123456",
		AssistantID: "AssistantID-1234",
		Sid:         "Sid-1234",
	}

	v8.RegisterFunction("testContextJsvalue", testContextJsvalueEmbed)
	res, err := v8.Call(v8.CallOptions{}, `
		function test(cxt) {
			return testContextJsvalue(cxt)
		}`, cxt)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
	assert.Equal(t, "ChatID-123456", res)
}

func testContextJsvalueEmbed(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, testContextJsvalueFunction)
}

func testContextJsvalueFunction(info *v8go.FunctionCallbackInfo) *v8go.Value {
	var args = info.Args()
	if len(args) < 1 {
		return bridge.JsException(info.Context(), "Missing parameters")
	}

	ctx, err := args[0].AsObject()
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	chatID, err := ctx.Get("ChatID")
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	return chatID
}
