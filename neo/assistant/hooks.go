package assistant

import (
	"fmt"

	chatctx "github.com/yaoapp/yao/neo/context"
	"github.com/yaoapp/yao/neo/message"
)

const (
	// HookErrorMethodNotFound is the error message for method not found
	HookErrorMethodNotFound = "method not found"
)

// ResHookInit the response of the init hook
type ResHookInit struct {
	AssistantID string `json:"assistant_id,omitempty"`
	ChatID      string `json:"chat_id,omitempty"`
}

// HookInit initialize the assistant
func (ast *Assistant) HookInit(context chatctx.Context, messages []message.Message) (*ResHookInit, error) {
	v, err := ast.call("Init", context, messages)
	if err != nil {
		if err.Error() == HookErrorMethodNotFound {
			return nil, nil
		}
		return nil, err
	}

	response := &ResHookInit{}
	switch v := v.(type) {
	case map[string]interface{}:
		if res, ok := v["assistant_id"].(string); ok {
			response.AssistantID = res
		}
		if res, ok := v["chat_id"].(string); ok {
			response.ChatID = res
		}

	case string:
		response.AssistantID = v
		response.ChatID = context.ChatID

	case nil:
		response.AssistantID = ast.ID
		response.ChatID = context.ChatID
	}

	return response, nil
}

// Call the script method
func (ast *Assistant) call(method string, context chatctx.Context, args ...any) (interface{}, error) {

	if ast.Script == nil {
		return nil, nil
	}

	ctx, err := ast.Script.NewContext(context.Sid, nil)
	if err != nil {
		return nil, err
	}
	defer ctx.Close()

	// Check if the method exists
	if !ctx.Global().Has(method) {
		return nil, fmt.Errorf(HookErrorMethodNotFound)
	}

	// Call the method
	args = append([]interface{}{context.Map()}, args...)
	return ctx.Call(method, args...)
}
