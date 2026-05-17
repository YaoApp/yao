//go:build e2e

package api_test

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/agent/robot/api"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
)

// TestE2EInteractNewAssignment tests the full Interact flow for a new task assignment.
// With the conversational Host Agent, the first turn may return natural language
// (waiting_for_more) or an action decision depending on request clarity.
func TestE2EInteractNewAssignment(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test - requires real LLM calls")
	}

	testutils.PrepareAgent(t)
	testutils.RequireE2EKeys(t)
	defer testutils.Clean(t)

	cleanupInteractRobots(t)
	cleanupInteractExecutions(t)
	defer cleanupInteractRobots(t)
	defer cleanupInteractExecutions(t)

	t.Run("assign_via_interact_creates_execution_and_gets_host_reply", func(t *testing.T) {
		memberID := "robot_e2e_interact_assign"
		setupInteractRobot(t, memberID, "team_e2e_interact")

		err := api.Start()
		require.NoError(t, err)
		defer api.Stop()

		ctx := types.NewContext(context.Background(), testAuth())

		robot, err := api.GetRobot(ctx, memberID)
		require.NoError(t, err)
		require.NotNil(t, robot)

		result, err := api.Interact(ctx, memberID, &api.InteractRequest{
			Source:  types.InteractSourceUI,
			Message: "Please write a short greeting email for our team meeting tomorrow morning.",
		})
		require.NoError(t, err)
		require.NotNil(t, result)

		t.Logf("Interact result: status=%s, message=%s, reply=%s, exec_id=%s, wait_for_more=%v",
			result.Status, result.Message, result.Reply, result.ExecutionID, result.WaitForMore)

		assert.NotEmpty(t, result.ExecutionID, "should create an execution")
		assert.NotEmpty(t, result.ChatID, "should have a chat session")
		assert.NotEmpty(t, result.Reply, "Host Agent should provide a reply")

		validStatuses := []string{"confirmed", "waiting_for_more", "adjusted", "acknowledged"}
		assert.Contains(t, validStatuses, result.Status,
			"status should be one of the valid Host Agent action outcomes")

		if result.Status == "confirmed" {
			time.Sleep(2 * time.Second)
			executions, err := api.ListExecutions(ctx, memberID, &api.ExecutionQuery{Page: 1, PageSize: 5})
			require.NoError(t, err)
			assert.Greater(t, len(executions.Data), 0, "confirmed execution should exist in store")
		}
	})
}

// TestE2EInteractStream tests the streaming version end-to-end.
func TestE2EInteractStream(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test - requires real LLM calls")
	}

	testutils.PrepareAgent(t)
	testutils.RequireE2EKeys(t)
	defer testutils.Clean(t)

	cleanupInteractRobots(t)
	cleanupInteractExecutions(t)
	defer cleanupInteractRobots(t)
	defer cleanupInteractExecutions(t)

	t.Run("stream_assign_returns_chunks_and_valid_result", func(t *testing.T) {
		memberID := "robot_e2e_interact_stream"
		setupInteractRobot(t, memberID, "team_e2e_interact")

		err := api.Start()
		require.NoError(t, err)
		defer api.Stop()

		ctx := types.NewContext(context.Background(), testAuth())

		var mu sync.Mutex
		var chunks []*standard.StreamChunk

		streamFn := func(chunk *standard.StreamChunk) int {
			mu.Lock()
			defer mu.Unlock()
			chunks = append(chunks, chunk)
			return 0
		}

		result, err := api.InteractStream(ctx, memberID, &api.InteractRequest{
			Source:  types.InteractSourceUI,
			Message: "Help me draft a brief status update email about completing the Q4 report.",
		}, streamFn)
		require.NoError(t, err)
		require.NotNil(t, result)

		mu.Lock()
		chunkCount := len(chunks)
		var textChunks []string
		for _, c := range chunks {
			if c.Type == "text" && c.Delta {
				textChunks = append(textChunks, c.Content)
			}
		}
		mu.Unlock()

		combined := strings.Join(textChunks, "")

		t.Logf("Stream received %d total chunks, %d text chunks, combined length: %d",
			chunkCount, len(textChunks), len(combined))
		t.Logf("Result: status=%s, exec_id=%s, reply_len=%d, wait_for_more=%v",
			result.Status, result.ExecutionID, len(result.Reply), result.WaitForMore)

		assert.Greater(t, len(textChunks), 0, "should receive streaming text chunks from Host Agent")
		assert.NotEmpty(t, combined, "combined text should not be empty")
		assert.NotEmpty(t, result.ExecutionID, "should create an execution")
		assert.NotEmpty(t, result.Reply, "final result should contain reply")

		validStatuses := []string{"confirmed", "waiting_for_more", "adjusted"}
		assert.Contains(t, validStatuses, result.Status)
	})
}

// TestE2EInteractMultiTurn tests a multi-turn conversation:
// Turn 1: Send vague message -> Host Agent replies conversationally (waiting_for_more)
// Turn 2: Send clear confirmation -> Host Agent returns action JSON (confirmed or other action)
func TestE2EInteractMultiTurn(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test - requires real LLM calls")
	}

	testutils.PrepareAgent(t)
	testutils.RequireE2EKeys(t)
	defer testutils.Clean(t)

	cleanupInteractRobots(t)
	cleanupInteractExecutions(t)
	defer cleanupInteractRobots(t)
	defer cleanupInteractExecutions(t)

	t.Run("multi_turn_assign_conversation", func(t *testing.T) {
		memberID := "robot_e2e_interact_multiturn"
		setupInteractRobot(t, memberID, "team_e2e_interact")

		err := api.Start()
		require.NoError(t, err)
		defer api.Stop()

		ctx := types.NewContext(context.Background(), testAuth())

		// Turn 1: Send vague message — expect conversational reply
		result1, err := api.Interact(ctx, memberID, &api.InteractRequest{
			Source:  types.InteractSourceUI,
			Message: "Do something with emails.",
		})
		require.NoError(t, err)
		require.NotNil(t, result1)

		t.Logf("Turn 1: status=%s, reply=%s, exec_id=%s, wait_for_more=%v",
			result1.Status, result1.Reply, result1.ExecutionID, result1.WaitForMore)

		assert.NotEmpty(t, result1.ExecutionID)
		assert.NotEmpty(t, result1.Reply)

		// Turn 2: Clarify/confirm with the same execution_id
		result2, err := api.Interact(ctx, memberID, &api.InteractRequest{
			ExecutionID: result1.ExecutionID,
			Source:      types.InteractSourceUI,
			Message:     "Yes, please write a brief thank-you email to the design team for their Q4 work. Go ahead and confirm.",
		})
		require.NoError(t, err)
		require.NotNil(t, result2)

		t.Logf("Turn 2: status=%s, reply=%s, exec_id=%s, wait_for_more=%v",
			result2.Status, result2.Reply, result2.ExecutionID, result2.WaitForMore)

		assert.NotEmpty(t, result2.Reply)
		assert.Equal(t, result1.ExecutionID, result2.ExecutionID, "should be same execution")

		validStatuses := []string{"confirmed", "waiting_for_more", "adjusted", "acknowledged"}
		assert.Contains(t, validStatuses, result2.Status,
			"second turn should produce a valid outcome")
	})
}

// TestE2EInteractStreamMultiTurn tests multi-turn with streaming.
func TestE2EInteractStreamMultiTurn(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test - requires real LLM calls")
	}

	testutils.PrepareAgent(t)
	testutils.RequireE2EKeys(t)
	defer testutils.Clean(t)

	cleanupInteractRobots(t)
	cleanupInteractExecutions(t)
	defer cleanupInteractRobots(t)
	defer cleanupInteractExecutions(t)

	t.Run("stream_multi_turn", func(t *testing.T) {
		memberID := "robot_e2e_interact_stream_mt"
		setupInteractRobot(t, memberID, "team_e2e_interact")

		err := api.Start()
		require.NoError(t, err)
		defer api.Stop()

		ctx := types.NewContext(context.Background(), testAuth())

		// Turn 1
		var mu1 sync.Mutex
		var chunks1 []*standard.StreamChunk
		result1, err := api.InteractStream(ctx, memberID, &api.InteractRequest{
			Source:  types.InteractSourceUI,
			Message: "I need help with something.",
		}, func(chunk *standard.StreamChunk) int {
			mu1.Lock()
			chunks1 = append(chunks1, chunk)
			mu1.Unlock()
			return 0
		})
		require.NoError(t, err)
		require.NotNil(t, result1)

		mu1.Lock()
		t.Logf("Turn 1 stream: %d chunks, status=%s, reply=%s, wait_for_more=%v",
			len(chunks1), result1.Status, result1.Reply, result1.WaitForMore)
		mu1.Unlock()

		assert.NotEmpty(t, result1.ExecutionID)
		assert.NotEmpty(t, result1.Reply)

		// Turn 2: Clarify with same execution_id
		var mu2 sync.Mutex
		var chunks2 []*standard.StreamChunk
		result2, err := api.InteractStream(ctx, memberID, &api.InteractRequest{
			ExecutionID: result1.ExecutionID,
			Source:      types.InteractSourceUI,
			Message:     "Please compose a short farewell message for a colleague leaving the team. Yes, go ahead.",
		}, func(chunk *standard.StreamChunk) int {
			mu2.Lock()
			chunks2 = append(chunks2, chunk)
			mu2.Unlock()
			return 0
		})
		require.NoError(t, err)
		require.NotNil(t, result2)

		mu2.Lock()
		t.Logf("Turn 2 stream: %d chunks, status=%s, reply=%s, wait_for_more=%v",
			len(chunks2), result2.Status, result2.Reply, result2.WaitForMore)
		mu2.Unlock()

		assert.NotEmpty(t, result2.Reply)
		assert.Equal(t, result1.ExecutionID, result2.ExecutionID)
	})
}

// ==================== Helper Functions ====================

func setupInteractRobot(t *testing.T, memberID, teamID string) {
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{
			"role":   "Email Assistant",
			"duties": []string{"Write and manage emails"},
			"rules":  []string{"Always confirm before sending", "Keep emails professional"},
		},
		"quota": map[string]interface{}{
			"max":      5,
			"queue":    20,
			"priority": 5,
		},
		"triggers": map[string]interface{}{
			"intervene": map[string]interface{}{"enabled": true},
		},
		"resources": map[string]interface{}{
			"phases": map[string]interface{}{
				"inspiration": "robot.inspiration",
				"goals":       "robot.goals",
				"tasks":       "robot.tasks",
				"run":         "robot.validation",
				"validation":  "robot.validation",
				"delivery":    "robot.delivery",
				"learning":    "robot.learning",
				"host":        "robot.host",
			},
			"agents": []string{},
		},
	}
	configJSON, _ := json.Marshal(robotConfig)

	systemPrompt := `You are an email assistant for E2E testing of the Interact API.
When asked to write an email, confirm the task and generate a brief email draft.`

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "E2E Interact Test Robot " + memberID,
			"system_prompt":   systemPrompt,
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": false,
			"robot_status":    "idle",
			"robot_config":    string(configJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert interact robot %s: %v", memberID, err)
	}
}

func cleanupInteractRobots(t *testing.T) {
	m := model.Select("__yao.member")
	if m == nil {
		return
	}
	qb := capsule.Query()
	_, err := qb.Table(m.MetaData.Table.Name).Where("member_id", "like", "robot_e2e_interact%").Delete()
	if err != nil {
		t.Logf("Warning: cleanup interact robots: %v", err)
	}
}

func cleanupInteractExecutions(t *testing.T) {
	m := model.Select("__yao.agent.execution")
	if m == nil {
		return
	}
	qb := capsule.Query()
	_, err := qb.Table(m.MetaData.Table.Name).Where("member_id", "like", "robot_e2e_interact%").Delete()
	if err != nil {
		t.Logf("Warning: cleanup interact executions: %v", err)
	}
}
