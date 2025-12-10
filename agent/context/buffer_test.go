package context_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/context"
)

// =============================================================================
// ChatBuffer Creation Tests
// =============================================================================

func TestBufferNewChatBuffer(t *testing.T) {
	t.Run("CreateWithAllFields", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-123", "req-456", "assistant-789", "", "")

		assert.NotNil(t, buffer)
		assert.Equal(t, "chat-123", buffer.ChatID())
		assert.Equal(t, "req-456", buffer.RequestID())
		assert.Equal(t, "assistant-789", buffer.AssistantID())
		assert.Empty(t, buffer.GetMessages())
		assert.Empty(t, buffer.GetAllSteps())
		assert.Equal(t, 0, buffer.GetMessageCount())
	})

	t.Run("CreateWithEmptyFields", func(t *testing.T) {
		buffer := context.NewChatBuffer("", "", "", "", "")

		assert.NotNil(t, buffer)
		assert.Empty(t, buffer.ChatID())
		assert.Empty(t, buffer.RequestID())
		assert.Empty(t, buffer.AssistantID())
	})
}

// =============================================================================
// Message Buffer Tests
// =============================================================================

func TestBufferAddMessage(t *testing.T) {
	t.Run("AddSingleMessage", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-1", "req-1", "assistant-1", "", "")

		msg := &context.BufferedMessage{
			Role:  "assistant",
			Type:  "text",
			Props: map[string]interface{}{"content": "Hello"},
		}
		buffer.AddMessage(msg)

		messages := buffer.GetMessages()
		require.Len(t, messages, 1)
		assert.Equal(t, "assistant", messages[0].Role)
		assert.Equal(t, "text", messages[0].Type)
		assert.Equal(t, 1, messages[0].Sequence)
		assert.NotEmpty(t, messages[0].MessageID) // Auto-generated
		assert.Equal(t, "chat-1", messages[0].ChatID)
		assert.Equal(t, "req-1", messages[0].RequestID)
		assert.False(t, messages[0].CreatedAt.IsZero())
	})

	t.Run("AddMultipleMessages", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-2", "req-2", "assistant-2", "", "")

		for i := 0; i < 5; i++ {
			buffer.AddMessage(&context.BufferedMessage{
				Role:  "assistant",
				Type:  "text",
				Props: map[string]interface{}{"content": fmt.Sprintf("Message %d", i+1)},
			})
		}

		messages := buffer.GetMessages()
		require.Len(t, messages, 5)

		// Verify sequence numbers
		for i, msg := range messages {
			assert.Equal(t, i+1, msg.Sequence)
		}
	})

	t.Run("AddNilMessage", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-3", "req-3", "assistant-3", "", "")
		buffer.AddMessage(nil)

		assert.Equal(t, 0, buffer.GetMessageCount())
	})

	t.Run("AddMessageWithExistingID", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-4", "req-4", "assistant-4", "", "")

		msg := &context.BufferedMessage{
			MessageID: "custom-id-123",
			Role:      "assistant",
			Type:      "text",
		}
		buffer.AddMessage(msg)

		messages := buffer.GetMessages()
		require.Len(t, messages, 1)
		assert.Equal(t, "custom-id-123", messages[0].MessageID) // Preserved
	})

	t.Run("AddMessageWithExistingTimestamp", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-5", "req-5", "assistant-5", "", "")

		customTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		msg := &context.BufferedMessage{
			Role:      "assistant",
			Type:      "text",
			CreatedAt: customTime,
		}
		buffer.AddMessage(msg)

		messages := buffer.GetMessages()
		require.Len(t, messages, 1)
		assert.Equal(t, customTime, messages[0].CreatedAt) // Preserved
	})
}

func TestBufferAddUserInput(t *testing.T) {
	t.Run("AddStringContent", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-1", "req-1", "assistant-1", "", "")
		buffer.AddUserInput("What is the weather?", "")

		messages := buffer.GetMessages()
		require.Len(t, messages, 1)
		assert.Equal(t, "user", messages[0].Role)
		assert.Equal(t, "user_input", messages[0].Type)
		assert.Equal(t, "What is the weather?", messages[0].Props["content"])
		assert.Equal(t, "user", messages[0].Props["role"])
	})

	t.Run("AddUserInputWithName", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-2", "req-2", "assistant-2", "", "")
		buffer.AddUserInput("Hello", "John")

		messages := buffer.GetMessages()
		require.Len(t, messages, 1)
		assert.Equal(t, "John", messages[0].Props["name"])
	})

	t.Run("AddComplexContent", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-3", "req-3", "assistant-3", "", "")
		complexContent := []map[string]interface{}{
			{"type": "text", "text": "Look at this image"},
			{"type": "image_url", "image_url": map[string]string{"url": "https://example.com/image.jpg"}},
		}
		buffer.AddUserInput(complexContent, "")

		messages := buffer.GetMessages()
		require.Len(t, messages, 1)
		content, ok := messages[0].Props["content"].([]map[string]interface{})
		require.True(t, ok)
		assert.Len(t, content, 2)
	})
}

func TestBufferAddAssistantMessage(t *testing.T) {
	t.Run("AddTextMessage", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-1", "req-1", "assistant-1", "", "")
		buffer.AddAssistantMessage(
			"M1",
			"text",
			map[string]interface{}{"content": "Hello, how can I help?"},
			"block-1",
			"thread-1",
			"assistant-1",
			map[string]interface{}{"model": "gpt-4"},
		)

		messages := buffer.GetMessages()
		require.Len(t, messages, 1)
		assert.Equal(t, "M1", messages[0].MessageID)
		assert.Equal(t, "assistant", messages[0].Role)
		assert.Equal(t, "text", messages[0].Type)
		assert.Equal(t, "block-1", messages[0].BlockID)
		assert.Equal(t, "thread-1", messages[0].ThreadID)
		assert.Equal(t, "assistant-1", messages[0].AssistantID)
		assert.Equal(t, "gpt-4", messages[0].Metadata["model"])
	})

	t.Run("SkipEventMessage", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-2", "req-2", "assistant-2", "", "")
		buffer.AddAssistantMessage(
			"E1",
			"event",
			map[string]interface{}{"event": "message_start"},
			"", "", "", nil,
		)

		// Event messages should be skipped
		assert.Equal(t, 0, buffer.GetMessageCount())
	})

	t.Run("AddRetrievalMessage", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-3", "req-3", "assistant-3", "", "")
		buffer.AddAssistantMessage(
			"M2",
			"retrieval",
			map[string]interface{}{
				"sources": []map[string]interface{}{
					{"title": "Doc 1", "score": 0.95},
					{"title": "Doc 2", "score": 0.87},
				},
			},
			"block-1", "", "assistant-3", nil,
		)

		messages := buffer.GetMessages()
		require.Len(t, messages, 1)
		assert.Equal(t, "retrieval", messages[0].Type)
	})

	t.Run("AddToolCallMessage", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-4", "req-4", "assistant-4", "", "")
		buffer.AddAssistantMessage(
			"M3",
			"tool_call",
			map[string]interface{}{
				"name":      "get_weather",
				"arguments": `{"location": "San Francisco"}`,
			},
			"block-1", "", "assistant-4", nil,
		)

		messages := buffer.GetMessages()
		require.Len(t, messages, 1)
		assert.Equal(t, "tool_call", messages[0].Type)
		assert.Equal(t, "get_weather", messages[0].Props["name"])
	})

	t.Run("AddCustomTypeMessage", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-5", "req-5", "assistant-5", "", "")
		buffer.AddAssistantMessage(
			"M4",
			"custom_chart",
			map[string]interface{}{
				"chart_type": "bar",
				"data":       []int{1, 2, 3, 4, 5},
			},
			"block-1", "", "assistant-5", nil,
		)

		messages := buffer.GetMessages()
		require.Len(t, messages, 1)
		assert.Equal(t, "custom_chart", messages[0].Type)
	})
}

func TestBufferGetMessages(t *testing.T) {
	t.Run("GetMessagesReturnsSliceCopy", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-1", "req-1", "assistant-1", "", "")
		buffer.AddUserInput("Hello", "")

		messages1 := buffer.GetMessages()
		messages2 := buffer.GetMessages()

		// Slices should be different (copy of slice)
		// But pointers point to same underlying objects (shallow copy)
		assert.Len(t, messages1, 1)
		assert.Len(t, messages2, 1)
	})

	t.Run("GetEmptyMessages", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-2", "req-2", "assistant-2", "", "")
		messages := buffer.GetMessages()

		assert.NotNil(t, messages)
		assert.Empty(t, messages)
	})
}

func TestBufferGetMessageCount(t *testing.T) {
	buffer := context.NewChatBuffer("chat-1", "req-1", "assistant-1", "", "")
	assert.Equal(t, 0, buffer.GetMessageCount())

	buffer.AddUserInput("Message 1", "")
	assert.Equal(t, 1, buffer.GetMessageCount())

	buffer.AddAssistantMessage("M1", "text", map[string]interface{}{"content": "Reply"}, "", "", "", nil)
	assert.Equal(t, 2, buffer.GetMessageCount())
}

// =============================================================================
// Step Buffer Tests (for Resume)
// =============================================================================

func TestBufferBeginStep(t *testing.T) {
	t.Run("BeginStepWithStack", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-1", "req-1", "assistant-1", "", "")

		stack := &context.Stack{
			ID:       "stack-123",
			ParentID: "stack-parent-456",
			Depth:    2,
		}

		step := buffer.BeginStep(context.StepTypeLLM, map[string]interface{}{"prompt": "Hello"}, stack)

		require.NotNil(t, step)
		assert.NotEmpty(t, step.ResumeID)
		assert.Equal(t, "chat-1", step.ChatID)
		assert.Equal(t, "req-1", step.RequestID)
		assert.Equal(t, "assistant-1", step.AssistantID)
		assert.Equal(t, "stack-123", step.StackID)
		assert.Equal(t, "stack-parent-456", step.StackParentID)
		assert.Equal(t, 2, step.StackDepth)
		assert.Equal(t, context.StepTypeLLM, step.Type)
		assert.Equal(t, context.StepStatusRunning, step.Status)
		assert.Equal(t, 1, step.Sequence)
		assert.Equal(t, "Hello", step.Input["prompt"])
		assert.False(t, step.CreatedAt.IsZero())
	})

	t.Run("BeginStepWithNilStack", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-2", "req-2", "assistant-2", "", "")

		step := buffer.BeginStep(context.StepTypeInput, nil, nil)

		require.NotNil(t, step)
		assert.Empty(t, step.StackID)
		assert.Empty(t, step.StackParentID)
		assert.Equal(t, 0, step.StackDepth)
	})

	t.Run("BeginMultipleSteps", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-3", "req-3", "assistant-3", "", "")

		step1 := buffer.BeginStep(context.StepTypeInput, nil, nil)
		step2 := buffer.BeginStep(context.StepTypeHookCreate, nil, nil)
		step3 := buffer.BeginStep(context.StepTypeLLM, nil, nil)

		assert.Equal(t, 1, step1.Sequence)
		assert.Equal(t, 2, step2.Sequence)
		assert.Equal(t, 3, step3.Sequence)

		steps := buffer.GetAllSteps()
		require.Len(t, steps, 3)
	})

	t.Run("BeginStepWithSpaceSnapshot", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-4", "req-4", "assistant-4", "", "")

		// Set space snapshot before beginning step
		buffer.SetSpaceSnapshot(map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		})

		step := buffer.BeginStep(context.StepTypeLLM, nil, nil)

		require.NotNil(t, step.SpaceSnapshot)
		assert.Equal(t, "value1", step.SpaceSnapshot["key1"])
		assert.Equal(t, 42, step.SpaceSnapshot["key2"])
	})
}

func TestBufferCompleteStep(t *testing.T) {
	t.Run("CompleteCurrentStep", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-1", "req-1", "assistant-1", "", "")

		buffer.BeginStep(context.StepTypeLLM, map[string]interface{}{"prompt": "Hello"}, nil)
		buffer.CompleteStep(map[string]interface{}{"response": "Hi there!"})

		steps := buffer.GetAllSteps()
		require.Len(t, steps, 1)
		assert.Equal(t, context.StepStatusCompleted, steps[0].Status)
		assert.Equal(t, "Hi there!", steps[0].Output["response"])
		assert.Nil(t, buffer.GetCurrentStep()) // Current step cleared
	})

	t.Run("CompleteWithNoCurrentStep", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-2", "req-2", "assistant-2", "", "")

		// Should not panic
		buffer.CompleteStep(map[string]interface{}{"response": "test"})
		assert.Nil(t, buffer.GetCurrentStep())
	})

	t.Run("CompleteMultipleStepsSequentially", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-3", "req-3", "assistant-3", "", "")

		buffer.BeginStep(context.StepTypeInput, nil, nil)
		buffer.CompleteStep(map[string]interface{}{"done": true})

		buffer.BeginStep(context.StepTypeHookCreate, nil, nil)
		buffer.CompleteStep(map[string]interface{}{"hook_result": "ok"})

		buffer.BeginStep(context.StepTypeLLM, nil, nil)
		buffer.CompleteStep(map[string]interface{}{"llm_response": "hello"})

		steps := buffer.GetAllSteps()
		require.Len(t, steps, 3)
		for _, step := range steps {
			assert.Equal(t, context.StepStatusCompleted, step.Status)
		}
	})
}

func TestBufferFailCurrentStep(t *testing.T) {
	t.Run("FailWithError", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-1", "req-1", "assistant-1", "", "")

		buffer.BeginStep(context.StepTypeLLM, nil, nil)
		buffer.FailCurrentStep(context.ResumeStatusFailed, fmt.Errorf("API error: rate limit exceeded"))

		steps := buffer.GetAllSteps()
		require.Len(t, steps, 1)
		assert.Equal(t, context.ResumeStatusFailed, steps[0].Status)
		assert.Equal(t, "API error: rate limit exceeded", steps[0].Error)
	})

	t.Run("FailWithInterrupted", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-2", "req-2", "assistant-2", "", "")

		buffer.BeginStep(context.StepTypeLLM, nil, nil)
		buffer.FailCurrentStep(context.ResumeStatusInterrupted, nil)

		steps := buffer.GetAllSteps()
		require.Len(t, steps, 1)
		assert.Equal(t, context.ResumeStatusInterrupted, steps[0].Status)
		assert.Empty(t, steps[0].Error)
	})

	t.Run("FailAlreadyCompletedStep", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-3", "req-3", "assistant-3", "", "")

		buffer.BeginStep(context.StepTypeLLM, nil, nil)
		buffer.CompleteStep(map[string]interface{}{"done": true})

		// Try to fail completed step (should be no-op since currentStep is nil)
		buffer.FailCurrentStep(context.ResumeStatusFailed, fmt.Errorf("late error"))

		steps := buffer.GetAllSteps()
		require.Len(t, steps, 1)
		assert.Equal(t, context.StepStatusCompleted, steps[0].Status) // Still completed
	})

	t.Run("FailWithNoCurrentStep", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-4", "req-4", "assistant-4", "", "")

		// Should not panic
		buffer.FailCurrentStep(context.ResumeStatusFailed, fmt.Errorf("error"))
	})
}

func TestBufferGetCurrentStep(t *testing.T) {
	t.Run("NoCurrentStep", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-1", "req-1", "assistant-1", "", "")
		assert.Nil(t, buffer.GetCurrentStep())
	})

	t.Run("HasCurrentStep", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-2", "req-2", "assistant-2", "", "")
		buffer.BeginStep(context.StepTypeLLM, nil, nil)

		current := buffer.GetCurrentStep()
		require.NotNil(t, current)
		assert.Equal(t, context.StepTypeLLM, current.Type)
	})

	t.Run("CurrentStepClearedAfterComplete", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-3", "req-3", "assistant-3", "", "")
		buffer.BeginStep(context.StepTypeLLM, nil, nil)
		buffer.CompleteStep(nil)

		assert.Nil(t, buffer.GetCurrentStep())
	})
}

func TestBufferGetStepsForResume(t *testing.T) {
	t.Run("CompletedSuccessfully", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-1", "req-1", "assistant-1", "", "")

		buffer.BeginStep(context.StepTypeInput, nil, nil)
		buffer.CompleteStep(nil)
		buffer.BeginStep(context.StepTypeLLM, nil, nil)
		buffer.CompleteStep(nil)

		// Completed successfully - no steps need to be saved
		steps := buffer.GetStepsForResume(context.StepStatusCompleted)
		assert.Nil(t, steps)
	})

	t.Run("FailedRequest", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-2", "req-2", "assistant-2", "", "")

		buffer.BeginStep(context.StepTypeInput, nil, nil)
		buffer.CompleteStep(nil)
		buffer.BeginStep(context.StepTypeLLM, nil, nil)
		// Step still running when failure occurs

		steps := buffer.GetStepsForResume(context.ResumeStatusFailed)
		require.NotNil(t, steps)
		assert.Len(t, steps, 2)

		// Current step should be marked as failed
		assert.Equal(t, context.ResumeStatusFailed, steps[1].Status)
	})

	t.Run("InterruptedRequest", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-3", "req-3", "assistant-3", "", "")

		buffer.BeginStep(context.StepTypeInput, nil, nil)
		buffer.CompleteStep(nil)
		buffer.BeginStep(context.StepTypeHookCreate, nil, nil)
		buffer.CompleteStep(nil)
		buffer.BeginStep(context.StepTypeLLM, nil, nil)
		// Interrupted during LLM

		steps := buffer.GetStepsForResume(context.ResumeStatusInterrupted)
		require.NotNil(t, steps)
		assert.Len(t, steps, 3)
		assert.Equal(t, context.ResumeStatusInterrupted, steps[2].Status)
	})
}

func TestBufferGetAllSteps(t *testing.T) {
	t.Run("GetStepsReturnsSliceCopy", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-1", "req-1", "assistant-1", "", "")
		buffer.BeginStep(context.StepTypeLLM, nil, nil)

		steps1 := buffer.GetAllSteps()
		steps2 := buffer.GetAllSteps()

		// Slices should be different (copy of slice)
		assert.Len(t, steps1, 1)
		assert.Len(t, steps2, 1)
	})

	t.Run("GetEmptySteps", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-2", "req-2", "assistant-2", "", "")
		steps := buffer.GetAllSteps()

		assert.NotNil(t, steps)
		assert.Empty(t, steps)
	})
}

// =============================================================================
// Space Snapshot Tests
// =============================================================================

func TestBufferSpaceSnapshot(t *testing.T) {
	t.Run("SetAndGetSnapshot", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-1", "req-1", "assistant-1", "", "")

		snapshot := map[string]interface{}{
			"user_id":   "user-123",
			"session":   map[string]interface{}{"token": "abc"},
			"counter":   42,
			"is_active": true,
		}
		buffer.SetSpaceSnapshot(snapshot)

		retrieved := buffer.GetSpaceSnapshot()
		assert.Equal(t, "user-123", retrieved["user_id"])
		assert.Equal(t, 42, retrieved["counter"])
		assert.Equal(t, true, retrieved["is_active"])
	})

	t.Run("SnapshotIsCopy", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-2", "req-2", "assistant-2", "", "")

		original := map[string]interface{}{"key": "original"}
		buffer.SetSpaceSnapshot(original)

		// Modify original
		original["key"] = "modified"

		// Buffer should have original value
		retrieved := buffer.GetSpaceSnapshot()
		assert.Equal(t, "original", retrieved["key"])
	})

	t.Run("GetSnapshotReturnsCopy", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-3", "req-3", "assistant-3", "", "")
		buffer.SetSpaceSnapshot(map[string]interface{}{"key": "value"})

		retrieved1 := buffer.GetSpaceSnapshot()
		retrieved1["key"] = "modified"

		retrieved2 := buffer.GetSpaceSnapshot()
		assert.Equal(t, "value", retrieved2["key"]) // Original unchanged
	})

	t.Run("GetNilSnapshot", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-4", "req-4", "assistant-4", "", "")
		snapshot := buffer.GetSpaceSnapshot()
		assert.Nil(t, snapshot)
	})

	t.Run("SetNilSnapshot", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-5", "req-5", "assistant-5", "", "")
		buffer.SetSpaceSnapshot(map[string]interface{}{"key": "value"})
		buffer.SetSpaceSnapshot(nil)

		snapshot := buffer.GetSpaceSnapshot()
		assert.Nil(t, snapshot)
	})
}

// =============================================================================
// Identity Methods Tests
// =============================================================================

func TestBufferIdentityMethods(t *testing.T) {
	t.Run("SetAssistantID", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-1", "req-1", "assistant-original", "", "")

		assert.Equal(t, "assistant-original", buffer.AssistantID())

		buffer.SetAssistantID("assistant-new")
		assert.Equal(t, "assistant-new", buffer.AssistantID())
	})

	t.Run("ChatID", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-test", "req-test", "assistant-test", "", "")
		assert.Equal(t, "chat-test", buffer.ChatID())
	})

	t.Run("RequestID", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-test", "req-test", "assistant-test", "", "")
		assert.Equal(t, "req-test", buffer.RequestID())
	})

	t.Run("Connector", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-test", "req-test", "assistant-test", "openai", "")
		assert.Equal(t, "openai", buffer.Connector())
	})

	t.Run("SetConnector", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-1", "req-1", "assistant-1", "openai", "")
		assert.Equal(t, "openai", buffer.Connector())

		// Simulate user switching connector mid-conversation
		buffer.SetConnector("anthropic")
		assert.Equal(t, "anthropic", buffer.Connector())
	})

	t.Run("EmptyConnector", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-test", "req-test", "assistant-test", "", "")
		assert.Equal(t, "", buffer.Connector())
	})
}

func TestBufferConnectorInMessages(t *testing.T) {
	t.Run("MessageInheritsConnector", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-1", "req-1", "assistant-1", "openai", "")

		// Add assistant message - should inherit connector from buffer
		buffer.AddAssistantMessage(
			"M1",
			"text",
			map[string]interface{}{"content": "Hello"},
			"block-1", "thread-1", "assistant-1", nil,
		)

		messages := buffer.GetMessages()
		require.Len(t, messages, 1)
		assert.Equal(t, "openai", messages[0].Connector)
	})

	t.Run("MessageConnectorUpdatesWithBuffer", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-1", "req-1", "assistant-1", "openai", "")

		// First message with openai
		buffer.AddAssistantMessage(
			"M1",
			"text",
			map[string]interface{}{"content": "Using OpenAI"},
			"", "", "assistant-1", nil,
		)

		// User switches connector
		buffer.SetConnector("anthropic")

		// Second message with anthropic
		buffer.AddAssistantMessage(
			"M2",
			"text",
			map[string]interface{}{"content": "Now using Claude"},
			"", "", "assistant-1", nil,
		)

		messages := buffer.GetMessages()
		require.Len(t, messages, 2)
		assert.Equal(t, "openai", messages[0].Connector, "First message should use openai")
		assert.Equal(t, "anthropic", messages[1].Connector, "Second message should use anthropic")
	})

	t.Run("UserInputDoesNotSetConnector", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-1", "req-1", "assistant-1", "openai", "")

		// User input doesn't have connector (it's set by the system based on which model processes it)
		buffer.AddUserInput("Hello", "")

		messages := buffer.GetMessages()
		require.Len(t, messages, 1)
		// User input messages don't have connector field set by AddUserInput
		// Connector is only set for assistant messages
		assert.Equal(t, "", messages[0].Connector)
	})

	t.Run("MultipleConnectorSwitches", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-1", "req-1", "assistant-1", "openai", "")

		// Simulate a conversation with multiple connector switches
		connectors := []string{"openai", "anthropic", "openai", "google"}
		for i, conn := range connectors {
			buffer.SetConnector(conn)
			buffer.AddAssistantMessage(
				fmt.Sprintf("M%d", i+1),
				"text",
				map[string]interface{}{"content": fmt.Sprintf("Message %d", i+1)},
				"", "", "assistant-1", nil,
			)
		}

		messages := buffer.GetMessages()
		require.Len(t, messages, 4)

		for i, msg := range messages {
			assert.Equal(t, connectors[i], msg.Connector, "Message %d should have connector %s", i+1, connectors[i])
		}
	})
}

// =============================================================================
// Concurrency Tests
// =============================================================================

func TestBufferConcurrentMessageOperations(t *testing.T) {
	buffer := context.NewChatBuffer("chat-concurrent", "req-concurrent", "assistant-concurrent", "", "")

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			buffer.AddMessage(&context.BufferedMessage{
				Role:  "assistant",
				Type:  "text",
				Props: map[string]interface{}{"content": fmt.Sprintf("Message %d", idx)},
			})
		}(i)
	}

	wg.Wait()

	// Verify all messages were added
	messages := buffer.GetMessages()
	assert.Len(t, messages, numGoroutines)

	// Verify sequences are unique
	sequences := make(map[int]bool)
	for _, msg := range messages {
		assert.False(t, sequences[msg.Sequence], "Duplicate sequence found: %d", msg.Sequence)
		sequences[msg.Sequence] = true
	}
}

func TestBufferConcurrentStepOperations(t *testing.T) {
	buffer := context.NewChatBuffer("chat-concurrent", "req-concurrent", "assistant-concurrent", "", "")

	var wg sync.WaitGroup
	numGoroutines := 50

	// Concurrent step operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			buffer.BeginStep(context.StepTypeLLM, map[string]interface{}{"idx": idx}, nil)
			time.Sleep(time.Millisecond) // Simulate some work
			buffer.CompleteStep(map[string]interface{}{"result": idx})
		}(i)
	}

	wg.Wait()

	// Verify all steps were recorded
	steps := buffer.GetAllSteps()
	assert.Len(t, steps, numGoroutines)
}

func TestBufferConcurrentReadWrite(t *testing.T) {
	buffer := context.NewChatBuffer("chat-rw", "req-rw", "assistant-rw", "", "")

	var wg sync.WaitGroup
	done := make(chan bool)

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			buffer.AddMessage(&context.BufferedMessage{
				Role:  "assistant",
				Type:  "text",
				Props: map[string]interface{}{"content": fmt.Sprintf("Message %d", i)},
			})
			time.Sleep(time.Microsecond)
		}
	}()

	// Reader goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				_ = buffer.GetMessages()
				_ = buffer.GetMessageCount()
				time.Sleep(time.Microsecond)
			}
		}
	}()

	// Let it run for a bit
	time.Sleep(50 * time.Millisecond)
	close(done)
	wg.Wait()

	// Should complete without race conditions
	assert.Equal(t, 100, buffer.GetMessageCount())
}

// =============================================================================
// Step Type Constants Tests
// =============================================================================

func TestBufferStepTypeConstants(t *testing.T) {
	// Verify all step types are defined
	assert.Equal(t, "input", context.StepTypeInput)
	assert.Equal(t, "hook_create", context.StepTypeHookCreate)
	assert.Equal(t, "llm", context.StepTypeLLM)
	assert.Equal(t, "tool", context.StepTypeTool)
	assert.Equal(t, "hook_next", context.StepTypeHookNext)
	assert.Equal(t, "delegate", context.StepTypeDelegate)
}

func TestBufferResumeStatusConstants(t *testing.T) {
	assert.Equal(t, "failed", context.ResumeStatusFailed)
	assert.Equal(t, "interrupted", context.ResumeStatusInterrupted)
}

func TestBufferStepStatusConstants(t *testing.T) {
	assert.Equal(t, "running", context.StepStatusRunning)
	assert.Equal(t, "completed", context.StepStatusCompleted)
}

// =============================================================================
// Edge Cases and Error Handling Tests
// =============================================================================

func TestBufferEdgeCases(t *testing.T) {
	t.Run("LargeNumberOfMessages", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-large", "req-large", "assistant-large", "", "")

		// Add 10000 messages
		for i := 0; i < 10000; i++ {
			buffer.AddMessage(&context.BufferedMessage{
				Role:  "assistant",
				Type:  "text",
				Props: map[string]interface{}{"content": fmt.Sprintf("Message %d", i)},
			})
		}

		assert.Equal(t, 10000, buffer.GetMessageCount())
		messages := buffer.GetMessages()
		assert.Len(t, messages, 10000)
	})

	t.Run("MessageWithEmptyProps", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-empty", "req-empty", "assistant-empty", "", "")

		buffer.AddMessage(&context.BufferedMessage{
			Role:  "assistant",
			Type:  "text",
			Props: nil,
		})

		messages := buffer.GetMessages()
		require.Len(t, messages, 1)
		assert.Nil(t, messages[0].Props)
	})

	t.Run("StepWithEmptyInput", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-step", "req-step", "assistant-step", "", "")

		step := buffer.BeginStep(context.StepTypeLLM, nil, nil)
		assert.Nil(t, step.Input)

		buffer.CompleteStep(nil)
		steps := buffer.GetAllSteps()
		assert.Nil(t, steps[0].Output)
	})

	t.Run("AllMessageTypes", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-types", "req-types", "assistant-types", "", "")

		messageTypes := []string{
			"text", "image", "loading", "tool_call", "tool_result",
			"retrieval", "thinking", "action", "chart", "table",
			"custom_type_1", "custom_type_2",
		}

		for i, msgType := range messageTypes {
			buffer.AddAssistantMessage(fmt.Sprintf("M%d", i+1), msgType, map[string]interface{}{"type": msgType}, "", "", "", nil)
		}

		assert.Equal(t, len(messageTypes), buffer.GetMessageCount())
	})

	t.Run("AllStepTypes", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-step-types", "req-step-types", "assistant-step-types", "", "")

		stepTypes := []string{
			context.StepTypeInput, context.StepTypeHookCreate, context.StepTypeLLM,
			context.StepTypeTool, context.StepTypeHookNext, context.StepTypeDelegate,
		}

		for _, stepType := range stepTypes {
			buffer.BeginStep(stepType, nil, nil)
			buffer.CompleteStep(nil)
		}

		steps := buffer.GetAllSteps()
		assert.Len(t, steps, len(stepTypes))
	})
}

// =============================================================================
// Integration-like Tests (Simulating Real Workflow)
// =============================================================================

func TestBufferCompleteWorkflow(t *testing.T) {
	t.Run("SuccessfulChatFlow", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-workflow", "req-workflow", "assistant-main", "", "")

		// 1. User input
		buffer.AddUserInput("What's the weather in San Francisco?", "John")
		buffer.BeginStep(context.StepTypeInput, map[string]interface{}{"content": "What's the weather in San Francisco?"}, nil)
		buffer.CompleteStep(nil)

		// 2. Create hook
		buffer.BeginStep(context.StepTypeHookCreate, nil, nil)
		buffer.AddAssistantMessage("M1", "thinking", map[string]interface{}{"content": "Processing your request..."}, "block-1", "", "assistant-main", nil)
		buffer.CompleteStep(nil)

		// 3. LLM call with tool
		buffer.BeginStep(context.StepTypeLLM, map[string]interface{}{"model": "gpt-4"}, nil)
		buffer.AddAssistantMessage("M2", "tool_call", map[string]interface{}{
			"name":      "get_weather",
			"arguments": `{"location":"San Francisco"}`,
		}, "block-2", "", "assistant-main", nil)
		buffer.CompleteStep(map[string]interface{}{"tool_calls": 1})

		// 4. Tool execution
		buffer.BeginStep(context.StepTypeTool, map[string]interface{}{"tool": "get_weather"}, nil)
		buffer.AddAssistantMessage("M3", "tool_result", map[string]interface{}{
			"result": "72°F, Sunny",
		}, "block-2", "", "assistant-main", nil)
		buffer.CompleteStep(map[string]interface{}{"result": "72°F, Sunny"})

		// 5. Final LLM response
		buffer.BeginStep(context.StepTypeLLM, nil, nil)
		buffer.AddAssistantMessage("M4", "text", map[string]interface{}{
			"content": "The weather in San Francisco is currently 72°F and sunny.",
		}, "block-3", "", "assistant-main", nil)
		buffer.CompleteStep(nil)

		// Verify: 1 user_input + 4 assistant messages (thinking, tool_call, tool_result, text)
		assert.Equal(t, 5, buffer.GetMessageCount())
		assert.Len(t, buffer.GetAllSteps(), 5) // 5 steps (no hook_next in this flow)

		// All steps should be completed
		steps := buffer.GetStepsForResume(context.StepStatusCompleted)
		assert.Nil(t, steps)
	})

	t.Run("InterruptedChatFlow", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-interrupted", "req-interrupted", "assistant-main", "", "")

		// Set space snapshot
		buffer.SetSpaceSnapshot(map[string]interface{}{
			"user_context": "previous conversation",
			"preferences":  map[string]interface{}{"language": "en"},
		})

		// 1. User input
		buffer.AddUserInput("Generate a long story", "")
		buffer.BeginStep(context.StepTypeInput, nil, nil)
		buffer.CompleteStep(nil)

		// 2. LLM starts generating
		buffer.BeginStep(context.StepTypeLLM, map[string]interface{}{"model": "gpt-4"}, nil)
		buffer.AddAssistantMessage("M1", "text", map[string]interface{}{"content": "Once upon a time..."}, "block-1", "", "assistant-main", nil)
		// User interrupts here!

		// Get steps for resume
		steps := buffer.GetStepsForResume(context.ResumeStatusInterrupted)
		require.NotNil(t, steps)
		assert.Len(t, steps, 2)

		// Last step should be interrupted with space snapshot
		lastStep := steps[len(steps)-1]
		assert.Equal(t, context.ResumeStatusInterrupted, lastStep.Status)
		assert.NotNil(t, lastStep.SpaceSnapshot)
		assert.Equal(t, "previous conversation", lastStep.SpaceSnapshot["user_context"])
	})

	t.Run("A2ACallWithDelegation", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-a2a", "req-a2a", "assistant-main", "", "")

		mainStack := &context.Stack{ID: "stack-main", Depth: 0}
		childStack := &context.Stack{ID: "stack-child", ParentID: "stack-main", Depth: 1}

		// Main assistant starts
		buffer.BeginStep(context.StepTypeInput, nil, mainStack)
		buffer.CompleteStep(nil)

		// Delegate to child assistant
		buffer.SetAssistantID("assistant-child")
		buffer.BeginStep(context.StepTypeDelegate, map[string]interface{}{"delegate_to": "assistant-child"}, childStack)

		// Child assistant messages
		buffer.AddAssistantMessage("M1", "text", map[string]interface{}{"content": "Child assistant responding"}, "block-child", "", "assistant-child", nil)
		buffer.CompleteStep(map[string]interface{}{"delegate_result": "success"})

		// Return to main assistant
		buffer.SetAssistantID("assistant-main")
		buffer.BeginStep(context.StepTypeLLM, nil, mainStack)
		buffer.AddAssistantMessage("M2", "text", map[string]interface{}{"content": "Main assistant continuing"}, "block-main", "", "assistant-main", nil)
		buffer.CompleteStep(nil)

		// Verify
		messages := buffer.GetMessages()
		assert.Len(t, messages, 2)
		assert.Equal(t, "assistant-child", messages[0].AssistantID)
		assert.Equal(t, "assistant-main", messages[1].AssistantID)

		steps := buffer.GetAllSteps()
		assert.Len(t, steps, 3)
		assert.Equal(t, "stack-child", steps[1].StackID)
		assert.Equal(t, "stack-main", steps[1].StackParentID)
	})

	t.Run("ConcurrentAgentCalls", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-concurrent-a2a", "req-concurrent-a2a", "assistant-main", "", "")

		// Main assistant spawns multiple concurrent calls
		buffer.BeginStep(context.StepTypeInput, nil, nil)
		buffer.CompleteStep(nil)

		// Simulate concurrent responses with thread IDs
		var wg sync.WaitGroup
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				threadID := fmt.Sprintf("thread-%d", idx)
				buffer.AddAssistantMessage(
					fmt.Sprintf("M%d", idx),
					"text",
					map[string]interface{}{"content": fmt.Sprintf("Response from thread %d", idx)},
					"block-concurrent",
					threadID,
					fmt.Sprintf("assistant-%d", idx),
					nil,
				)
			}(i)
		}
		wg.Wait()

		messages := buffer.GetMessages()
		assert.Len(t, messages, 3)

		// Verify all have same block ID but different thread IDs
		threadIDs := make(map[string]bool)
		for _, msg := range messages {
			assert.Equal(t, "block-concurrent", msg.BlockID)
			assert.False(t, threadIDs[msg.ThreadID], "Duplicate thread ID")
			threadIDs[msg.ThreadID] = true
		}
	})
}

// =============================================================================
// Message Sequence Tests
// =============================================================================

func TestBufferMessageSequence(t *testing.T) {
	t.Run("SequenceAutoIncrement", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-seq", "req-seq", "assistant-seq", "", "")

		for i := 0; i < 10; i++ {
			buffer.AddMessage(&context.BufferedMessage{
				Role: "assistant",
				Type: "text",
			})
		}

		messages := buffer.GetMessages()
		for i, msg := range messages {
			assert.Equal(t, i+1, msg.Sequence)
		}
	})

	t.Run("MixedMessageTypes", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-mixed", "req-mixed", "assistant-mixed", "", "")

		buffer.AddUserInput("Hello", "")
		buffer.AddAssistantMessage("M1", "text", nil, "", "", "", nil)
		buffer.AddUserInput("Follow up", "")
		buffer.AddAssistantMessage("M2", "tool_call", nil, "", "", "", nil)

		messages := buffer.GetMessages()
		assert.Len(t, messages, 4)
		for i, msg := range messages {
			assert.Equal(t, i+1, msg.Sequence)
		}
	})
}

// =============================================================================
// Step Sequence Tests
// =============================================================================

func TestBufferStepSequence(t *testing.T) {
	t.Run("SequenceAutoIncrement", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-step-seq", "req-step-seq", "assistant-step-seq", "", "")

		for i := 0; i < 5; i++ {
			buffer.BeginStep(context.StepTypeLLM, nil, nil)
			buffer.CompleteStep(nil)
		}

		steps := buffer.GetAllSteps()
		for i, step := range steps {
			assert.Equal(t, i+1, step.Sequence)
		}
	})
}

// =============================================================================
// Buffer Reset/Clear Tests (if needed in future)
// =============================================================================

func TestBufferMultipleRequests(t *testing.T) {
	t.Run("NewBufferPerRequest", func(t *testing.T) {
		// Simulate multiple requests with separate buffers
		buffer1 := context.NewChatBuffer("chat-1", "req-1", "assistant-1", "", "")
		buffer1.AddUserInput("Request 1", "")

		buffer2 := context.NewChatBuffer("chat-1", "req-2", "assistant-1", "", "")
		buffer2.AddUserInput("Request 2", "")

		// Buffers should be independent
		assert.Equal(t, 1, buffer1.GetMessageCount())
		assert.Equal(t, 1, buffer2.GetMessageCount())

		msg1 := buffer1.GetMessages()[0]
		msg2 := buffer2.GetMessages()[0]

		assert.Equal(t, "req-1", msg1.RequestID)
		assert.Equal(t, "req-2", msg2.RequestID)
	})
}

// =============================================================================
// Streaming Message Tests
// =============================================================================

func TestBufferStreamingMessage(t *testing.T) {
	t.Run("AddStreamingMessage", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-1", "req-1", "assistant-1", "openai", "")

		buffer.AddStreamingMessage(
			"msg-stream-1",
			"text",
			map[string]interface{}{"content": "# Title\n\n"},
			"block-1",
			"thread-1",
			"assistant-1",
			nil,
		)

		assert.Equal(t, 1, buffer.GetMessageCount())

		// Verify streaming message is added
		msg := buffer.GetStreamingMessage("msg-stream-1")
		assert.NotNil(t, msg)
		assert.Equal(t, "msg-stream-1", msg.MessageID)
		assert.Equal(t, "text", msg.Type)
		assert.Equal(t, "# Title\n\n", msg.Props["content"])
		assert.True(t, msg.IsStreaming)
	})

	t.Run("AppendMessageContent", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-1", "req-1", "assistant-1", "openai", "")

		// Add streaming message
		buffer.AddStreamingMessage(
			"msg-stream-2",
			"text",
			map[string]interface{}{"content": "Initial "},
			"", "", "", nil,
		)

		// Append content
		ok := buffer.AppendMessageContent("msg-stream-2", "Line 1\n")
		assert.True(t, ok)

		ok = buffer.AppendMessageContent("msg-stream-2", "Line 2\n")
		assert.True(t, ok)

		// Verify accumulated content
		msg := buffer.GetStreamingMessage("msg-stream-2")
		assert.NotNil(t, msg)
		assert.Equal(t, "Initial Line 1\nLine 2\n", msg.Props["content"])
	})

	t.Run("AppendToNonExistentMessage", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-1", "req-1", "assistant-1", "openai", "")

		// Try to append to non-existent message
		ok := buffer.AppendMessageContent("non-existent", "content")
		assert.False(t, ok)
	})

	t.Run("AppendToCompletedMessage", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-1", "req-1", "assistant-1", "openai", "")

		// Add and complete streaming message
		buffer.AddStreamingMessage(
			"msg-stream-3",
			"text",
			map[string]interface{}{"content": "Initial"},
			"", "", "", nil,
		)
		buffer.CompleteStreamingMessage("msg-stream-3")

		// Try to append to completed message (should fail)
		ok := buffer.AppendMessageContent("msg-stream-3", " more")
		assert.False(t, ok)
	})

	t.Run("CompleteStreamingMessage", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-1", "req-1", "assistant-1", "openai", "")

		// Add streaming message
		buffer.AddStreamingMessage(
			"msg-stream-4",
			"text",
			map[string]interface{}{"content": "Hello "},
			"", "", "", nil,
		)

		// Append content
		buffer.AppendMessageContent("msg-stream-4", "World!")

		// Complete the message
		content, ok := buffer.CompleteStreamingMessage("msg-stream-4")
		assert.True(t, ok)
		assert.Equal(t, "Hello World!", content)

		// Message should no longer be streaming
		msg := buffer.GetStreamingMessage("msg-stream-4")
		assert.Nil(t, msg)

		// But should still exist in messages
		messages := buffer.GetMessages()
		assert.Equal(t, 1, len(messages))
		assert.False(t, messages[0].IsStreaming)
	})

	t.Run("CompleteNonExistentMessage", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-1", "req-1", "assistant-1", "openai", "")

		content, ok := buffer.CompleteStreamingMessage("non-existent")
		assert.False(t, ok)
		assert.Empty(t, content)
	})

	t.Run("StreamingMessageWorkflow", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-1", "req-1", "assistant-1", "deepseek", "")

		// Simulate a typical streaming workflow:
		// 1. SendStream sends initial content
		buffer.AddStreamingMessage(
			"msg-workflow",
			"text",
			map[string]interface{}{"content": "# Available Tests\n\n"},
			"block-main",
			"",
			"assistant-1",
			nil,
		)

		// 2. Multiple Append calls add content
		buffer.AppendMessageContent("msg-workflow", "Send one of these keywords:\n\n")
		buffer.AppendMessageContent("msg-workflow", "- **basic** - Basic tests\n")
		buffer.AppendMessageContent("msg-workflow", "- **advanced** - Advanced tests\n")

		// 3. End completes the message
		finalContent, ok := buffer.CompleteStreamingMessage("msg-workflow")
		assert.True(t, ok)

		expectedContent := "# Available Tests\n\nSend one of these keywords:\n\n- **basic** - Basic tests\n- **advanced** - Advanced tests\n"
		assert.Equal(t, expectedContent, finalContent)

		// Verify final message state
		messages := buffer.GetMessages()
		assert.Equal(t, 1, len(messages))
		assert.Equal(t, "msg-workflow", messages[0].MessageID)
		assert.Equal(t, "deepseek", messages[0].Connector) // Connector should be set
		assert.False(t, messages[0].IsStreaming)
	})

	t.Run("MixedStreamingAndRegularMessages", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-1", "req-1", "assistant-1", "openai", "")

		// Add user input (regular)
		buffer.AddUserInput("Hello", "user1")

		// Add streaming assistant message
		buffer.AddStreamingMessage(
			"msg-stream",
			"text",
			map[string]interface{}{"content": "Hi "},
			"", "", "", nil,
		)
		buffer.AppendMessageContent("msg-stream", "there!")
		buffer.CompleteStreamingMessage("msg-stream")

		// Add regular assistant message
		buffer.AddAssistantMessage("M3", "text", map[string]interface{}{"content": "How can I help?"}, "", "", "", nil)

		// Verify all messages
		messages := buffer.GetMessages()
		assert.Equal(t, 3, len(messages))

		// Check sequence
		assert.Equal(t, 1, messages[0].Sequence)
		assert.Equal(t, 2, messages[1].Sequence)
		assert.Equal(t, 3, messages[2].Sequence)

		// Check content
		assert.Equal(t, "user", messages[0].Role)
		assert.Equal(t, "Hi there!", messages[1].Props["content"])
		assert.Equal(t, "How can I help?", messages[2].Props["content"])
	})

	t.Run("StreamingMessageWithEmptyInitialContent", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-1", "req-1", "assistant-1", "openai", "")

		// Add streaming message with nil props
		buffer.AddStreamingMessage(
			"msg-empty",
			"text",
			nil,
			"", "", "", nil,
		)

		// Append content
		buffer.AppendMessageContent("msg-empty", "Content")

		// Complete
		content, ok := buffer.CompleteStreamingMessage("msg-empty")
		assert.True(t, ok)
		assert.Equal(t, "Content", content)
	})

	t.Run("ConcurrentStreamingOperations", func(t *testing.T) {
		buffer := context.NewChatBuffer("chat-1", "req-1", "assistant-1", "openai", "")

		// Add streaming message
		buffer.AddStreamingMessage(
			"msg-concurrent",
			"text",
			map[string]interface{}{"content": ""},
			"", "", "", nil,
		)

		// Concurrent appends with fixed-length content
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				buffer.AppendMessageContent("msg-concurrent", "x")
			}()
		}
		wg.Wait()

		// Complete
		content, ok := buffer.CompleteStreamingMessage("msg-concurrent")
		assert.True(t, ok)

		// Content should have 100 'x' characters
		assert.Equal(t, 100, len(content))
	})
}
