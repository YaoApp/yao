package assistant

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	chatctx "github.com/yaoapp/yao/neo/context"
	"github.com/yaoapp/yao/neo/message"
)

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

		// input
		if input, has := v["input"]; has {
			raw, _ := jsoniter.MarshalToString(input)
			vv := []message.Message{}
			err := jsoniter.UnmarshalFromString(raw, &vv)
			if err != nil {
				return nil, err
			}
			response.Input = vv
		}

		if res, ok := v["next"].(map[string]interface{}); ok {
			response.Next = &NextAction{}
			if name, ok := res["action"].(string); ok {
				response.Next.Action = name
			}
			if payload, ok := res["payload"].(map[string]interface{}); ok {
				response.Next.Payload = payload
			}
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

// HookStream Handle streaming response from LLM
func (ast *Assistant) HookStream(c *gin.Context, context chatctx.Context, input []message.Message, output []message.Data) (*ResHookStream, error) {

	// Create timeout context
	ctx, cancel := ast.createTimeoutContext(c)
	defer cancel()

	v, err := ast.call(ctx, "Stream", context, input, output, c.Writer)
	if err != nil {
		if err.Error() == HookErrorMethodNotFound {
			return nil, nil
		}
		return nil, err
	}

	response := &ResHookStream{}
	switch v := v.(type) {
	case map[string]interface{}:
		if res, ok := v["output"].(string); ok {
			vv := []message.Data{}
			err := jsoniter.UnmarshalFromString(res, &vv)
			if err != nil {
				return nil, err
			}
			response.Output = vv
		}

		if res, ok := v["output"].([]interface{}); ok {
			vv := []message.Data{}
			raw, _ := jsoniter.MarshalToString(res)
			err := jsoniter.UnmarshalFromString(raw, &vv)
			if err != nil {
				return nil, err
			}
			response.Output = vv
		}

		if res, ok := v["next"].(map[string]interface{}); ok {
			response.Next = &NextAction{}
			if name, ok := res["action"].(string); ok {
				response.Next.Action = name
			}
			if payload, ok := res["payload"].(map[string]interface{}); ok {
				response.Next.Payload = payload
			}
		}

		// Custom silent from hook
		if res, ok := v["silent"].(bool); ok {
			response.Silent = res
		}

	case string:
		vv := []message.Data{}
		err := jsoniter.UnmarshalFromString(v, &vv)
		if err != nil {
			return nil, err
		}
		response.Output = vv
	}

	return response, nil
}

// HookDone Handle completion of assistant response
func (ast *Assistant) HookDone(c *gin.Context, context chatctx.Context, input []message.Message, output []message.Data) (*ResHookDone, error) {
	// Create timeout context
	ctx, cancel := ast.createTimeoutContext(c)
	defer cancel()

	v, err := ast.call(ctx, "Done", context, input, output, c.Writer)
	if err != nil {
		if err.Error() == HookErrorMethodNotFound {
			return nil, nil
		}
		return nil, err
	}

	response := &ResHookDone{
		Input:  input,
		Output: output,
	}

	switch v := v.(type) {
	case map[string]interface{}:
		if res, ok := v["output"].(string); ok {
			vv := []message.Data{}
			err := jsoniter.UnmarshalFromString(res, &vv)
			if err != nil {
				return nil, err
			}
			response.Output = vv
		}

		if res, ok := v["output"].([]interface{}); ok {
			vv := []message.Data{}
			raw, _ := jsoniter.MarshalToString(res)
			err := jsoniter.UnmarshalFromString(raw, &vv)
			if err != nil {
				return nil, err
			}
			response.Output = vv
		}

		if res, ok := v["next"].(map[string]interface{}); ok {
			response.Next = &NextAction{}
			if name, ok := res["action"].(string); ok {
				response.Next.Action = name
			}
			if payload, ok := res["payload"].(map[string]interface{}); ok {
				response.Next.Payload = payload
			}
		}
	case string:
		vv := []message.Data{}
		err := jsoniter.UnmarshalFromString(v, &vv)
		if err != nil {
			return nil, err
		}
		response.Output = vv
	}

	return response, nil
}

// HookFail Handle failure of assistant response
func (ast *Assistant) HookFail(c *gin.Context, context chatctx.Context, input []message.Message, output string, err error) (*ResHookFail, error) {
	// Create timeout context
	ctx, cancel := ast.createTimeoutContext(c)
	defer cancel()

	v, callErr := ast.call(ctx, "Fail", context, input, output, err.Error(), c.Writer)
	if callErr != nil {
		if callErr.Error() == HookErrorMethodNotFound {
			return nil, nil
		}
		return nil, callErr
	}

	response := &ResHookFail{
		Input:  input,
		Output: output,
		Error:  err.Error(),
	}

	switch v := v.(type) {
	case map[string]interface{}:
		if res, ok := v["output"].(string); ok {
			response.Output = res
		}
		if res, ok := v["error"].(string); ok {
			response.Error = res
		}
		if res, ok := v["next"].(map[string]interface{}); ok {
			response.Next = &NextAction{}
			if name, ok := res["action"].(string); ok {
				response.Next.Action = name
			}
			if payload, ok := res["payload"].(map[string]interface{}); ok {
				response.Next.Payload = payload
			}
		}
	case string:
		response.Output = v
	}

	return response, nil
}

// createTimeoutContext creates a timeout context with 5 seconds timeout
func (ast *Assistant) createTimeoutContext(c *gin.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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

	// Call the method directly in the current thread
	args = append([]interface{}{context.Map()}, args...)
	if scriptCtx != nil {
		return scriptCtx.CallWith(ctx, method, args...)
	}
	return nil, nil
}
