package neo

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/process"
	chatctx "github.com/yaoapp/yao/neo/context"
)

// HookCreate create the assistant
func (neo *DSL) HookCreate(ctx chatctx.Context, messages []map[string]interface{}, c *gin.Context) (CreateResponse, error) {

	// Default assistant
	assistantID := neo.Use
	if ctx.AssistantID != "" {
		assistantID = ctx.AssistantID
	}

	// Empty hook
	if neo.Create == "" {
		return CreateResponse{AssistantID: assistantID, ChatID: ctx.ChatID}, nil
	}

	// Create a context with 10 second timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	p, err := process.Of(neo.Create, ctx, messages, c.Writer)
	if err != nil {
		return CreateResponse{}, err
	}

	err = p.WithContext(timeoutCtx).Execute()
	if err != nil {
		return CreateResponse{}, err
	}
	defer p.Release()

	// Check if context was canceled
	if timeoutCtx.Err() != nil {
		return CreateResponse{}, timeoutCtx.Err()
	}

	value := p.Value()
	switch v := value.(type) {
	case CreateResponse:
		return v, nil

	case map[string]interface{}:
		if id, ok := v["assistant_id"].(string); ok {
			assistantID = id
		}

		chatID := ""
		if id, ok := v["chat_id"].(string); ok {
			chatID = id
		}

		if chatID == "" {
			chatID = ctx.ChatID
		}

		return CreateResponse{AssistantID: assistantID, ChatID: chatID}, nil
	}

	return CreateResponse{AssistantID: assistantID, ChatID: ctx.ChatID}, nil
}

// HookPrepare executes the prepare hook before AI is called
func (neo *DSL) HookPrepare(ctx chatctx.Context, messages []map[string]interface{}) ([]map[string]interface{}, error) {
	if neo.Prepare == "" {
		return messages, nil
	}

	// Create a context with 10 second timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	p, err := process.Of(neo.Prepare, ctx, messages)
	if err != nil {
		return nil, err
	}

	err = p.WithContext(timeoutCtx).Execute()
	if err != nil {
		return nil, err
	}
	defer p.Release()

	// Check if context was canceled
	if timeoutCtx.Err() != nil {
		return nil, timeoutCtx.Err()
	}

	value := p.Value()
	if value == nil {
		return messages, nil
	}

	var result []map[string]interface{}
	bytes, err := jsoniter.Marshal(value)
	if err != nil {
		return nil, err
	}

	err = jsoniter.Unmarshal(bytes, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// HookWrite executes the write hook when response is received from AI
func (neo *DSL) HookWrite(ctx chatctx.Context, messages []map[string]interface{}, response map[string]interface{}, content string, writer *gin.ResponseWriter) ([]map[string]interface{}, error) {
	if neo.Write == "" {
		return []map[string]interface{}{response}, nil
	}

	p, err := process.Of(neo.Write, ctx, messages, response, content, writer)
	if err != nil {
		return nil, err
	}

	err = p.WithContext(ctx).Execute()
	if err != nil {
		return nil, err
	}
	defer p.Release()

	value := p.Value()
	if value == nil {
		return []map[string]interface{}{response}, nil
	}

	var result []map[string]interface{}
	bytes, err := jsoniter.Marshal(value)
	if err != nil {
		return nil, err
	}

	err = jsoniter.Unmarshal(bytes, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
