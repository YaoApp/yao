package robot_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/agent/assistant"
	storetypes "github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/agent/testutils"

	// Register robot process handlers via init()
	_ "github.com/yaoapp/yao/agent/robot"
)

// ============================================================================
// robot.UpdateChatTitle
// ============================================================================

func TestProcessUpdateChatTitle(t *testing.T) {
	testutils.PrepareAgent(t)
	defer testutils.Clean(t)

	chatStore := assistant.GetChatStore()
	if chatStore == nil {
		t.Skip("Chat store not configured, skipping UpdateChatTitle tests")
	}

	t.Run("UpdatesTitle", func(t *testing.T) {
		chatID := fmt.Sprintf("robot_test_proc_%s", uuid.New().String()[:8])

		// Create a chat record first
		err := chatStore.CreateChat(&storetypes.Chat{
			ChatID:      chatID,
			AssistantID: "robot.host",
			Status:      "active",
			Share:       "private",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
		require.NoError(t, err)
		defer chatStore.DeleteChat(chatID)

		title := "Create a mecha image, sci-fi style"
		p := process.New("robot.UpdateChatTitle", chatID, title)
		_, err = p.Exec()
		require.NoError(t, err)

		chat, err := chatStore.GetChat(chatID)
		require.NoError(t, err)
		assert.Equal(t, title, chat.Title, "Title should be updated to the confirmed goals")
		t.Logf("✓ robot.UpdateChatTitle: chat_id=%s, title=%q", chatID, chat.Title)
	})

	t.Run("UpdatesLongGoalsTitle", func(t *testing.T) {
		chatID := fmt.Sprintf("robot_test_long_%s", uuid.New().String()[:8])

		err := chatStore.CreateChat(&storetypes.Chat{
			ChatID:      chatID,
			AssistantID: "robot.host",
			Status:      "active",
			Share:       "private",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
		require.NoError(t, err)
		defer chatStore.DeleteChat(chatID)

		// Long goals string — title field should accommodate it
		title := "请帮我制作一张充满未来感的机甲图片，风格参考《攻壳机动队》，以赛博朋克城市为背景，色调偏冷，蓝紫配色"
		p := process.New("robot.UpdateChatTitle", chatID, title)
		_, err = p.Exec()
		require.NoError(t, err)

		chat, err := chatStore.GetChat(chatID)
		require.NoError(t, err)
		assert.Equal(t, title, chat.Title)
		t.Logf("✓ Long goals title persisted: %d chars", len(title))
	})

	t.Run("ErrorOnNonExistentChat", func(t *testing.T) {
		p := process.New("robot.UpdateChatTitle", "non_existent_chat_id", "some title")
		_, err := p.Exec()
		assert.Error(t, err, "Should error when chat does not exist")
		t.Logf("✓ Non-existent chat correctly returns error")
	})
}

// ============================================================================
// robot.Get
// ============================================================================

func TestProcessGet(t *testing.T) {
	testutils.PrepareAgent(t)
	defer testutils.Clean(t)

	t.Run("ErrorOnNotFound", func(t *testing.T) {
		p := process.New("robot.Get", "non_existent_robot_member_id")
		_, err := p.Exec()
		assert.Error(t, err, "Should error for non-existent robot")
		t.Logf("✓ robot.Get returns error for non-existent robot")
	})
}

// ============================================================================
// robot.List
// ============================================================================

func TestProcessList(t *testing.T) {
	testutils.PrepareAgent(t)
	defer testutils.Clean(t)

	t.Run("ReturnsListWithNoFilter", func(t *testing.T) {
		p := process.New("robot.List")
		result, err := p.Exec()
		require.NoError(t, err)
		// Result is a paginated list — just assert it's not nil
		assert.NotNil(t, result)
		t.Logf("✓ robot.List returned: %T", result)
	})

	t.Run("ReturnsListWithPageFilter", func(t *testing.T) {
		p := process.New("robot.List", map[string]interface{}{
			"page":     1,
			"pagesize": 5,
		})
		result, err := p.Exec()
		require.NoError(t, err)
		assert.NotNil(t, result)
		t.Logf("✓ robot.List with page filter returned: %T", result)
	})
}

// ============================================================================
// robot.Status
// ============================================================================

func TestProcessStatus(t *testing.T) {
	testutils.PrepareAgent(t)
	defer testutils.Clean(t)

	t.Run("ErrorOnNotFound", func(t *testing.T) {
		p := process.New("robot.Status", "non_existent_robot_member_id")
		_, err := p.Exec()
		assert.Error(t, err, "Should error for non-existent robot")
		t.Logf("✓ robot.Status returns error for non-existent robot")
	})
}

// ============================================================================
// robot.Executions
// ============================================================================

func TestProcessExecutions(t *testing.T) {
	testutils.PrepareAgent(t)
	defer testutils.Clean(t)

	t.Run("ReturnsEmptyForUnknownRobot", func(t *testing.T) {
		memberID := fmt.Sprintf("proc_exec_test_%d", time.Now().UnixNano())
		p := process.New("robot.Executions", memberID)
		result, err := p.Exec()
		// May error or return empty — both acceptable
		if err == nil {
			assert.NotNil(t, result)
		}
		t.Logf("✓ robot.Executions handled for unknown robot")
	})

	t.Run("AcceptsFilterMap", func(t *testing.T) {
		memberID := fmt.Sprintf("proc_exec_filter_%d", time.Now().UnixNano())
		p := process.New("robot.Executions", memberID, map[string]interface{}{
			"page":     1,
			"pagesize": 10,
			"status":   "completed",
		})
		result, err := p.Exec()
		if err == nil {
			assert.NotNil(t, result)
		}
		t.Logf("✓ robot.Executions with filter handled")
	})
}

// ============================================================================
// robot.Execution
// ============================================================================

func TestProcessExecution(t *testing.T) {
	testutils.PrepareAgent(t)
	defer testutils.Clean(t)

	t.Run("ErrorOnNonExistentExecution", func(t *testing.T) {
		p := process.New("robot.Execution", "some_member_id", "non_existent_exec_id")
		_, err := p.Exec()
		assert.Error(t, err, "Should error for non-existent execution")
		t.Logf("✓ robot.Execution returns error for non-existent execution")
	})
}

// ============================================================================
// Argument Validation
// ============================================================================

func TestProcessArgumentValidation(t *testing.T) {
	testutils.PrepareAgent(t)
	defer testutils.Clean(t)

	t.Run("UpdateChatTitle_RequiresTwoArgs", func(t *testing.T) {
		// Missing title argument
		p := process.New("robot.UpdateChatTitle", "some_chat_id")
		_, err := p.Exec()
		assert.Error(t, err, "Should require 2 arguments")
	})

	t.Run("Get_RequiresOneArg", func(t *testing.T) {
		p := process.New("robot.Get")
		_, err := p.Exec()
		assert.Error(t, err, "Should require 1 argument")
	})

	t.Run("Status_RequiresOneArg", func(t *testing.T) {
		p := process.New("robot.Status")
		_, err := p.Exec()
		assert.Error(t, err, "Should require 1 argument")
	})

	t.Run("Execution_RequiresTwoArgs", func(t *testing.T) {
		p := process.New("robot.Execution", "only_one_arg")
		_, err := p.Exec()
		assert.Error(t, err, "Should require 2 arguments")
	})
}

// ============================================================================
// Integration: UpdateChatTitle flow (simulate Host Agent Next Hook)
// ============================================================================

func TestProcessUpdateChatTitleIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.PrepareAgent(t)
	defer testutils.Clean(t)

	chatStore := assistant.GetChatStore()
	if chatStore == nil {
		t.Skip("Chat store not configured")
	}

	t.Run("SimulatesHostAgentNextHook", func(t *testing.T) {
		// Simulate the full flow:
		// 1. ChatDrawer creates a chat with robot_id in metadata
		// 2. Host Agent Next Hook calls robot.UpdateChatTitle with confirmed goals
		// 3. History dropdown shows the goals as the chat title

		memberID := "120004485525"
		chatID := fmt.Sprintf("robot_%s_%d", memberID, time.Now().UnixMilli())
		confirmedGoals := "制作一张机甲图片，风格和设计由AI自主决定"

		// Step 1: Create chat (simulating AssignTaskDrawer)
		err := chatStore.CreateChat(&storetypes.Chat{
			ChatID:      chatID,
			AssistantID: "yao.robot-host",
			Status:      "active",
			Share:       "private",
			Metadata: map[string]interface{}{
				"robot_id": memberID,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
		require.NoError(t, err)
		defer chatStore.DeleteChat(chatID)

		// Step 2: Next Hook calls robot.UpdateChatTitle
		p := process.New("robot.UpdateChatTitle", chatID, confirmedGoals)
		_, err = p.Exec()
		require.NoError(t, err)

		// Step 3: Verify title is set (history dropdown will display this)
		chat, err := chatStore.GetChat(chatID)
		require.NoError(t, err)
		assert.Equal(t, confirmedGoals, chat.Title,
			"History dropdown should show confirmed goals as title")
		require.NotNil(t, chat.Metadata)
		assert.Equal(t, memberID, chat.Metadata["robot_id"],
			"Metadata robot_id should be preserved after title update")

		t.Logf("✓ Full Host Agent flow: chat_id=%s, title=%q, robot_id=%v",
			chatID, chat.Title, chat.Metadata["robot_id"])
	})
}

// ensure context is used (avoid unused import)
var _ = context.Background
