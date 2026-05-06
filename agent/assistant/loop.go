package assistant

import (
	"fmt"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/trace/types"
)

// ToolLoopParams holds all parameters needed by executeToolLoop.
type ToolLoopParams struct {
	CompletionMessages []context.Message
	CompletionOptions  *context.CompletionOptions
	CompletionResponse *context.CompletionResponse
	ToolCallResponses  []context.ToolCallResponse
	FullMessages       []context.Message
	AgentNode          types.Node
	StreamHandler      message.StreamFunc
	CreateResponse     *context.HookCreateResponse
	Opts               *context.Options
}

// executeToolLoop feeds tool results back to the LLM in a loop until
// the LLM produces a final text response (no more tool_calls) or
// the maximum number of turns is reached.
//
// Returns the final Response, the last CompletionResponse (for tracing),
// accumulated ToolCallResponses, and any error.
func (ast *Assistant) executeToolLoop(
	ctx *context.Context,
	params *ToolLoopParams,
) (*context.Response, *context.CompletionResponse, []context.ToolCallResponse, error) {

	maxTurns := ast.getMaxToolLoopTurns()
	currentMessages := params.CompletionMessages
	currentCompletion := params.CompletionResponse
	allToolResponses := make([]context.ToolCallResponse, 0, len(params.ToolCallResponses))
	allToolResponses = append(allToolResponses, params.ToolCallResponses...)

	for turn := 0; turn < maxTurns; turn++ {
		ctx.Logger.Debug("Tool loop turn %d/%d", turn+1, maxTurns)

		// Build messages: previous messages + assistant(tool_calls) + tool results
		loopMessages := buildToolLoopMessages(currentMessages, currentCompletion, allToolResponses[len(allToolResponses)-len(params.ToolCallResponses):])

		// Step tracking: LLM call
		ast.BeginStep(ctx, context.StepTypeLLM, map[string]interface{}{
			"messages":  loopMessages,
			"loop_turn": turn + 1,
		})

		// Call LLM with tool results included
		newCompletion, err := ast.executeLLMStream(ctx, loopMessages, params.CompletionOptions, params.AgentNode, params.StreamHandler, params.Opts)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tool loop LLM call failed (turn %d): %w", turn+1, err)
		}

		ast.CompleteStep(ctx, map[string]interface{}{
			"content":    newCompletion.Content,
			"tool_calls": newCompletion.ToolCalls,
		})

		// No tool_calls → LLM gave final text response
		if newCompletion.ToolCalls == nil || len(newCompletion.ToolCalls) == 0 {
			finalResponse := ast.buildStandardResponse(&NextProcessContext{
				Context:            ctx,
				CompletionResponse: newCompletion,
				FullMessages:       params.FullMessages,
				ToolCallResponses:  allToolResponses,
				StreamHandler:      params.StreamHandler,
				CreateResponse:     params.CreateResponse,
			})
			return finalResponse, newCompletion, allToolResponses, nil
		}

		// Has tool_calls → execute them
		ast.BeginStep(ctx, context.StepTypeTool, map[string]interface{}{
			"tool_calls": newCompletion.ToolCalls,
			"loop_turn":  turn + 1,
		})

		toolResults, _ := ast.executeToolCalls(ctx, newCompletion.ToolCalls, 0)

		// Convert ToolCallResult → ToolCallResponse
		toolCallArgsMap := make(map[string]interface{})
		for _, tc := range newCompletion.ToolCalls {
			toolCallArgsMap[tc.ID] = tc.Function.Arguments
		}

		turnResponses := make([]context.ToolCallResponse, len(toolResults))
		for i, result := range toolResults {
			parsedContent, _ := result.ParsedContent()
			turnResponses[i] = context.ToolCallResponse{
				ToolCallID: result.ToolCallID,
				Server:     result.Server(),
				Tool:       result.Tool(),
				Arguments:  toolCallArgsMap[result.ToolCallID],
				Result:     parsedContent,
				Error:      "",
			}
			if result.Error != nil {
				turnResponses[i].Error = result.Error.Error()
			}
		}

		ast.CompleteStep(ctx, map[string]interface{}{
			"results":   turnResponses,
			"loop_turn": turn + 1,
		})

		// Accumulate and prepare next iteration
		allToolResponses = append(allToolResponses, turnResponses...)
		currentMessages = loopMessages
		currentCompletion = newCompletion
		params.ToolCallResponses = turnResponses
	}

	return nil, nil, allToolResponses, fmt.Errorf("tool loop reached max turns (%d)", maxTurns)
}

// buildToolLoopMessages constructs the message sequence for the next LLM call:
// previous messages + assistant message (with tool_calls) + tool result messages.
// Unlike buildToolRetryMessages, this does NOT append a retry system prompt.
func buildToolLoopMessages(
	previousMessages []context.Message,
	completion *context.CompletionResponse,
	toolResponses []context.ToolCallResponse,
) []context.Message {
	messages := make([]context.Message, 0, len(previousMessages)+len(toolResponses)+2)
	messages = append(messages, previousMessages...)

	// Assistant message with tool_calls
	messages = append(messages, context.Message{
		Role:             context.RoleAssistant,
		Content:          completion.Content,
		ReasoningContent: completion.ReasoningContent,
		ToolCalls:        completion.ToolCalls,
	})

	// One tool-role message per tool call result
	for _, tr := range toolResponses {
		var content string
		if tr.Error != "" {
			content = fmt.Sprintf("Error: %s", tr.Error)
		} else if tr.Result != nil {
			raw, _ := jsoniter.MarshalToString(tr.Result)
			content = raw
		}
		toolCallID := tr.ToolCallID
		messages = append(messages, context.Message{
			Role:       context.RoleTool,
			Content:    content,
			ToolCallID: &toolCallID,
		})
	}

	return messages
}

// isToolLoopDisabled checks mcp.options.tool_loop.
// Default is enabled (returns false). Only disabled when explicitly set to false.
func (ast *Assistant) isToolLoopDisabled() bool {
	if ast.MCP == nil || ast.MCP.Options == nil {
		return false
	}
	if v, ok := ast.MCP.Options["tool_loop"]; ok {
		if enabled, ok := v.(bool); ok {
			return !enabled
		}
	}
	return false
}

// getMaxToolLoopTurns reads mcp.options.max_turn. Default is 5.
func (ast *Assistant) getMaxToolLoopTurns() int {
	const defaultMaxTurns = 5
	if ast.MCP == nil || ast.MCP.Options == nil {
		return defaultMaxTurns
	}
	if v, ok := ast.MCP.Options["max_turn"]; ok {
		switch n := v.(type) {
		case float64:
			if n > 0 {
				return int(n)
			}
		case int:
			if n > 0 {
				return n
			}
		}
	}
	return defaultMaxTurns
}

// ---------------------------------------------------------------------------
// Fallback: __yao.loop_fallback delegation (used when tool loop fails/maxes out)
// ---------------------------------------------------------------------------

// buildLoopFallbackDelegate constructs a DelegateConfig for __yao.loop_fallback.
// It packages conversation context and tool results into a Markdown user message.
func (ast *Assistant) buildLoopFallbackDelegate(
	ctx *context.Context,
	fullMessages []context.Message,
	completion *context.CompletionResponse,
	toolResults []context.ToolCallResponse,
) *context.DelegateConfig {

	content := buildLoopFallbackMarkdown(fullMessages, toolResults)
	return &context.DelegateConfig{
		AgentID: "__yao.loop_fallback",
		Messages: []context.Message{
			{Role: context.RoleUser, Content: content},
		},
	}
}

// buildLoopFallbackMarkdown formats context into a Markdown string for the fallback agent.
func buildLoopFallbackMarkdown(
	fullMessages []context.Message,
	toolResults []context.ToolCallResponse,
) string {
	var sb strings.Builder

	sb.WriteString("## Assistant Context\n\n")
	for _, msg := range fullMessages {
		if msg.Role == context.RoleSystem {
			if text := messageText(msg); text != "" {
				sb.WriteString(text)
				sb.WriteString("\n\n")
			}
		}
	}

	sb.WriteString("## Conversation\n\n")
	for _, msg := range fullMessages {
		text := messageText(msg)
		switch msg.Role {
		case context.RoleUser:
			if text != "" {
				sb.WriteString(fmt.Sprintf("**User**: %s\n\n", text))
			}
		case context.RoleAssistant:
			if text != "" {
				sb.WriteString(fmt.Sprintf("**Assistant**: %s\n\n", text))
			}
		}
	}

	sb.WriteString("## Tool Results\n\n")
	for _, tr := range toolResults {
		toolName := tr.Tool
		if tr.Server != "" {
			toolName = tr.Server + "." + tr.Tool
		}
		sb.WriteString(fmt.Sprintf("### %s\n", toolName))
		if tr.Error != "" {
			sb.WriteString(fmt.Sprintf("Error: %s\n\n", tr.Error))
		} else {
			raw, _ := jsoniter.MarshalToString(tr.Result)
			sb.WriteString(fmt.Sprintf("```json\n%s\n```\n\n", raw))
		}
	}

	sb.WriteString("---\nPlease answer the user's question based on the above context and tool results.\n")
	sb.WriteString("Respond in the same language as the user.\n")
	return sb.String()
}

// messageText extracts text content from a message's Content field.
// Content can be a string or an array of content parts (multimodal).
func messageText(msg context.Message) string {
	if msg.Content == nil {
		return ""
	}
	if str, ok := msg.Content.(string); ok {
		return str
	}
	if parts, ok := msg.Content.([]interface{}); ok {
		var texts []string
		for _, part := range parts {
			if partMap, ok := part.(map[string]interface{}); ok {
				if partMap["type"] == "text" {
					if text, ok := partMap["text"].(string); ok {
						texts = append(texts, text)
					}
				}
			}
		}
		return strings.Join(texts, "\n")
	}
	return fmt.Sprintf("%v", msg.Content)
}
