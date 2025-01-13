package assistant

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	chatctx "github.com/yaoapp/yao/neo/context"
	"github.com/yaoapp/yao/neo/message"
)

const (
	// HookErrorMethodNotFound is the error message for method not found
	HookErrorMethodNotFound = "method not found"
)

// ResHookInit the response of the init hook
type ResHookInit struct {
	AssistantID string                 `json:"assistant_id,omitempty"`
	ChatID      string                 `json:"chat_id,omitempty"`
	Next        *NextAction            `json:"next,omitempty"`
	Input       []message.Message      `json:"input,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
}

// NextAction the next action
type NextAction struct {
	Action  string                 `json:"action"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

// HookInit initialize the assistant
func (ast *Assistant) HookInit(c *gin.Context, context chatctx.Context, input []message.Message, options map[string]interface{}) (*ResHookInit, error) {
	// Create timeout context
	ctx, cancel := ast.createTimeoutContext(c)
	defer cancel()

	v, err := ast.call(ctx, "Init", context, input, c.Writer)
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

// createTimeoutContext creates a timeout context with 5 seconds timeout
func (ast *Assistant) createTimeoutContext(c *gin.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	return ctx, cancel
}

// Call the script method
func (ast *Assistant) call(ctx context.Context, method string, context chatctx.Context, args ...any) (interface{}, error) {
	if ast.Script == nil {
		return nil, nil
	}

	scriptCtx, err := ast.Script.NewContext(context.Sid, nil)
	if err != nil {
		return nil, err
	}
	defer scriptCtx.Close()

	// Check if the method exists
	if !scriptCtx.Global().Has(method) {
		return nil, fmt.Errorf(HookErrorMethodNotFound)
	}

	// Create done channel for handling cancellation
	done := make(chan struct{})
	var result interface{}
	var callErr error

	go func() {
		defer close(done)
		// Call the method
		args = append([]interface{}{context.Map()}, args...)
		result, callErr = scriptCtx.Call(method, args...)
	}()

	// Wait for either context cancellation or method completion
	select {
	case <-ctx.Done():
		scriptCtx.Close() // Force close the script context
		return nil, ctx.Err()
	case <-done:
		return result, callErr
	}
}
