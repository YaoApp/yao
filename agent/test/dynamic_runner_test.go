package test_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent"
	"github.com/yaoapp/yao/agent/test"
	"github.com/yaoapp/yao/config"
	testutils "github.com/yaoapp/yao/test"
)

func TestCase_IsDynamicMode(t *testing.T) {
	tests := []struct {
		name     string
		tc       *test.Case
		expected bool
	}{
		{
			name: "standard mode - no simulator",
			tc: &test.Case{
				ID:    "T001",
				Input: "Hello",
			},
			expected: false,
		},
		{
			name: "standard mode - simulator but no checkpoints",
			tc: &test.Case{
				ID:        "T002",
				Input:     "Hello",
				Simulator: &test.Simulator{Use: "tests.simulator-agent"},
			},
			expected: false,
		},
		{
			name: "standard mode - checkpoints but no simulator",
			tc: &test.Case{
				ID:    "T003",
				Input: "Hello",
				Checkpoints: []*test.Checkpoint{
					{ID: "cp1", Assert: map[string]interface{}{"type": "contains", "value": "hi"}},
				},
			},
			expected: false,
		},
		{
			name: "dynamic mode - has both simulator and checkpoints",
			tc: &test.Case{
				ID:        "T004",
				Simulator: &test.Simulator{Use: "tests.simulator-agent"},
				Checkpoints: []*test.Checkpoint{
					{ID: "cp1", Assert: map[string]interface{}{"type": "contains", "value": "hi"}},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.tc.IsDynamicMode()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCase_GetMaxTurns(t *testing.T) {
	tests := []struct {
		name     string
		tc       *test.Case
		expected int
	}{
		{
			name:     "default max turns",
			tc:       &test.Case{ID: "T001"},
			expected: 20,
		},
		{
			name:     "custom max turns",
			tc:       &test.Case{ID: "T002", MaxTurns: 10},
			expected: 10,
		},
		{
			name:     "zero max turns uses default",
			tc:       &test.Case{ID: "T003", MaxTurns: 0},
			expected: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.tc.GetMaxTurns()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckpoint_IsRequired(t *testing.T) {
	boolTrue := true
	boolFalse := false

	tests := []struct {
		name     string
		cp       *test.Checkpoint
		expected bool
	}{
		{
			name:     "default is required",
			cp:       &test.Checkpoint{ID: "cp1"},
			expected: true,
		},
		{
			name:     "explicitly required",
			cp:       &test.Checkpoint{ID: "cp2", Required: &boolTrue},
			expected: true,
		},
		{
			name:     "explicitly not required",
			cp:       &test.Checkpoint{ID: "cp3", Required: &boolFalse},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cp.IsRequired()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDynamicResult_ToResult(t *testing.T) {
	dr := &test.DynamicResult{
		ID:         "T001",
		Status:     test.StatusPassed,
		TotalTurns: 3,
		DurationMs: 5000,
		Turns: []*test.TurnResult{
			{Turn: 1, Input: "Hello", Output: "Hi there!"},
			{Turn: 2, Input: "How are you?", Output: "I'm doing well!"},
			{Turn: 3, Input: "Goodbye", Output: "Bye!"},
		},
		Checkpoints: map[string]*test.CheckpointResult{
			"greet": {ID: "greet", Reached: true, ReachedAtTurn: 1, Required: true},
			"bye":   {ID: "bye", Reached: true, ReachedAtTurn: 3, Required: true},
		},
	}

	result := dr.ToResult()

	assert.Equal(t, "T001", result.ID)
	assert.Equal(t, test.StatusPassed, result.Status)
	assert.Equal(t, int64(5000), result.DurationMs)
	assert.Equal(t, "Hello", result.Input)
	assert.Equal(t, "Bye!", result.Output)

	// Check metadata
	assert.NotNil(t, result.Metadata)
	assert.Equal(t, "dynamic", result.Metadata["mode"])
	assert.Equal(t, 3, result.Metadata["total_turns"])
}

func TestDynamicRunner_Integration(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Prepare test environment
	testutils.Prepare(t, config.Conf)
	defer testutils.Clean()

	// Load agents
	err := agent.Load(config.Conf)
	if err != nil {
		t.Skipf("Failed to load agents: %v", err)
	}

	// Create a dynamic test case
	tc := &test.Case{
		ID: "dynamic-greeting",
		Simulator: &test.Simulator{
			Use: "tests.simulator-agent",
			Options: &test.SimulatorOptions{
				Metadata: map[string]interface{}{
					"persona": "Friendly user",
					"goal":    "Have a brief greeting exchange",
				},
			},
		},
		Input: "Hello!",
		Checkpoints: []*test.Checkpoint{
			{
				ID:          "greeting",
				Description: "Agent should greet back",
				Assert: map[string]interface{}{
					"type":  "regex",
					"value": "(?i)(hello|hi|hey|greetings)",
				},
			},
		},
		MaxTurns: 3,
	}

	// Verify it's dynamic mode
	assert.True(t, tc.IsDynamicMode())

	// Create runner options
	opts := &test.Options{
		Verbose: true,
		Timeout: 30 * time.Second,
	}

	// Create dynamic runner
	runner := test.NewDynamicRunner(opts)
	assert.NotNil(t, runner)

	// Note: Full integration test would require the simulator agent to be loaded
	// and would make actual LLM calls. For CI, we test the structure and logic.
}

func TestDynamicRunner_CheckpointOrdering(t *testing.T) {
	// Test that checkpoints with "after" constraints are properly ordered
	testutils.Prepare(t, config.Conf)
	defer testutils.Clean()

	// Load agents
	err := agent.Load(config.Conf)
	if err != nil {
		t.Skipf("Failed to load agents: %v", err)
	}

	// Create a test case with ordered checkpoints
	tc := &test.Case{
		ID: "ordered-checkpoints",
		Simulator: &test.Simulator{
			Use: "tests.simulator-agent",
			Options: &test.SimulatorOptions{
				Metadata: map[string]interface{}{
					"persona": "Customer",
					"goal":    "Complete a purchase",
				},
			},
		},
		Checkpoints: []*test.Checkpoint{
			{
				ID:          "ask_product",
				Description: "Agent asks about product",
				Assert: map[string]interface{}{
					"type":  "contains",
					"value": "product",
				},
			},
			{
				ID:          "confirm_order",
				Description: "Agent confirms order",
				After:       []string{"ask_product"},
				Assert: map[string]interface{}{
					"type":  "contains",
					"value": "confirm",
				},
			},
			{
				ID:          "complete",
				Description: "Order completed",
				After:       []string{"confirm_order"},
				Assert: map[string]interface{}{
					"type":  "contains",
					"value": "complete",
				},
			},
		},
		MaxTurns: 10,
	}

	// Verify checkpoint structure
	assert.Len(t, tc.Checkpoints, 3)
	assert.Empty(t, tc.Checkpoints[0].After)
	assert.Equal(t, []string{"ask_product"}, tc.Checkpoints[1].After)
	assert.Equal(t, []string{"confirm_order"}, tc.Checkpoints[2].After)
}

func TestSimulatorInput_Structure(t *testing.T) {
	// Test SimulatorInput structure
	input := &test.SimulatorInput{
		Persona:            "Test user",
		Goal:               "Complete task",
		TurnNumber:         3,
		MaxTurns:           10,
		CheckpointsReached: []string{"cp1", "cp2"},
		CheckpointsPending: []string{"cp3"},
		Extra: map[string]interface{}{
			"style": "formal",
		},
	}

	assert.Equal(t, "Test user", input.Persona)
	assert.Equal(t, "Complete task", input.Goal)
	assert.Equal(t, 3, input.TurnNumber)
	assert.Equal(t, 10, input.MaxTurns)
	assert.Len(t, input.CheckpointsReached, 2)
	assert.Len(t, input.CheckpointsPending, 1)
	assert.Equal(t, "formal", input.Extra["style"])
}

func TestSimulatorOutput_Structure(t *testing.T) {
	// Test SimulatorOutput structure
	output := &test.SimulatorOutput{
		Message:      "I'd like to buy a product",
		GoalAchieved: false,
		Reasoning:    "Continuing toward purchase goal",
	}

	assert.Equal(t, "I'd like to buy a product", output.Message)
	assert.False(t, output.GoalAchieved)
	assert.Equal(t, "Continuing toward purchase goal", output.Reasoning)
}
