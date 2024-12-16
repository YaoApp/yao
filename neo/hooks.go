package neo

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/neo/assistant"
)

// HookCreate create the assistant
func (neo *DSL) HookCreate(ctx Context, messages []map[string]interface{}, c *gin.Context) (CreateResponse, error) {

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

// HookAssistants query the assistant list from the assistant list hook
func (neo *DSL) HookAssistants(ctx context.Context, param assistant.QueryParam) ([]assistant.Assistant, error) {
	if neo.AssistantListHook == "" {
		return nil, nil
	}

	// Create a context with 10 second timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	p, err := process.Of(neo.AssistantListHook, param)
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
		return nil, nil
	}

	var list []assistant.Assistant
	bytes, err := jsoniter.Marshal(value)
	if err != nil {
		return nil, err
	}

	err = jsoniter.Unmarshal(bytes, &list)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// HookPrepare executes the prepare hook before AI is called
func (neo *DSL) HookPrepare(ctx Context, messages []map[string]interface{}) ([]map[string]interface{}, error) {
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
func (neo *DSL) HookWrite(ctx Context, messages []map[string]interface{}, response map[string]interface{}, content string, writer *gin.ResponseWriter) ([]map[string]interface{}, error) {
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

// HookMention query the mention list
func (neo *DSL) HookMention(ctx context.Context, keywords string) ([]Mention, error) {

	// Default Get the assistant list
	if neo.MentionHook == "" {
		var mentions []Mention
		assistants := neo.GetAssistants()
		for _, assistant := range assistants {
			mentions = append(mentions, Mention{
				ID:   assistant.ID,
				Name: assistant.Name,
				Type: "assistant",
			})
		}

		return mentions, nil
	}

	// Create a context with 10 second timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	p, err := process.Of(neo.MentionHook, keywords)
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
		return nil, nil
	}

	var list []Mention
	bytes, err := jsoniter.Marshal(value)
	if err != nil {
		return nil, err
	}

	err = jsoniter.Unmarshal(bytes, &list)
	if err != nil {
		return nil, err
	}

	return list, nil
}
