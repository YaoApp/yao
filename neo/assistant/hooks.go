package assistant

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/log"
	chatctx "github.com/yaoapp/yao/neo/context"
	"github.com/yaoapp/yao/neo/message"
	chatMessage "github.com/yaoapp/yao/neo/message"
)

// HookCreate create a new assistant
func (ast *Assistant) HookCreate(c *gin.Context, context chatctx.Context, input []chatMessage.Message, options map[string]interface{}, contents *chatMessage.Contents) (*ResHookInit, error) {
	// Create timeout context
	ctx := ast.createBackgroundContext()
	v, err := ast.call(ctx, "Create", c, contents, context, input, options)
	if err != nil {
		if err.Error() == HookErrorMethodNotFound {
			return nil, nil
		}
		return nil, err
	}

	response := &ResHookInit{Result: nil}
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

		// result
		if result, has := v["result"]; has {
			response.Result = result
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
func (ast *Assistant) HookStream(c *gin.Context, context chatctx.Context, input []message.Message, msg *message.Message, contents *chatMessage.Contents) (*ResHookStream, error) {

	// Create timeout context
	ctx, cancel := ast.createTimeoutContext(5 * time.Second)
	defer cancel()

	v, err := ast.call(ctx, "Stream", c, contents, context, input, msg, contents.JSON())
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

// HookRetry Handle retry of assistant response
func (ast *Assistant) HookRetry(c *gin.Context, context chatctx.Context, input []message.Message, contents *chatMessage.Contents, errmsg string) (interface{}, error) {
	ctx := ast.createBackgroundContext()
	output := []message.Data{}
	if len(input) < 1 {
		return "", fmt.Errorf("no input")
	}

	var lastInput message.Message = input[len(input)-1]
	for _, data := range contents.Data {
		if data.Type == "think" {
			continue
		}
		output = append(output, data)
	}

	v, err := ast.call(ctx, "Retry", c, contents, context, lastInput.String(), output, errmsg)
	if err != nil {
		if err.Error() == HookErrorMethodNotFound {
			return nil, nil
		}
		return nil, err
	}

	switch v := v.(type) {
	case string:
		return v, nil
	case map[string]interface{}:
		var next NextAction
		raw, _ := jsoniter.MarshalToString(v)
		err := jsoniter.UnmarshalFromString(raw, &next)
		if err != nil {
			return nil, err
		}
		return &next, nil
	}

	return nil, nil
}

// HookDone Handle completion of assistant response
func (ast *Assistant) HookDone(c *gin.Context, context chatctx.Context, input []message.Message, contents *chatMessage.Contents) (*ResHookDone, error) {
	// Create timeout context
	ctx := ast.createBackgroundContext()

	// format the output
	// 1. Remove thinking message
	// 2. Parse the tool call message content
	output := []message.Data{}
	if contents != nil && contents.Data != nil {
		for _, data := range contents.Data {
			if data.Type == "think" {
				continue
			}

			// parse the tool call message content
			if data.Type == "tool" && data.Props != nil {
				props := map[string]interface{}{}
				if text, ok := data.Props["text"].(string); ok {

					// Extract the content between <tool> and </tool> tags more reliably
					startTag := "<tool>"
					endTag := "</tool>"
					startIndex := strings.Index(text, startTag)
					if startIndex != -1 {
						// Find the content after <tool>
						content := text[startIndex+len(startTag):]
						endIndex := strings.LastIndex(content, endTag)
						if endIndex != -1 {
							// Extract the content between tags
							text = content[:endIndex]
							text = strings.TrimSpace(text)
							if os.Getenv("YAO_AGENT_PRINT_TOOL_CALL") == "true" {
								log.Trace("[TOOL CALL] %s", text)
							}
						}
					}

					// Parse the text into props
					err := ParseJSON(text, &props)
					if err != nil {
						props["error"] = fmt.Sprintf("Can not parse the tool call: %s\n--original--\n%s", err.Error(), text)
					}
				}

				output = append(output, message.Data{Type: "tool", Props: props})
				continue
			}
			output = append(output, data)
		}
	}

	v, err := ast.call(ctx, "Done", c, contents, context, input, output)
	if err != nil {
		if err.Error() == HookErrorMethodNotFound {
			return nil, nil
		}
		return nil, err
	}

	response := &ResHookDone{Input: input, Output: contents.Data}

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

		// has result
		if res, has := v["result"]; has {
			response.Result = res
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
func (ast *Assistant) HookFail(c *gin.Context, context chatctx.Context, input []message.Message, err error, contents *chatMessage.Contents) (*ResHookFail, error) {
	// Create timeout context
	ctx, cancel := ast.createTimeoutContext(5 * time.Second)
	defer cancel()

	v, callErr := ast.call(ctx, "Fail", c, contents, context, input, err.Error())
	if callErr != nil {
		if callErr.Error() == HookErrorMethodNotFound {
			return nil, nil
		}
		return nil, callErr
	}

	response := &ResHookFail{
		Input:  input,
		Output: contents.Text(),
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
func (ast *Assistant) createTimeoutContext(time time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), time)
	return ctx, cancel
}

// createBackgroundContext creates a background context
func (ast *Assistant) createBackgroundContext() context.Context {
	return context.Background()
}

// Call the script method
func (ast *Assistant) call(ctx context.Context, method string, c *gin.Context, contents *chatMessage.Contents, context chatctx.Context, args ...any) (interface{}, error) {
	if ast.Script == nil {
		return nil, nil
	}

	scriptCtx, err := ast.Script.NewContext(context.Sid, nil)
	if err != nil {
		return nil, err
	}
	defer scriptCtx.Close()

	// Initialize the object, add the global variables, methods to the script context
	ast.InitObject(scriptCtx, c, context, contents)

	// Check if the method exists
	if !scriptCtx.Global().Has(method) {
		return nil, fmt.Errorf(HookErrorMethodNotFound)
	}

	// Call the method directly in the current thread
	if scriptCtx != nil {
		return scriptCtx.CallWith(ctx, method, args...)
	}
	return nil, nil
}
