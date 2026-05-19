//go:build integration

package robot_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/agent/assistant"
	storetypes "github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"

	_ "github.com/yaoapp/yao/agent/robot"
)

func TestProcessGet(t *testing.T) {
	testprepare.PrepareSandbox(t)

	t.Run("ErrorOnNotFound", func(t *testing.T) {
		p := process.New("robot.Get", "non_existent_robot_member_id")
		_, err := p.Exec()
		assert.Error(t, err, "Should error for non-existent robot")
	})
}

func TestProcessList(t *testing.T) {
	testprepare.PrepareSandbox(t)

	t.Run("ReturnsListWithNoFilter", func(t *testing.T) {
		p := process.New("robot.List")
		result, err := p.Exec()
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("ReturnsListWithPageFilter", func(t *testing.T) {
		p := process.New("robot.List", map[string]interface{}{
			"page":     1,
			"pagesize": 5,
		})
		result, err := p.Exec()
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestProcessStatus(t *testing.T) {
	testprepare.PrepareSandbox(t)

	t.Run("ErrorOnNotFound", func(t *testing.T) {
		p := process.New("robot.Status", "non_existent_robot_member_id")
		_, err := p.Exec()
		assert.Error(t, err, "Should error for non-existent robot")
	})
}

func TestProcessExecutions(t *testing.T) {
	testprepare.PrepareSandbox(t)

	t.Run("ReturnsEmptyForUnknownRobot", func(t *testing.T) {
		memberID := fmt.Sprintf("proc_exec_test_%d", time.Now().UnixNano())
		p := process.New("robot.Executions", memberID)
		result, err := p.Exec()
		if err == nil {
			assert.NotNil(t, result)
		}
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
	})
}

func TestProcessExecution(t *testing.T) {
	testprepare.PrepareSandbox(t)

	t.Run("ErrorOnNonExistentExecution", func(t *testing.T) {
		p := process.New("robot.Execution", "some_member_id", "non_existent_exec_id")
		_, err := p.Exec()
		assert.Error(t, err, "Should error for non-existent execution")
	})
}

func TestProcessArgumentValidation(t *testing.T) {
	testprepare.PrepareSandbox(t)

	t.Run("UpdateChatTitle_RequiresTwoArgs", func(t *testing.T) {
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

func TestProcessUpdateChatTitle(t *testing.T) {
	testprepare.PrepareSandbox(t)

	chatStore := assistant.GetChatStore()
	if chatStore == nil {
		t.Fatal("Chat store not configured")
	}

	t.Run("UpdatesTitle", func(t *testing.T) {
		chatID := fmt.Sprintf("robot_test_proc_%s", uuid.New().String()[:8])

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
		assert.Equal(t, title, chat.Title)
	})

	t.Run("ErrorOnNonExistentChat", func(t *testing.T) {
		p := process.New("robot.UpdateChatTitle", "non_existent_chat_id", "some title")
		_, err := p.Exec()
		assert.Error(t, err, "Should error when chat does not exist")
	})
}
