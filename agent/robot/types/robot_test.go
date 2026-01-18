package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/robot/types"
)

func TestRobotCanRun(t *testing.T) {
	t.Run("can run when under quota", func(t *testing.T) {
		robot := &types.Robot{
			Config: &types.Config{
				Quota: &types.Quota{Max: 2},
			},
		}
		assert.True(t, robot.CanRun())
	})

	t.Run("can run with nil config (uses default quota)", func(t *testing.T) {
		robot := &types.Robot{
			Config: nil, // nil config should not panic
		}
		// Should not panic and use default max (2)
		assert.True(t, robot.CanRun())
	})

	t.Run("can run with nil quota (uses default)", func(t *testing.T) {
		robot := &types.Robot{
			Config: &types.Config{
				Quota: nil, // nil quota should use default
			},
		}
		assert.True(t, robot.CanRun())
	})

	t.Run("cannot run when at quota", func(t *testing.T) {
		robot := &types.Robot{
			Config: &types.Config{
				Quota: &types.Quota{Max: 2},
			},
		}

		// Add 2 executions to reach quota
		exec1 := &types.Execution{ID: "exec1"}
		exec2 := &types.Execution{ID: "exec2"}
		robot.AddExecution(exec1)
		robot.AddExecution(exec2)

		assert.False(t, robot.CanRun())
	})

	t.Run("can run after removing execution", func(t *testing.T) {
		robot := &types.Robot{
			Config: &types.Config{
				Quota: &types.Quota{Max: 2},
			},
		}

		exec1 := &types.Execution{ID: "exec1"}
		exec2 := &types.Execution{ID: "exec2"}
		robot.AddExecution(exec1)
		robot.AddExecution(exec2)

		assert.False(t, robot.CanRun())

		robot.RemoveExecution("exec1")
		assert.True(t, robot.CanRun())
	})
}

func TestRobotRunningCount(t *testing.T) {
	robot := &types.Robot{
		Config: &types.Config{
			Quota: &types.Quota{Max: 5},
		},
	}

	assert.Equal(t, 0, robot.RunningCount())

	exec1 := &types.Execution{ID: "exec1"}
	robot.AddExecution(exec1)
	assert.Equal(t, 1, robot.RunningCount())

	exec2 := &types.Execution{ID: "exec2"}
	robot.AddExecution(exec2)
	assert.Equal(t, 2, robot.RunningCount())

	robot.RemoveExecution("exec1")
	assert.Equal(t, 1, robot.RunningCount())

	robot.RemoveExecution("exec2")
	assert.Equal(t, 0, robot.RunningCount())
}

func TestRobotAddExecution(t *testing.T) {
	robot := &types.Robot{
		Config: &types.Config{
			Quota: &types.Quota{Max: 2},
		},
	}

	exec := &types.Execution{
		ID:       "exec1",
		MemberID: "member1",
	}

	robot.AddExecution(exec)
	assert.Equal(t, 1, robot.RunningCount())

	retrieved := robot.GetExecution("exec1")
	assert.NotNil(t, retrieved)
	assert.Equal(t, "exec1", retrieved.ID)
	assert.Equal(t, "member1", retrieved.MemberID)
}

func TestRobotRemoveExecution(t *testing.T) {
	robot := &types.Robot{
		Config: &types.Config{
			Quota: &types.Quota{Max: 2},
		},
	}

	exec := &types.Execution{ID: "exec1"}
	robot.AddExecution(exec)
	assert.Equal(t, 1, robot.RunningCount())

	robot.RemoveExecution("exec1")
	assert.Equal(t, 0, robot.RunningCount())

	retrieved := robot.GetExecution("exec1")
	assert.Nil(t, retrieved)
}

func TestRobotGetExecution(t *testing.T) {
	robot := &types.Robot{
		Config: &types.Config{
			Quota: &types.Quota{Max: 2},
		},
	}

	t.Run("get existing execution", func(t *testing.T) {
		exec := &types.Execution{
			ID:       "exec1",
			MemberID: "member1",
		}
		robot.AddExecution(exec)

		retrieved := robot.GetExecution("exec1")
		assert.NotNil(t, retrieved)
		assert.Equal(t, "exec1", retrieved.ID)
	})

	t.Run("get non-existing execution", func(t *testing.T) {
		retrieved := robot.GetExecution("non-existing")
		assert.Nil(t, retrieved)
	})
}

func TestRobotGetExecutions(t *testing.T) {
	robot := &types.Robot{
		Config: &types.Config{
			Quota: &types.Quota{Max: 5},
		},
	}

	t.Run("empty executions", func(t *testing.T) {
		execs := robot.GetExecutions()
		assert.Empty(t, execs)
	})

	t.Run("multiple executions", func(t *testing.T) {
		exec1 := &types.Execution{ID: "exec1"}
		exec2 := &types.Execution{ID: "exec2"}
		exec3 := &types.Execution{ID: "exec3"}

		robot.AddExecution(exec1)
		robot.AddExecution(exec2)
		robot.AddExecution(exec3)

		execs := robot.GetExecutions()
		assert.Len(t, execs, 3)

		// Check all executions are present
		ids := make(map[string]bool)
		for _, exec := range execs {
			ids[exec.ID] = true
		}
		assert.True(t, ids["exec1"])
		assert.True(t, ids["exec2"])
		assert.True(t, ids["exec3"])
	})
}

func TestRobotConcurrentAccess(t *testing.T) {
	// Test thread-safe execution management
	robot := &types.Robot{
		Config: &types.Config{
			Quota: &types.Quota{Max: 10},
		},
	}

	// Add executions concurrently
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func(id int) {
			exec := &types.Execution{ID: string(rune('0' + id))}
			robot.AddExecution(exec)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	// Verify count
	count := robot.RunningCount()
	assert.Equal(t, 5, count)

	// Remove executions concurrently
	for i := 0; i < 5; i++ {
		go func(id int) {
			robot.RemoveExecution(string(rune('0' + id)))
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	// Verify count
	count = robot.RunningCount()
	assert.Equal(t, 0, count)
}

func TestRobotTryAcquireSlot(t *testing.T) {
	t.Run("acquire slot when under quota", func(t *testing.T) {
		robot := &types.Robot{
			Config: &types.Config{
				Quota: &types.Quota{Max: 2},
			},
		}

		exec := &types.Execution{ID: "exec1"}
		acquired := robot.TryAcquireSlot(exec)

		assert.True(t, acquired)
		assert.Equal(t, 1, robot.RunningCount())
		assert.NotNil(t, robot.GetExecution("exec1"))
	})

	t.Run("fail to acquire when at quota", func(t *testing.T) {
		robot := &types.Robot{
			Config: &types.Config{
				Quota: &types.Quota{Max: 2},
			},
		}

		// Fill quota
		robot.TryAcquireSlot(&types.Execution{ID: "exec1"})
		robot.TryAcquireSlot(&types.Execution{ID: "exec2"})

		// Try to acquire one more
		exec3 := &types.Execution{ID: "exec3"}
		acquired := robot.TryAcquireSlot(exec3)

		assert.False(t, acquired)
		assert.Equal(t, 2, robot.RunningCount())
		assert.Nil(t, robot.GetExecution("exec3"))
	})

	t.Run("acquire with nil config uses default quota", func(t *testing.T) {
		robot := &types.Robot{
			Config: nil, // default quota is 2
		}

		exec1 := &types.Execution{ID: "exec1"}
		exec2 := &types.Execution{ID: "exec2"}
		exec3 := &types.Execution{ID: "exec3"}

		assert.True(t, robot.TryAcquireSlot(exec1))
		assert.True(t, robot.TryAcquireSlot(exec2))
		assert.False(t, robot.TryAcquireSlot(exec3)) // should fail at default max=2
	})
}

func TestRobotTryAcquireSlotConcurrent(t *testing.T) {
	// Test that TryAcquireSlot is atomic and prevents exceeding quota
	robot := &types.Robot{
		Config: &types.Config{
			Quota: &types.Quota{Max: 5},
		},
	}

	// Launch 20 goroutines trying to acquire slots
	successCount := make(chan bool, 20)
	for i := 0; i < 20; i++ {
		go func(id int) {
			exec := &types.Execution{ID: string(rune('A' + id))}
			success := robot.TryAcquireSlot(exec)
			successCount <- success
		}(i)
	}

	// Count successes
	acquired := 0
	for i := 0; i < 20; i++ {
		if <-successCount {
			acquired++
		}
	}

	// Should have exactly 5 successful acquisitions (quota max)
	assert.Equal(t, 5, acquired, "Should acquire exactly quota max slots")
	assert.Equal(t, 5, robot.RunningCount(), "Running count should match quota max")
}

func TestRobotTryAcquireSlotRaceCondition(t *testing.T) {
	// Stress test to verify no race condition in TryAcquireSlot
	for iteration := 0; iteration < 100; iteration++ {
		robot := &types.Robot{
			Config: &types.Config{
				Quota: &types.Quota{Max: 3},
			},
		}

		// Launch many goroutines simultaneously
		successCount := make(chan bool, 50)
		for i := 0; i < 50; i++ {
			go func(id int) {
				exec := &types.Execution{ID: string(rune('A'+id%26)) + string(rune('0'+id/26))}
				success := robot.TryAcquireSlot(exec)
				successCount <- success
			}(i)
		}

		// Count successes
		acquired := 0
		for i := 0; i < 50; i++ {
			if <-successCount {
				acquired++
			}
		}

		// Should never exceed quota
		assert.Equal(t, 3, acquired, "Iteration %d: Should acquire exactly quota max slots", iteration)
		assert.Equal(t, 3, robot.RunningCount(), "Iteration %d: Running count should match quota max", iteration)
	}
}

func TestExecutionStructure(t *testing.T) {
	t.Run("execution with all fields", func(t *testing.T) {
		exec := &types.Execution{
			ID:          "exec1",
			MemberID:    "member1",
			TeamID:      "team1",
			TriggerType: types.TriggerClock,
			Status:      types.ExecRunning,
			Phase:       types.PhaseGoals,
			JobID:       "job1",
		}

		assert.Equal(t, "exec1", exec.ID)
		assert.Equal(t, "member1", exec.MemberID)
		assert.Equal(t, "team1", exec.TeamID)
		assert.Equal(t, types.TriggerClock, exec.TriggerType)
		assert.Equal(t, types.ExecRunning, exec.Status)
		assert.Equal(t, types.PhaseGoals, exec.Phase)
		assert.Equal(t, "job1", exec.JobID)
	})

	t.Run("execution with trigger input", func(t *testing.T) {
		exec := &types.Execution{
			ID: "exec1",
			Input: &types.TriggerInput{
				Action: types.ActionTaskAdd,
				UserID: "user1",
			},
		}

		assert.NotNil(t, exec.Input)
		assert.Equal(t, types.ActionTaskAdd, exec.Input.Action)
		assert.Equal(t, "user1", exec.Input.UserID)
	})
}

func TestTaskStructure(t *testing.T) {
	task := &types.Task{
		ID:           "task1",
		GoalRef:      "Goal 1",
		Source:       types.TaskSourceAuto,
		ExecutorType: types.ExecutorAssistant,
		ExecutorID:   "assistant1",
		Status:       types.TaskPending,
		Order:        0,
		// P3 validation fields
		ExpectedOutput: "JSON with sales_total and growth_rate fields",
		ValidationRules: []string{
			// Natural language rules (matched by validator)
			"output must be valid JSON",
			"must contain 'sales_total'",
			// Structured rule: check field type
			`{"type": "type", "path": "growth_rate", "value": "number"}`,
		},
	}

	assert.Equal(t, "task1", task.ID)
	assert.Equal(t, "Goal 1", task.GoalRef)
	assert.Equal(t, types.TaskSourceAuto, task.Source)
	assert.Equal(t, types.ExecutorAssistant, task.ExecutorType)
	assert.Equal(t, "assistant1", task.ExecutorID)
	assert.Equal(t, types.TaskPending, task.Status)
	assert.Equal(t, 0, task.Order)
	// Validation fields
	assert.Contains(t, task.ExpectedOutput, "sales_total")
	assert.Len(t, task.ValidationRules, 3)
}

func TestGoalsStructure(t *testing.T) {
	goals := &types.Goals{
		Content: "## Goals\n1. [High] Complete project\n2. [Normal] Review code",
		Delivery: &types.DeliveryTarget{
			Type:       types.DeliveryEmail,
			Recipients: []string{"team@example.com"},
			Format:     "markdown",
		},
	}

	assert.Contains(t, goals.Content, "Goals")
	assert.Contains(t, goals.Content, "Complete project")
	assert.NotNil(t, goals.Delivery)
	assert.Equal(t, types.DeliveryEmail, goals.Delivery.Type)
}

func TestTaskResultStructure(t *testing.T) {
	result := &types.TaskResult{
		TaskID:   "task1",
		Success:  true,
		Output:   "Task completed successfully",
		Duration: 1500,
		Validation: &types.ValidationResult{
			Passed: true,
			Score:  0.98,
		},
	}

	assert.Equal(t, "task1", result.TaskID)
	assert.True(t, result.Success)
	assert.Equal(t, "Task completed successfully", result.Output)
	assert.Equal(t, int64(1500), result.Duration)
	assert.NotNil(t, result.Validation)
	assert.True(t, result.Validation.Passed)
	assert.Equal(t, 0.98, result.Validation.Score)
}

func TestValidationResultStructure(t *testing.T) {
	validation := &types.ValidationResult{
		Passed:      false,
		Score:       0.45,
		Issues:      []string{"Missing required field: sales_total", "Growth rate is negative"},
		Suggestions: []string{"Add sales_total calculation", "Verify data source"},
		Details:     "Detailed validation report...",
	}

	assert.False(t, validation.Passed)
	assert.Equal(t, 0.45, validation.Score)
	assert.Len(t, validation.Issues, 2)
	assert.Contains(t, validation.Issues[0], "sales_total")
	assert.Len(t, validation.Suggestions, 2)
}

func TestValidationResultMultiTurnFields(t *testing.T) {
	// Test new multi-turn conversation control fields
	t.Run("complete and passed", func(t *testing.T) {
		validation := &types.ValidationResult{
			Passed:   true,
			Score:    0.95,
			Complete: true,
		}
		assert.True(t, validation.Passed)
		assert.True(t, validation.Complete)
		assert.False(t, validation.NeedReply)
		assert.Empty(t, validation.ReplyContent)
	})

	t.Run("passed but not complete - needs reply", func(t *testing.T) {
		validation := &types.ValidationResult{
			Passed:       true,
			Score:        0.7,
			Complete:     false,
			NeedReply:    true,
			ReplyContent: "Please continue and provide the complete result.",
		}
		assert.True(t, validation.Passed)
		assert.False(t, validation.Complete)
		assert.True(t, validation.NeedReply)
		assert.NotEmpty(t, validation.ReplyContent)
	})

	t.Run("failed with suggestions - needs reply", func(t *testing.T) {
		validation := &types.ValidationResult{
			Passed:       false,
			Score:        0.3,
			Complete:     false,
			Issues:       []string{"Missing required field"},
			Suggestions:  []string{"Add the field"},
			NeedReply:    true,
			ReplyContent: "## Validation Feedback\n\nPlease fix: Missing required field",
		}
		assert.False(t, validation.Passed)
		assert.False(t, validation.Complete)
		assert.True(t, validation.NeedReply)
		assert.Contains(t, validation.ReplyContent, "Validation Feedback")
	})

	t.Run("failed without suggestions - no reply", func(t *testing.T) {
		validation := &types.ValidationResult{
			Passed:    false,
			Score:     0.0,
			Complete:  false,
			Issues:    []string{"Critical error: invalid format"},
			NeedReply: false,
		}
		assert.False(t, validation.Passed)
		assert.False(t, validation.Complete)
		assert.False(t, validation.NeedReply)
		assert.Empty(t, validation.ReplyContent)
	})
}

func TestDeliveryResultStructure(t *testing.T) {
	sentAt := time.Now()
	delivery := &types.DeliveryResult{
		RequestID: "req-12345",
		Content: &types.DeliveryContent{
			Summary: "Weekly sales report completed",
			Body:    "# Weekly Report\n\nSales increased by 20%...",
			Attachments: []types.DeliveryAttachment{
				{
					Title:       "Sales Report",
					Description: "Detailed sales analysis",
					TaskID:      "task-1",
					File:        "__s3://report-12345.pdf",
				},
			},
		},
		Results: []types.ChannelResult{
			{
				Type:    types.DeliveryEmail,
				Target:  "user@example.com",
				Success: true,
				Details: map[string]interface{}{
					"message_id": "msg-12345",
				},
			},
			{
				Type:    types.DeliveryWebhook,
				Target:  "https://webhook.example.com/notify",
				Success: true,
			},
		},
		Success: true,
		SentAt:  &sentAt,
	}

	assert.Equal(t, "req-12345", delivery.RequestID)
	assert.True(t, delivery.Success)
	assert.NotNil(t, delivery.Content)
	assert.Equal(t, "Weekly sales report completed", delivery.Content.Summary)
	assert.Contains(t, delivery.Content.Body, "Weekly Report")
	assert.Len(t, delivery.Content.Attachments, 1)
	assert.Equal(t, "__s3://report-12345.pdf", delivery.Content.Attachments[0].File)
	assert.Len(t, delivery.Results, 2)
	assert.Equal(t, types.DeliveryEmail, delivery.Results[0].Type)
	assert.NotNil(t, delivery.SentAt)
}

func TestDeliveryContentStructure(t *testing.T) {
	content := &types.DeliveryContent{
		Summary: "Task execution completed successfully",
		Body:    "# Execution Report\n\n## Summary\n- 3 tasks completed\n- 1 task failed",
		Attachments: []types.DeliveryAttachment{
			{
				Title:       "Analysis Results",
				Description: "JSON data from analysis task",
				TaskID:      "task-analysis",
				File:        "__local://files/analysis-result.json",
			},
			{
				Title:  "Generated Chart",
				TaskID: "task-chart",
				File:   "__s3://charts/sales-chart.png",
			},
		},
	}

	assert.NotEmpty(t, content.Summary)
	assert.Contains(t, content.Body, "Execution Report")
	assert.Len(t, content.Attachments, 2)
	assert.Equal(t, "Analysis Results", content.Attachments[0].Title)
	assert.Equal(t, "task-analysis", content.Attachments[0].TaskID)
}

func TestDeliveryAttachmentStructure(t *testing.T) {
	attachment := &types.DeliveryAttachment{
		Title:       "Sales Report PDF",
		Description: "Monthly sales analysis report",
		TaskID:      "task-report",
		File:        "__s3://reports/sales-2024-01.pdf",
	}

	assert.Equal(t, "Sales Report PDF", attachment.Title)
	assert.Equal(t, "Monthly sales analysis report", attachment.Description)
	assert.Equal(t, "task-report", attachment.TaskID)
	assert.Contains(t, attachment.File, "__s3://")
}

func TestDeliveryRequestStructure(t *testing.T) {
	request := &types.DeliveryRequest{
		Content: &types.DeliveryContent{
			Summary: "Report ready",
			Body:    "# Report\n\nDetails...",
		},
		Context: &types.DeliveryContext{
			MemberID:    "member-123",
			ExecutionID: "exec-456",
			TriggerType: types.TriggerClock,
			TeamID:      "team-789",
		},
	}

	assert.NotNil(t, request.Content)
	assert.NotNil(t, request.Context)
	assert.Equal(t, "member-123", request.Context.MemberID)
	assert.Equal(t, "exec-456", request.Context.ExecutionID)
	assert.Equal(t, types.TriggerClock, request.Context.TriggerType)
}

func TestDeliveryPreferencesStructure(t *testing.T) {
	prefs := &types.DeliveryPreferences{
		Email: &types.EmailPreference{
			Enabled: true,
			Targets: []types.EmailTarget{
				{
					To:       []string{"team@example.com"},
					Template: "weekly-report",
					Subject:  "Weekly Report - {{.Date}}",
				},
				{
					To: []string{"backup@example.com"},
				},
			},
		},
		Webhook: &types.WebhookPreference{
			Enabled: true,
			Targets: []types.WebhookTarget{
				{
					URL:    "https://api.example.com/webhook",
					Method: "POST",
					Headers: map[string]string{
						"X-API-Key": "secret-key",
					},
					Secret: "signing-secret",
				},
			},
		},
		Process: &types.ProcessPreference{
			Enabled: true,
			Targets: []types.ProcessTarget{
				{
					Process: "scripts.notify.slack",
					Args:    []any{"#general", "Report ready"},
				},
			},
		},
	}

	// Email
	assert.True(t, prefs.Email.Enabled)
	assert.Len(t, prefs.Email.Targets, 2)
	assert.Equal(t, "weekly-report", prefs.Email.Targets[0].Template)
	assert.Len(t, prefs.Email.Targets[0].To, 1)

	// Webhook
	assert.True(t, prefs.Webhook.Enabled)
	assert.Len(t, prefs.Webhook.Targets, 1)
	assert.Equal(t, "https://api.example.com/webhook", prefs.Webhook.Targets[0].URL)
	assert.Equal(t, "POST", prefs.Webhook.Targets[0].Method)

	// Process
	assert.True(t, prefs.Process.Enabled)
	assert.Len(t, prefs.Process.Targets, 1)
	assert.Equal(t, "scripts.notify.slack", prefs.Process.Targets[0].Process)
	assert.Len(t, prefs.Process.Targets[0].Args, 2)
}

func TestChannelResultStructure(t *testing.T) {
	t.Run("email result with recipients", func(t *testing.T) {
		sentAt := time.Now()
		result := &types.ChannelResult{
			Type:       types.DeliveryEmail,
			Target:     "user@example.com",
			Success:    true,
			Recipients: []string{"user@example.com", "manager@example.com"},
			Details: map[string]interface{}{
				"message_id": "msg-123",
			},
			SentAt: &sentAt,
		}
		assert.Equal(t, types.DeliveryEmail, result.Type)
		assert.Equal(t, "user@example.com", result.Target)
		assert.True(t, result.Success)
		assert.Len(t, result.Recipients, 2)
		assert.NotNil(t, result.SentAt)
	})

	t.Run("webhook result", func(t *testing.T) {
		sentAt := time.Now()
		result := &types.ChannelResult{
			Type:    types.DeliveryWebhook,
			Target:  "https://api.example.com/webhook",
			Success: true,
			Details: map[string]interface{}{
				"status_code": 200,
				"response":    "OK",
			},
			SentAt: &sentAt,
		}
		assert.Equal(t, types.DeliveryWebhook, result.Type)
		assert.True(t, result.Success)
		assert.NotNil(t, result.SentAt)
	})

	t.Run("process result", func(t *testing.T) {
		result := &types.ChannelResult{
			Type:    types.DeliveryProcess,
			Target:  "scripts.notify.slack",
			Success: true,
			Details: map[string]interface{}{
				"output": "Message sent",
			},
		}
		assert.Equal(t, types.DeliveryProcess, result.Type)
		assert.Equal(t, "scripts.notify.slack", result.Target)
	})

	t.Run("failed result", func(t *testing.T) {
		result := &types.ChannelResult{
			Type:    types.DeliveryWebhook,
			Target:  "https://api.example.com/webhook",
			Success: false,
			Error:   "Connection refused",
		}
		assert.False(t, result.Success)
		assert.Equal(t, "Connection refused", result.Error)
	})
}

func TestDeliveryTargetStructure(t *testing.T) {
	delivery := &types.DeliveryTarget{
		Type:       types.DeliveryEmail,
		Recipients: []string{"team@example.com"},
		Format:     "markdown",
		Template:   "weekly-report",
		Options: map[string]interface{}{
			"cc": []string{"manager@example.com"},
		},
	}

	assert.Equal(t, types.DeliveryEmail, delivery.Type)
	assert.Len(t, delivery.Recipients, 1)
	assert.Equal(t, "markdown", delivery.Format)
	assert.Equal(t, "weekly-report", delivery.Template)
}

func TestLearningEntryStructure(t *testing.T) {
	entry := &types.LearningEntry{
		Type:    types.LearnExecution,
		Content: "Successfully completed task using assistant",
		Tags:    []string{"success", "assistant"},
		Meta: map[string]interface{}{
			"duration": 1500,
			"phase":    "run",
		},
	}

	assert.Equal(t, types.LearnExecution, entry.Type)
	assert.Equal(t, "Successfully completed task using assistant", entry.Content)
	assert.Len(t, entry.Tags, 2)
	assert.NotNil(t, entry.Meta)
}
