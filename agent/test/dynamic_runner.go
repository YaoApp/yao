package test

import (
	"fmt"
	"time"

	jsoniter "github.com/json-iterator/go"
	goutext "github.com/yaoapp/gou/text"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
)

// DynamicRunner handles dynamic (simulator-driven) test execution
type DynamicRunner struct {
	opts     *Options
	output   *OutputWriter
	asserter *Asserter
}

// NewDynamicRunner creates a new dynamic runner
func NewDynamicRunner(opts *Options) *DynamicRunner {
	return &DynamicRunner{
		opts:     opts,
		output:   NewOutputWriter(opts.Verbose),
		asserter: NewAsserter(),
	}
}

// RunDynamic executes a dynamic test case
func (r *DynamicRunner) RunDynamic(ast *assistant.Assistant, tc *Case, agentID string) *DynamicResult {
	startTime := time.Now()

	result := &DynamicResult{
		ID:          tc.ID,
		Turns:       make([]*TurnResult, 0),
		Checkpoints: make(map[string]*CheckpointResult),
	}

	// Initialize checkpoints
	for _, cp := range tc.Checkpoints {
		result.Checkpoints[cp.ID] = &CheckpointResult{
			ID:       cp.ID,
			Reached:  false,
			Required: cp.IsRequired(),
		}
	}

	// Get simulator agent
	simAST, err := assistant.Get(tc.Simulator.Use)
	if err != nil {
		result.Status = StatusError
		result.Error = fmt.Sprintf("failed to get simulator agent: %s", err.Error())
		result.DurationMs = time.Since(startTime).Milliseconds()
		return result
	}

	// Get configuration
	maxTurns := tc.GetMaxTurns()
	timeout := tc.GetTimeout(r.opts.Timeout)

	// Build simulator metadata
	simMetadata := make(map[string]interface{})
	if tc.Simulator.Options != nil && tc.Simulator.Options.Metadata != nil {
		for k, v := range tc.Simulator.Options.Metadata {
			simMetadata[k] = v
		}
	}

	// Conversation history
	messages := make([]context.Message, 0)

	// Get initial input if provided
	initialMessages, err := tc.GetMessages()
	if err == nil && len(initialMessages) > 0 {
		messages = append(messages, initialMessages...)
	}

	// Output dynamic test start
	if r.opts.Verbose {
		r.output.Verbose("Dynamic test: %s (max %d turns)", tc.ID, maxTurns)
	}

	// Use consistent chatID across all turns to preserve session state (ctx.memory.chat)
	// Priority: context config > generated ID
	chatID := fmt.Sprintf("dynamic-%s", tc.ID)
	if r.opts.ContextData != nil && r.opts.ContextData.ChatID != "" {
		chatID = r.opts.ContextData.ChatID
	}

	// Conversation loop
	for turn := 1; turn <= maxTurns; turn++ {
		turnStart := time.Now()
		turnResult := &TurnResult{Turn: turn}

		// Check timeout
		if time.Since(startTime) > timeout {
			result.Status = StatusTimeout
			result.Error = fmt.Sprintf("timeout after %s", timeout)
			result.DurationMs = time.Since(startTime).Milliseconds()
			result.TotalTurns = turn - 1
			return result
		}

		// For turns after the first, get input from simulator
		if turn > 1 || len(messages) == 0 {
			simInput := r.buildSimulatorInput(tc, messages, result, turn, maxTurns, simMetadata)
			simOutput, err := r.callSimulator(simAST, tc, simInput)
			if err != nil {
				turnResult.Error = fmt.Sprintf("simulator error: %s", err.Error())
				result.Turns = append(result.Turns, turnResult)
				result.Status = StatusError
				result.Error = turnResult.Error
				result.DurationMs = time.Since(startTime).Milliseconds()
				result.TotalTurns = turn
				return result
			}

			// Check if goal achieved
			if simOutput.GoalAchieved {
				if r.opts.Verbose {
					r.output.Verbose("Turn %d: Simulator signaled goal achieved", turn)
				}

				// Check if all required checkpoints reached
				if r.allRequiredCheckpointsReached(result) {
					result.Status = StatusPassed
				} else {
					result.Status = StatusFailed
					result.Error = "simulator signaled goal achieved but not all required checkpoints reached"
				}
				result.DurationMs = time.Since(startTime).Milliseconds()
				result.TotalTurns = turn - 1
				return result
			}

			// Add user message
			userMessage := context.Message{
				Role:    context.RoleUser,
				Content: simOutput.Message,
			}
			messages = append(messages, userMessage)
			turnResult.Input = simOutput.Message

			if r.opts.Verbose {
				r.output.Verbose("Turn %d: User: %s", turn, truncateOutput(simOutput.Message, 50))
			}
		} else {
			// Use initial input for first turn
			if len(messages) > 0 {
				lastMsg := messages[len(messages)-1]
				turnResult.Input = lastMsg.Content
				if r.opts.Verbose {
					r.output.Verbose("Turn %d: User: %s", turn, truncateOutput(lastMsg.Content, 50))
				}
			}
		}

		// Call target agent
		// Use consistent chatID across all turns to preserve session state (ctx.memory.chat)
		ctx := NewTestContextFromOptions(
			chatID,
			agentID,
			r.opts,
			tc,
		)

		opts := buildContextOptions(tc, r.opts)
		response, err := ast.Stream(ctx, messages, opts)
		ctx.Release()

		if err != nil {
			turnResult.Error = err.Error()
			turnResult.DurationMs = time.Since(turnStart).Milliseconds()
			result.Turns = append(result.Turns, turnResult)
			result.Status = StatusError
			result.Error = fmt.Sprintf("agent error at turn %d: %s", turn, err.Error())
			result.DurationMs = time.Since(startTime).Milliseconds()
			result.TotalTurns = turn
			return result
		}

		// Extract output (summary for display and conversation history)
		output := extractOutput(response)
		turnResult.Output = output
		turnResult.DurationMs = time.Since(turnStart).Milliseconds()

		// Store full response for reporting
		turnResult.Response = buildTurnResponse(response)

		if r.opts.Verbose {
			r.output.Verbose("Turn %d: Agent: %s", turn, truncateOutput(output, 50))
		}

		// Add assistant response to messages, including tool calls if any
		messages = appendAssistantMessages(messages, response)

		// Check checkpoints against this response (including tool results)
		reachedIDs := r.checkCheckpoints(tc.Checkpoints, response, result)
		turnResult.CheckpointsReached = reachedIDs

		if r.opts.Verbose && len(reachedIDs) > 0 {
			for _, id := range reachedIDs {
				r.output.Verbose("  ✓ checkpoint: %s", id)
			}
		}

		result.Turns = append(result.Turns, turnResult)

		// Check if all required checkpoints reached
		if r.allRequiredCheckpointsReached(result) {
			result.Status = StatusPassed
			result.DurationMs = time.Since(startTime).Milliseconds()
			result.TotalTurns = turn
			return result
		}
	}

	// Max turns exceeded
	result.Status = StatusFailed
	result.Error = fmt.Sprintf("max turns (%d) exceeded without reaching all checkpoints", maxTurns)
	result.DurationMs = time.Since(startTime).Milliseconds()
	result.TotalTurns = maxTurns
	return result
}

// buildSimulatorInput builds the input for the simulator agent
func (r *DynamicRunner) buildSimulatorInput(
	tc *Case,
	messages []context.Message,
	result *DynamicResult,
	turn, maxTurns int,
	metadata map[string]interface{},
) *SimulatorInput {
	input := &SimulatorInput{
		Conversation: messages,
		TurnNumber:   turn,
		MaxTurns:     maxTurns,
	}

	// Extract persona and goal from metadata
	if persona, ok := metadata["persona"].(string); ok {
		input.Persona = persona
	}
	if goal, ok := metadata["goal"].(string); ok {
		input.Goal = goal
	}

	// Build checkpoint lists
	input.CheckpointsReached = make([]string, 0)
	input.CheckpointsPending = make([]string, 0)
	for id, cp := range result.Checkpoints {
		if cp.Reached {
			input.CheckpointsReached = append(input.CheckpointsReached, id)
		} else {
			input.CheckpointsPending = append(input.CheckpointsPending, id)
		}
	}

	// Store extra metadata
	input.Extra = make(map[string]interface{})
	for k, v := range metadata {
		if k != "persona" && k != "goal" {
			input.Extra[k] = v
		}
	}

	return input
}

// callSimulator calls the simulator agent and parses the response
func (r *DynamicRunner) callSimulator(simAST *assistant.Assistant, tc *Case, input *SimulatorInput) (*SimulatorOutput, error) {
	// Create context
	env := NewEnvironment("", "")
	ctx := NewTestContext("simulator", tc.Simulator.Use, env)
	defer ctx.Release()

	// Build options - skip history and trace
	opts := &context.Options{
		Skip: &context.Skip{
			History: true,
			Trace:   true,
			Output:  true,
		},
		Metadata: map[string]interface{}{
			"test_mode": "simulator",
		},
	}

	// Override connector if specified
	if tc.Simulator.Options != nil && tc.Simulator.Options.Connector != "" {
		opts.Connector = tc.Simulator.Options.Connector
	}

	// Build message
	inputJSON, err := jsoniter.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal simulator input: %w", err)
	}

	messages := []context.Message{{
		Role:    context.RoleUser,
		Content: string(inputJSON),
	}}

	// Call simulator
	response, err := simAST.Stream(ctx, messages, opts)
	if err != nil {
		return nil, fmt.Errorf("simulator agent error: %w", err)
	}

	// Parse response
	return r.parseSimulatorResponse(response)
}

// parseSimulatorResponse parses the simulator agent's response
func (r *DynamicRunner) parseSimulatorResponse(response *context.Response) (*SimulatorOutput, error) {
	if response == nil || response.Completion == nil {
		return nil, fmt.Errorf("empty response from simulator")
	}

	// Extract content
	content := response.Completion.Content
	if content == nil {
		return nil, fmt.Errorf("no content in simulator response")
	}

	// Convert to string
	var text string
	switch v := content.(type) {
	case string:
		text = v
	default:
		data, err := jsoniter.Marshal(content)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal content: %w", err)
		}
		text = string(data)
	}

	// Use goutext.ExtractJSON for fault-tolerant parsing
	parsed := goutext.ExtractJSON(text)
	if parsed == nil {
		// Try to use the text as the message directly
		return &SimulatorOutput{
			Message:      text,
			GoalAchieved: false,
		}, nil
	}

	// Parse as SimulatorOutput
	output := &SimulatorOutput{}
	if m, ok := parsed.(map[string]interface{}); ok {
		if msg, ok := m["message"].(string); ok {
			output.Message = msg
		}
		if achieved, ok := m["goal_achieved"].(bool); ok {
			output.GoalAchieved = achieved
		}
		if reasoning, ok := m["reasoning"].(string); ok {
			output.Reasoning = reasoning
		}
	}

	if output.Message == "" {
		return nil, fmt.Errorf("simulator returned empty message")
	}

	return output, nil
}

// checkCheckpoints validates checkpoints against current response
// It checks both the completion content and tool results for comprehensive validation
func (r *DynamicRunner) checkCheckpoints(checkpoints []*Checkpoint, response *context.Response, result *DynamicResult) []string {
	reachedIDs := make([]string, 0)

	// Build combined output for checkpoint validation
	// This includes both content and tool result messages
	combinedOutput := buildCombinedOutput(response)

	// Set response on asserter for tool-related assertions
	r.asserter.WithResponse(response)

	for _, cp := range checkpoints {
		cpResult := result.Checkpoints[cp.ID]
		if cpResult.Reached {
			continue // Already reached
		}

		// Check "after" constraint
		if len(cp.After) > 0 {
			allAfterReached := true
			for _, afterID := range cp.After {
				if afterResult, ok := result.Checkpoints[afterID]; ok {
					if !afterResult.Reached {
						allAfterReached = false
						break
					}
				}
			}
			if !allAfterReached {
				continue // Dependencies not met
			}
		}

		// Validate using asserter against combined output with full details
		tempCase := &Case{Assert: cp.Assert}
		assertResult := r.asserter.ValidateWithDetails(tempCase, combinedOutput)

		if assertResult.Passed {
			cpResult.Reached = true
			cpResult.Passed = true
			cpResult.ReachedAtTurn = len(result.Turns) + 1
			cpResult.Message = assertResult.Message
			reachedIDs = append(reachedIDs, cp.ID)
		} else {
			// Store failure message for debugging (but don't mark as failed yet - it might pass in a later turn)
			if cpResult.Message == "" {
				cpResult.Message = assertResult.Message
			}
		}

		// Store agent validation details if this is an agent assertion
		if isAgentAssertion(cp.Assert) {
			// Extract criteria from assertion value
			var criteria string
			if assertMap, ok := cp.Assert.(map[string]interface{}); ok {
				if c, ok := assertMap["value"].(string); ok {
					criteria = c
				}
			}

			cpResult.AgentValidation = &AgentValidationResult{
				Passed:   assertResult.Passed,
				Criteria: criteria,
				Input:    combinedOutput, // Content sent to validator for checking
			}

			// Extract reason and store full response from validator
			if assertResult.Expected != nil {
				if validatorResponse, ok := assertResult.Expected.(map[string]interface{}); ok {
					if reason, ok := validatorResponse["reason"].(string); ok {
						cpResult.AgentValidation.Reason = reason
					}
					// Store the full validator response
					cpResult.AgentValidation.Response = validatorResponse
				}
			}
		}
	}

	return reachedIDs
}

// isAgentAssertion checks if the assertion is an agent-based assertion
func isAgentAssertion(assert interface{}) bool {
	if assertMap, ok := assert.(map[string]interface{}); ok {
		if assertType, ok := assertMap["type"].(string); ok {
			return assertType == "agent"
		}
	}
	return false
}

// truncateForReport truncates content for report output
func truncateForReport(content interface{}, maxLen int) interface{} {
	if content == nil {
		return nil
	}

	str, ok := content.(string)
	if !ok {
		return content
	}

	if len(str) <= maxLen {
		return str
	}

	return str[:maxLen] + "... (truncated)"
}

// buildCombinedOutput builds a combined output string from response
// that includes both completion content and tool result messages
func buildCombinedOutput(response *context.Response) string {
	if response == nil {
		return ""
	}

	var parts []string

	// Add completion content
	if response.Completion != nil && response.Completion.Content != nil {
		if content := extractContentString(response.Completion.Content); content != "" {
			parts = append(parts, content)
		}
	}

	// Add tool result messages
	if len(response.Tools) > 0 {
		for _, tool := range response.Tools {
			if tool.Result != nil {
				// Try to extract message from result
				if resultMap, ok := tool.Result.(map[string]interface{}); ok {
					if msg, exists := resultMap["message"]; exists && msg != nil {
						if msgStr, ok := msg.(string); ok && msgStr != "" {
							parts = append(parts, msgStr)
						}
					}
				}
			}
		}
	}

	// Join all parts with newline for comprehensive matching
	return joinNonEmpty(parts, "\n")
}

// extractContentString extracts string content from various types
func extractContentString(content interface{}) string {
	if content == nil {
		return ""
	}

	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		// Handle array content (e.g., multimodal content)
		var texts []string
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				if text, ok := m["text"].(string); ok {
					texts = append(texts, text)
				}
			}
		}
		return joinNonEmpty(texts, "\n")
	default:
		// Try to marshal to string
		if data, err := jsoniter.MarshalToString(content); err == nil {
			return data
		}
		return ""
	}
}

// joinNonEmpty joins non-empty strings with separator
func joinNonEmpty(parts []string, sep string) string {
	var nonEmpty []string
	for _, p := range parts {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	if len(nonEmpty) == 0 {
		return ""
	}
	result := nonEmpty[0]
	for i := 1; i < len(nonEmpty); i++ {
		result += sep + nonEmpty[i]
	}
	return result
}

// allRequiredCheckpointsReached checks if all required checkpoints are reached
func (r *DynamicRunner) allRequiredCheckpointsReached(result *DynamicResult) bool {
	for _, cp := range result.Checkpoints {
		if cp.Required && !cp.Reached {
			return false
		}
	}
	return true
}

// buildTurnResponse builds a TurnResponse from the agent response
func buildTurnResponse(response *context.Response) *TurnResponse {
	if response == nil {
		return nil
	}

	tr := &TurnResponse{}

	// Extract completion content
	if response.Completion != nil {
		tr.Content = response.Completion.Content

		// Extract tool calls from completion
		if len(response.Completion.ToolCalls) > 0 {
			for _, tc := range response.Completion.ToolCalls {
				tr.ToolCalls = append(tr.ToolCalls, ToolCallInfo{
					Tool:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				})
			}
		}
	}

	// Add tool results
	if len(response.Tools) > 0 {
		// If we already have tool calls from completion, match results
		if len(tr.ToolCalls) > 0 {
			for i, toolResult := range response.Tools {
				if i < len(tr.ToolCalls) {
					tr.ToolCalls[i].Result = toolResult.Result
				}
			}
		} else {
			// Create tool call entries from results
			for _, toolResult := range response.Tools {
				tr.ToolCalls = append(tr.ToolCalls, ToolCallInfo{
					Tool:      toolResult.Tool,
					Arguments: toolResult.Arguments,
					Result:    toolResult.Result,
				})
			}
		}
	}

	// Extract Next hook data
	if response.Next != nil && !isEmptyValue(response.Next) {
		tr.Next = response.Next
	}

	return tr
}

// appendAssistantMessages appends assistant messages to the conversation history
// including tool calls and tool results if present
func appendAssistantMessages(messages []context.Message, response *context.Response) []context.Message {
	if response == nil {
		return messages
	}

	// Check if there are tool calls in the completion
	hasToolCalls := response.Completion != nil && len(response.Completion.ToolCalls) > 0

	if hasToolCalls {
		// Add assistant message with tool calls
		assistantMsg := context.Message{
			Role:      context.RoleAssistant,
			ToolCalls: response.Completion.ToolCalls,
		}
		// Include content if present
		if response.Completion.Content != nil && !isEmptyValue(response.Completion.Content) {
			assistantMsg.Content = response.Completion.Content
		}
		messages = append(messages, assistantMsg)

		// Add tool result messages for each tool call
		for i, tc := range response.Completion.ToolCalls {
			toolCallID := tc.ID
			var resultContent string

			// Get result from response.Tools if available
			if i < len(response.Tools) {
				resultJSON, err := jsoniter.MarshalToString(response.Tools[i].Result)
				if err == nil {
					resultContent = resultJSON
				} else {
					resultContent = fmt.Sprintf("%v", response.Tools[i].Result)
				}
			} else {
				resultContent = "{}"
			}

			messages = append(messages, context.Message{
				Role:       context.RoleTool,
				ToolCallID: &toolCallID,
				Content:    resultContent,
			})
		}
	} else {
		// No tool calls, just add content if present
		content := extractOutput(response)
		if content != nil && !isEmptyValue(content) {
			messages = append(messages, context.Message{
				Role:    context.RoleAssistant,
				Content: content,
			})
		}
	}

	return messages
}
