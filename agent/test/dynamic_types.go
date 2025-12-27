package test

import "github.com/yaoapp/yao/agent/context"

// DynamicResult represents the result of a dynamic (simulator-driven) test
type DynamicResult struct {
	// ID is the test case identifier
	ID string `json:"id"`

	// Status is the overall test status
	Status Status `json:"status"`

	// Turns contains results for each conversation turn
	Turns []*TurnResult `json:"turns"`

	// Checkpoints maps checkpoint ID to its result
	Checkpoints map[string]*CheckpointResult `json:"checkpoints"`

	// TotalTurns is the number of turns executed
	TotalTurns int `json:"total_turns"`

	// DurationMs is the total execution time in milliseconds
	DurationMs int64 `json:"duration_ms"`

	// Error contains error message if status is failed/error/timeout
	Error string `json:"error,omitempty"`
}

// TurnResult represents the result of a single conversation turn
type TurnResult struct {
	// Turn is the turn number (1-based)
	Turn int `json:"turn"`

	// Input is the user message (from simulator or initial input)
	Input interface{} `json:"input"`

	// Output is the agent's response (summary for display and conversation history)
	Output interface{} `json:"output,omitempty"`

	// Response is the full agent response including completion and tool results
	Response *TurnResponse `json:"response,omitempty"`

	// CheckpointsReached lists checkpoint IDs reached in this turn
	CheckpointsReached []string `json:"checkpoints_reached,omitempty"`

	// DurationMs is the turn execution time in milliseconds
	DurationMs int64 `json:"duration_ms"`

	// Error contains error message if this turn failed
	Error string `json:"error,omitempty"`
}

// TurnResponse contains the full agent response for a turn
type TurnResponse struct {
	// Content is the text content from LLM completion
	Content interface{} `json:"content,omitempty"`

	// ToolCalls contains the tool calls made by the agent
	ToolCalls []ToolCallInfo `json:"tool_calls,omitempty"`

	// Next is the data returned from Next hook
	Next interface{} `json:"next,omitempty"`
}

// ToolCallInfo contains information about a tool call
type ToolCallInfo struct {
	// Tool is the tool name
	Tool string `json:"tool"`

	// Arguments are the tool call arguments
	Arguments interface{} `json:"arguments,omitempty"`

	// Result is the tool execution result
	Result interface{} `json:"result,omitempty"`
}

// CheckpointResult represents the result of a checkpoint validation
type CheckpointResult struct {
	// ID is the checkpoint identifier
	ID string `json:"id"`

	// Reached indicates if the checkpoint was reached
	Reached bool `json:"reached"`

	// ReachedAtTurn is the turn number when checkpoint was reached (0 if not reached)
	ReachedAtTurn int `json:"reached_at_turn,omitempty"`

	// Required indicates if this checkpoint is required
	Required bool `json:"required"`

	// Passed indicates if the checkpoint assertion passed
	Passed bool `json:"passed"`

	// Message contains assertion result message
	Message string `json:"message,omitempty"`

	// AgentValidation contains the agent validator's response (for agent assertions)
	AgentValidation *AgentValidationResult `json:"agent_validation,omitempty"`
}

// AgentValidationResult contains the result from an agent-based assertion
type AgentValidationResult struct {
	// Passed indicates if the agent validator determined the assertion passed
	Passed bool `json:"passed"`

	// Reason is the explanation from the agent validator
	Reason string `json:"reason,omitempty"`

	// Criteria is the validation criteria that was checked
	Criteria string `json:"criteria,omitempty"`

	// Input is the content that was sent to validator for checking
	Input interface{} `json:"input,omitempty"`

	// Response is the raw response from the validator agent
	Response interface{} `json:"response,omitempty"`
}

// SimulatorInput is the input sent to the simulator agent
type SimulatorInput struct {
	// Persona describes the user being simulated
	Persona string `json:"persona,omitempty"`

	// Goal is what the user is trying to achieve
	Goal string `json:"goal,omitempty"`

	// Conversation is the message history
	Conversation []context.Message `json:"conversation"`

	// TurnNumber is the current turn (1-based)
	TurnNumber int `json:"turn_number"`

	// MaxTurns is the maximum allowed turns
	MaxTurns int `json:"max_turns"`

	// CheckpointsReached lists checkpoint IDs already reached
	CheckpointsReached []string `json:"checkpoints_reached,omitempty"`

	// CheckpointsPending lists checkpoint IDs still pending
	CheckpointsPending []string `json:"checkpoints_pending,omitempty"`

	// Extra metadata from simulator options
	Extra map[string]interface{} `json:"extra,omitempty"`
}

// SimulatorOutput is the expected output from the simulator agent
type SimulatorOutput struct {
	// Message is the simulated user message
	Message string `json:"message"`

	// GoalAchieved indicates if the user's goal has been accomplished
	GoalAchieved bool `json:"goal_achieved"`

	// Reasoning explains the simulator's response strategy
	Reasoning string `json:"reasoning,omitempty"`
}

// ToResult converts DynamicResult to standard Result for reporting
func (dr *DynamicResult) ToResult() *Result {
	result := &Result{
		ID:         dr.ID,
		Status:     dr.Status,
		DurationMs: dr.DurationMs,
		Error:      dr.Error,
	}

	// Store dynamic-specific data in metadata
	result.Metadata = map[string]interface{}{
		"mode":        "dynamic",
		"total_turns": dr.TotalTurns,
		"turns":       dr.Turns,
		"checkpoints": dr.Checkpoints,
	}

	// Set input from first turn
	if len(dr.Turns) > 0 {
		result.Input = dr.Turns[0].Input
	}

	// Set output from last turn
	if len(dr.Turns) > 0 {
		result.Output = dr.Turns[len(dr.Turns)-1].Output
	}

	return result
}

// IsDynamicMode checks if a test case should run in dynamic mode
func (tc *Case) IsDynamicMode() bool {
	return tc.Simulator != nil && len(tc.Checkpoints) > 0
}

// GetMaxTurns returns the max turns for dynamic mode
func (tc *Case) GetMaxTurns() int {
	if tc.MaxTurns > 0 {
		return tc.MaxTurns
	}
	return 20 // Default max turns
}

// IsRequired returns true if the checkpoint is required
func (cp *Checkpoint) IsRequired() bool {
	if cp.Required == nil {
		return true // Default to required
	}
	return *cp.Required
}
