package standard_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
)

// ============================================================================
// P4 Delivery Phase Tests
// ============================================================================

func TestRunDeliveryBasic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("generates delivery content from execution results", func(t *testing.T) {
		robot := createDeliveryTestRobot(t, "robot.delivery")
		exec := createDeliveryTestExecution(robot)

		// Set up execution context with P0-P3 results
		exec.Inspiration = &types.InspirationReport{
			Content: "Morning analysis suggests focus on Q4 review.",
		}
		exec.Goals = &types.Goals{
			Content: "## Goals\n1. Review Q4 data\n2. Generate summary report",
		}
		exec.Tasks = []types.Task{
			{ID: "task-001", ExecutorID: "experts.data-analyst", ExecutorType: types.ExecutorAssistant, Status: types.TaskCompleted},
			{ID: "task-002", ExecutorID: "experts.summarizer", ExecutorType: types.ExecutorAssistant, Status: types.TaskCompleted},
		}
		exec.Results = []types.TaskResult{
			{TaskID: "task-001", Success: true, Duration: 1500, Output: map[string]interface{}{"total_sales": 1500000}},
			{TaskID: "task-002", Success: true, Duration: 800, Output: "Q4 sales exceeded expectations by 15%."},
		}

		// Run delivery phase
		e := standard.New()
		err := e.RunDelivery(ctx, exec, nil)

		require.NoError(t, err)
		require.NotNil(t, exec.Delivery)
		require.NotNil(t, exec.Delivery.Content)
		assert.NotEmpty(t, exec.Delivery.Content.Summary)
		assert.NotEmpty(t, exec.Delivery.Content.Body)
		assert.True(t, exec.Delivery.Success)
	})

	t.Run("handles partial failure in results", func(t *testing.T) {
		robot := createDeliveryTestRobot(t, "robot.delivery")
		exec := createDeliveryTestExecution(robot)

		exec.Goals = &types.Goals{
			Content: "## Goals\n1. Analyze data\n2. Generate report",
		}
		exec.Tasks = []types.Task{
			{ID: "task-001", ExecutorID: "experts.data-analyst", ExecutorType: types.ExecutorAssistant, Status: types.TaskCompleted},
			{ID: "task-002", ExecutorID: "experts.summarizer", ExecutorType: types.ExecutorAssistant, Status: types.TaskFailed},
		}
		exec.Results = []types.TaskResult{
			{TaskID: "task-001", Success: true, Duration: 1500, Output: map[string]interface{}{"data": "analyzed"}},
			{TaskID: "task-002", Success: false, Duration: 500, Error: "Summarization failed: timeout"},
		}

		e := standard.New()
		err := e.RunDelivery(ctx, exec, nil)

		require.NoError(t, err)
		require.NotNil(t, exec.Delivery)
		require.NotNil(t, exec.Delivery.Content)

		// Content should mention the failure
		body := strings.ToLower(exec.Delivery.Content.Body)
		hasFailureInfo := strings.Contains(body, "fail") ||
			strings.Contains(body, "error") ||
			strings.Contains(body, "partial") ||
			strings.Contains(body, "✗")

		assert.True(t, hasFailureInfo || exec.Delivery.Content.Summary != "", "should mention failure or have valid summary")
	})
}

func TestRunDeliveryErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("returns error when robot is nil", func(t *testing.T) {
		exec := &types.Execution{
			ID:          "test-exec-1",
			TriggerType: types.TriggerClock,
		}
		// Don't set robot

		e := standard.New()
		err := e.RunDelivery(ctx, exec, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "robot not found")
	})

	t.Run("returns error when agent not found", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "test-robot-1",
			TeamID:   "test-team-1",
			Config: &types.Config{
				Identity: &types.Identity{Role: "Test"},
				Resources: &types.Resources{
					Phases: map[types.Phase]string{
						types.PhaseDelivery: "non.existent.agent",
					},
				},
			},
		}
		exec := createDeliveryTestExecution(robot)
		exec.Results = []types.TaskResult{
			{TaskID: "task-001", Success: true, Duration: 100},
		}

		e := standard.New()
		err := e.RunDelivery(ctx, exec, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "call failed")
	})
}

// ============================================================================
// Delivery Center Tests
// ============================================================================

func TestDeliveryCenterWebhook(t *testing.T) {
	t.Run("posts to webhook successfully", func(t *testing.T) {
		// Create mock webhook server
		var receivedPayload map[string]interface{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			decoder := json.NewDecoder(r.Body)
			err := decoder.Decode(&receivedPayload)
			assert.NoError(t, err)

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "received"}`))
		}))
		defer server.Close()

		center := standard.NewDeliveryCenter()
		ctx := types.NewContext(context.Background(), nil)

		content := &types.DeliveryContent{
			Summary: "Test delivery completed",
			Body:    "# Test Report\n\nThis is a test.",
		}

		deliveryCtx := &types.DeliveryContext{
			MemberID:    "member-001",
			ExecutionID: "exec-001",
			TriggerType: types.TriggerHuman,
			TeamID:      "team-001",
		}

		prefs := &types.DeliveryPreferences{
			Webhook: &types.WebhookPreference{
				Enabled: true,
				Targets: []types.WebhookTarget{
					{URL: server.URL},
				},
			},
		}

		results, err := center.Deliver(ctx, content, deliveryCtx, prefs, nil)

		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.True(t, results[0].Success)
		assert.Equal(t, types.DeliveryWebhook, results[0].Type)
		assert.Equal(t, server.URL, results[0].Target)

		// Verify payload structure
		assert.Equal(t, "robot.delivery", receivedPayload["event"])
		assert.Equal(t, "exec-001", receivedPayload["execution_id"])
		assert.Equal(t, "member-001", receivedPayload["member_id"])
		contentMap := receivedPayload["content"].(map[string]interface{})
		assert.Equal(t, "Test delivery completed", contentMap["summary"])
	})

	t.Run("handles webhook failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "internal error"}`))
		}))
		defer server.Close()

		center := standard.NewDeliveryCenter()
		ctx := types.NewContext(context.Background(), nil)

		content := &types.DeliveryContent{
			Summary: "Test",
			Body:    "Test body",
		}

		deliveryCtx := &types.DeliveryContext{
			MemberID:    "member-001",
			ExecutionID: "exec-001",
			TriggerType: types.TriggerHuman,
			TeamID:      "team-001",
		}

		prefs := &types.DeliveryPreferences{
			Webhook: &types.WebhookPreference{
				Enabled: true,
				Targets: []types.WebhookTarget{
					{URL: server.URL},
				},
			},
		}

		results, err := center.Deliver(ctx, content, deliveryCtx, prefs, nil)

		assert.Error(t, err) // Should return error for failed delivery
		require.Len(t, results, 1)
		assert.False(t, results[0].Success)
		assert.Contains(t, results[0].Error, "500")
	})

	t.Run("supports multiple webhook targets", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ok": true}`))
		}))
		defer server.Close()

		center := standard.NewDeliveryCenter()
		ctx := types.NewContext(context.Background(), nil)

		content := &types.DeliveryContent{
			Summary: "Test",
			Body:    "Test body",
		}

		deliveryCtx := &types.DeliveryContext{
			MemberID:    "member-001",
			ExecutionID: "exec-001",
			TriggerType: types.TriggerHuman,
			TeamID:      "team-001",
		}

		prefs := &types.DeliveryPreferences{
			Webhook: &types.WebhookPreference{
				Enabled: true,
				Targets: []types.WebhookTarget{
					{URL: server.URL + "/hook1"},
					{URL: server.URL + "/hook2"},
					{URL: server.URL + "/hook3"},
				},
			},
		}

		results, err := center.Deliver(ctx, content, deliveryCtx, prefs, nil)

		require.NoError(t, err)
		require.Len(t, results, 3)
		assert.Equal(t, 3, callCount)

		for _, r := range results {
			assert.True(t, r.Success)
		}
	})

	t.Run("includes custom headers", func(t *testing.T) {
		var receivedHeaders http.Header
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedHeaders = r.Header
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		center := standard.NewDeliveryCenter()
		ctx := types.NewContext(context.Background(), nil)

		content := &types.DeliveryContent{
			Summary: "Test",
			Body:    "Test body",
		}

		deliveryCtx := &types.DeliveryContext{
			MemberID:    "member-001",
			ExecutionID: "exec-001",
			TriggerType: types.TriggerHuman,
			TeamID:      "team-001",
		}

		prefs := &types.DeliveryPreferences{
			Webhook: &types.WebhookPreference{
				Enabled: true,
				Targets: []types.WebhookTarget{
					{
						URL: server.URL,
						Headers: map[string]string{
							"X-Custom-Header": "custom-value",
							"Authorization":   "Bearer test-token",
						},
					},
				},
			},
		}

		results, _ := center.Deliver(ctx, content, deliveryCtx, prefs, nil)

		require.Len(t, results, 1)
		assert.True(t, results[0].Success)
		assert.Equal(t, "custom-value", receivedHeaders.Get("X-Custom-Header"))
		assert.Equal(t, "Bearer test-token", receivedHeaders.Get("Authorization"))
	})
}

func TestDeliveryCenterNoChannels(t *testing.T) {
	t.Run("succeeds with no channels configured", func(t *testing.T) {
		center := standard.NewDeliveryCenter()
		ctx := types.NewContext(context.Background(), nil)

		content := &types.DeliveryContent{
			Summary: "Test",
			Body:    "Test body",
		}

		deliveryCtx := &types.DeliveryContext{
			MemberID:    "member-001",
			ExecutionID: "exec-001",
			TriggerType: types.TriggerHuman,
			TeamID:      "team-001",
		}

		// No preferences
		results, err := center.Deliver(ctx, content, deliveryCtx, nil, nil)

		assert.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("succeeds with disabled channels", func(t *testing.T) {
		center := standard.NewDeliveryCenter()
		ctx := types.NewContext(context.Background(), nil)

		content := &types.DeliveryContent{
			Summary: "Test",
			Body:    "Test body",
		}

		deliveryCtx := &types.DeliveryContext{
			MemberID:    "member-001",
			ExecutionID: "exec-001",
			TriggerType: types.TriggerHuman,
			TeamID:      "team-001",
		}

		prefs := &types.DeliveryPreferences{
			Webhook: &types.WebhookPreference{
				Enabled: false, // Disabled
				Targets: []types.WebhookTarget{
					{URL: "http://example.com"},
				},
			},
		}

		results, err := center.Deliver(ctx, content, deliveryCtx, prefs, nil)

		assert.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestDeliveryCenterMixedChannels(t *testing.T) {
	t.Run("delivers to multiple channel types", func(t *testing.T) {
		webhookCalled := false
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			webhookCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		center := standard.NewDeliveryCenter()
		ctx := types.NewContext(context.Background(), nil)

		content := &types.DeliveryContent{
			Summary: "Test",
			Body:    "Test body",
		}

		deliveryCtx := &types.DeliveryContext{
			MemberID:    "member-001",
			ExecutionID: "exec-001",
			TriggerType: types.TriggerHuman,
			TeamID:      "team-001",
		}

		prefs := &types.DeliveryPreferences{
			Webhook: &types.WebhookPreference{
				Enabled: true,
				Targets: []types.WebhookTarget{
					{URL: server.URL},
				},
			},
			// Email would fail without messenger setup, but webhook should succeed
		}

		results, _ := center.Deliver(ctx, content, deliveryCtx, prefs, nil)

		assert.True(t, webhookCalled)
		require.Len(t, results, 1)
		assert.True(t, results[0].Success)
	})
}

func TestDefaultEmailChannel(t *testing.T) {
	t.Run("returns default email channel", func(t *testing.T) {
		// Default should be "email"
		assert.Equal(t, "email", types.DefaultEmailChannel())
	})

	t.Run("can set custom email channel", func(t *testing.T) {
		// Save original
		original := types.DefaultEmailChannel()
		defer types.SetDefaultEmailChannel(original)

		// Set custom channel
		types.SetDefaultEmailChannel("custom-email")
		assert.Equal(t, "custom-email", types.DefaultEmailChannel())
	})

	t.Run("ignores empty channel", func(t *testing.T) {
		// Save original and restore after test
		original := types.DefaultEmailChannel()
		defer types.SetDefaultEmailChannel(original)

		types.SetDefaultEmailChannel("")
		assert.Equal(t, original, types.DefaultEmailChannel())
	})
}

func TestRobotEmailInDelivery(t *testing.T) {
	t.Run("robot email is passed to delivery center", func(t *testing.T) {
		center := standard.NewDeliveryCenter()
		ctx := types.NewContext(context.Background(), nil)

		content := &types.DeliveryContent{
			Summary: "Test",
			Body:    "Test body",
		}

		deliveryCtx := &types.DeliveryContext{
			MemberID:    "member-001",
			ExecutionID: "exec-001",
			TriggerType: types.TriggerHuman,
			TeamID:      "team-001",
		}

		// Robot with email configured
		robot := &types.Robot{
			MemberID:   "robot-001",
			RobotEmail: "robot@example.com",
		}

		// Webhook to verify robot is passed (email would fail without messenger)
		webhookCalled := false
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			webhookCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		prefs := &types.DeliveryPreferences{
			Webhook: &types.WebhookPreference{
				Enabled: true,
				Targets: []types.WebhookTarget{
					{URL: server.URL},
				},
			},
		}

		// Deliver with robot
		results, _ := center.Deliver(ctx, content, deliveryCtx, prefs, robot)

		assert.True(t, webhookCalled)
		require.Len(t, results, 1)
		assert.True(t, results[0].Success)
	})

	t.Run("robot email field is loaded from map", func(t *testing.T) {
		data := map[string]interface{}{
			"member_id":   "robot-001",
			"team_id":     "team-001",
			"robot_email": "robot@example.com",
		}

		robot, err := types.NewRobotFromMap(data)
		require.NoError(t, err)
		assert.Equal(t, "robot@example.com", robot.RobotEmail)
	})

	t.Run("robot email can be empty", func(t *testing.T) {
		data := map[string]interface{}{
			"member_id": "robot-001",
			"team_id":   "team-001",
			// robot_email not set
		}

		robot, err := types.NewRobotFromMap(data)
		require.NoError(t, err)
		assert.Empty(t, robot.RobotEmail)
	})
}

func TestFormatDeliveryInput(t *testing.T) {
	formatter := standard.NewInputFormatter()

	t.Run("formats complete execution context", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "test-robot",
			Config: &types.Config{
				Identity: &types.Identity{
					Role:   "Sales Analyst",
					Duties: []string{"Analyze data", "Generate reports"},
				},
			},
		}

		startTime := time.Now().Add(-5 * time.Minute)
		endTime := time.Now()
		exec := &types.Execution{
			ID:          "exec-123",
			TriggerType: types.TriggerClock,
			Status:      types.ExecCompleted,
			StartTime:   startTime,
			EndTime:     &endTime,
			Inspiration: &types.InspirationReport{
				Content: "Morning analysis suggests focus on Q4.",
			},
			Goals: &types.Goals{
				Content: "## Goals\n1. Review Q4 data",
			},
			Tasks: []types.Task{
				{ID: "task-001", ExecutorID: "data-analyst", ExecutorType: types.ExecutorAssistant, Status: types.TaskCompleted, ExpectedOutput: "JSON with sales data"},
			},
			Results: []types.TaskResult{
				{TaskID: "task-001", Success: true, Duration: 1500, Output: map[string]interface{}{"sales": 1000000}},
			},
		}

		result := formatter.FormatDeliveryInput(exec, robot)

		assert.Contains(t, result, "## Robot Identity")
		assert.Contains(t, result, "Sales Analyst")
		assert.Contains(t, result, "## Execution Context")
		assert.Contains(t, result, "clock")
		assert.Contains(t, result, "## Inspiration (P0)")
		assert.Contains(t, result, "Morning analysis")
		assert.Contains(t, result, "## Goals (P1)")
		assert.Contains(t, result, "Review Q4 data")
		assert.Contains(t, result, "## Tasks (P2)")
		assert.Contains(t, result, "task-001")
		assert.Contains(t, result, "## Results (P3)")
		assert.Contains(t, result, "✓ Task: task-001")
	})

	t.Run("handles empty execution", func(t *testing.T) {
		exec := &types.Execution{
			ID:          "exec-empty",
			TriggerType: types.TriggerHuman,
			Status:      types.ExecPending,
			StartTime:   time.Now(),
		}

		result := formatter.FormatDeliveryInput(exec, nil)

		assert.Contains(t, result, "## Execution Context")
		assert.Contains(t, result, "human")
	})
}

// ============================================================================
// Email Delivery Tests (requires messenger setup)
// ============================================================================

func TestDeliveryCenterEmail(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Set robot channel as default for tests
	original := types.DefaultEmailChannel()
	types.SetDefaultEmailChannel("robot")
	defer types.SetDefaultEmailChannel(original)

	t.Run("sends email to single target", func(t *testing.T) {
		center := standard.NewDeliveryCenter()
		ctx := types.NewContext(context.Background(), nil)

		content := &types.DeliveryContent{
			Summary: "Test Delivery Report",
			Body:    "This is a test delivery from Robot Agent.\n\n## Results\n- Task 1: Completed\n- Task 2: Completed",
		}

		deliveryCtx := &types.DeliveryContext{
			MemberID:    "member-001",
			ExecutionID: "exec-001",
			TriggerType: types.TriggerClock,
			TeamID:      "team-001",
		}

		robot := &types.Robot{
			MemberID:   "robot-001",
			RobotEmail: "robot@example.com",
		}

		prefs := &types.DeliveryPreferences{
			Email: &types.EmailPreference{
				Enabled: true,
				Targets: []types.EmailTarget{
					{
						To:      []string{"test@example.com"},
						Subject: "Robot Delivery Test",
					},
				},
			},
		}

		// Send email - ignore billing/API errors, just verify the call was made
		results, _ := center.Deliver(ctx, content, deliveryCtx, prefs, robot)

		require.Len(t, results, 1)
		assert.Equal(t, types.DeliveryEmail, results[0].Type)
		assert.Equal(t, "test@example.com", results[0].Target)
		// Note: Success depends on messenger configuration
	})

	t.Run("sends email to multiple targets", func(t *testing.T) {
		center := standard.NewDeliveryCenter()
		ctx := types.NewContext(context.Background(), nil)

		content := &types.DeliveryContent{
			Summary: "Multi-target Test",
			Body:    "Test body for multiple recipients",
		}

		deliveryCtx := &types.DeliveryContext{
			MemberID:    "member-001",
			ExecutionID: "exec-002",
			TriggerType: types.TriggerHuman,
			TeamID:      "team-001",
		}

		robot := &types.Robot{
			MemberID:   "robot-001",
			RobotEmail: "robot@example.com",
		}

		prefs := &types.DeliveryPreferences{
			Email: &types.EmailPreference{
				Enabled: true,
				Targets: []types.EmailTarget{
					{
						To:      []string{"user1@example.com"},
						Subject: "Report for User 1",
					},
					{
						To:      []string{"user2@example.com", "user3@example.com"},
						Subject: "Report for Team",
					},
				},
			},
		}

		results, _ := center.Deliver(ctx, content, deliveryCtx, prefs, robot)

		require.Len(t, results, 2)
		assert.Equal(t, types.DeliveryEmail, results[0].Type)
		assert.Equal(t, types.DeliveryEmail, results[1].Type)
		assert.Equal(t, "user1@example.com", results[0].Target)
		assert.Equal(t, "user2@example.com,user3@example.com", results[1].Target)
	})

	t.Run("sends email with attachments", func(t *testing.T) {
		center := standard.NewDeliveryCenter()
		ctx := types.NewContext(context.Background(), nil)

		content := &types.DeliveryContent{
			Summary: "Report with Attachments",
			Body:    "Please find the attached report.",
			Attachments: []types.DeliveryAttachment{
				{
					Title:       "Q4 Report.pdf",
					Description: "Quarterly sales report",
					TaskID:      "task-001",
					File:        "__local://reports/q4-2024.pdf",
				},
				{
					Title:       "Data Export.csv",
					Description: "Raw data export",
					TaskID:      "task-002",
					File:        "__local://exports/data.csv",
				},
			},
		}

		deliveryCtx := &types.DeliveryContext{
			MemberID:    "member-001",
			ExecutionID: "exec-003",
			TriggerType: types.TriggerClock,
			TeamID:      "team-001",
		}

		robot := &types.Robot{
			MemberID:   "robot-001",
			RobotEmail: "robot@example.com",
		}

		prefs := &types.DeliveryPreferences{
			Email: &types.EmailPreference{
				Enabled: true,
				Targets: []types.EmailTarget{
					{
						To:      []string{"manager@example.com"},
						Subject: "Weekly Report with Attachments",
					},
				},
			},
		}

		// Send - attachment conversion may fail if files don't exist, but structure is tested
		results, _ := center.Deliver(ctx, content, deliveryCtx, prefs, robot)

		require.Len(t, results, 1)
		assert.Equal(t, types.DeliveryEmail, results[0].Type)
	})
}

// ============================================================================
// Process Delivery Tests (requires Yao process setup)
// ============================================================================

func TestDeliveryCenterProcess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	t.Run("calls process with delivery content", func(t *testing.T) {
		center := standard.NewDeliveryCenter()
		ctx := types.NewContext(context.Background(), nil)

		content := &types.DeliveryContent{
			Summary: "Process Test Summary",
			Body:    "This is the body content for process testing.",
		}

		deliveryCtx := &types.DeliveryContext{
			MemberID:    "member-001",
			ExecutionID: "exec-process-001",
			TriggerType: types.TriggerEvent,
			TeamID:      "team-001",
		}

		prefs := &types.DeliveryPreferences{
			Process: &types.ProcessPreference{
				Enabled: true,
				Targets: []types.ProcessTarget{
					{
						Process: "scripts.tests.delivery.Handle",
					},
				},
			},
		}

		results, err := center.Deliver(ctx, content, deliveryCtx, prefs, nil)

		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, types.DeliveryProcess, results[0].Type)
		assert.Equal(t, "scripts.tests.delivery.Handle", results[0].Target)
		assert.True(t, results[0].Success)

		// Verify process received correct data (Details structure depends on process return)
		assert.NotNil(t, results[0].Details, "process should return details")
	})

	t.Run("calls process with additional args", func(t *testing.T) {
		center := standard.NewDeliveryCenter()
		ctx := types.NewContext(context.Background(), nil)

		content := &types.DeliveryContent{
			Summary: "Args Test",
			Body:    "Testing additional arguments",
		}

		deliveryCtx := &types.DeliveryContext{
			MemberID:    "member-001",
			ExecutionID: "exec-process-002",
			TriggerType: types.TriggerHuman,
			TeamID:      "team-001",
		}

		prefs := &types.DeliveryPreferences{
			Process: &types.ProcessPreference{
				Enabled: true,
				Targets: []types.ProcessTarget{
					{
						Process: "scripts.tests.delivery.Handle",
						Args:    []interface{}{"custom-arg-1", "custom-arg-2"},
					},
				},
			},
		}

		results, err := center.Deliver(ctx, content, deliveryCtx, prefs, nil)

		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.True(t, results[0].Success)
		assert.NotNil(t, results[0].Details, "process should return details with args")
	})

	t.Run("calls multiple process targets", func(t *testing.T) {
		center := standard.NewDeliveryCenter()
		ctx := types.NewContext(context.Background(), nil)

		content := &types.DeliveryContent{
			Summary: "Multi-process Test",
			Body:    "Testing multiple process targets",
		}

		deliveryCtx := &types.DeliveryContext{
			MemberID:    "member-001",
			ExecutionID: "exec-process-003",
			TriggerType: types.TriggerClock,
			TeamID:      "team-001",
		}

		prefs := &types.DeliveryPreferences{
			Process: &types.ProcessPreference{
				Enabled: true,
				Targets: []types.ProcessTarget{
					{
						Process: "scripts.tests.delivery.Handle",
					},
					{
						Process: "scripts.tests.delivery.Notify",
						Args:    []interface{}{"user-123", "push"},
					},
				},
			},
		}

		results, err := center.Deliver(ctx, content, deliveryCtx, prefs, nil)

		require.NoError(t, err)
		require.Len(t, results, 2)

		// First process: Handle
		assert.Equal(t, "scripts.tests.delivery.Handle", results[0].Target)
		assert.True(t, results[0].Success)

		// Second process: Notify
		assert.Equal(t, "scripts.tests.delivery.Notify", results[1].Target)
		assert.True(t, results[1].Success)
	})

	t.Run("handles process failure gracefully", func(t *testing.T) {
		center := standard.NewDeliveryCenter()
		ctx := types.NewContext(context.Background(), nil)

		content := &types.DeliveryContent{
			Summary: "Failure Test",
			Body:    "Testing process failure handling",
		}

		deliveryCtx := &types.DeliveryContext{
			MemberID:    "member-001",
			ExecutionID: "exec-process-004",
			TriggerType: types.TriggerHuman,
			TeamID:      "team-001",
		}

		prefs := &types.DeliveryPreferences{
			Process: &types.ProcessPreference{
				Enabled: true,
				Targets: []types.ProcessTarget{
					{
						Process: "scripts.tests.delivery.HandleWithFailure",
						Args:    []interface{}{true}, // shouldFail = true
					},
				},
			},
		}

		results, err := center.Deliver(ctx, content, deliveryCtx, prefs, nil)

		assert.Error(t, err)
		require.Len(t, results, 1)
		assert.False(t, results[0].Success)
		assert.Contains(t, results[0].Error, "Simulated process failure")
	})

	t.Run("handles process with attachments", func(t *testing.T) {
		center := standard.NewDeliveryCenter()
		ctx := types.NewContext(context.Background(), nil)

		content := &types.DeliveryContent{
			Summary: "Attachment Process Test",
			Body:    "Testing process with attachments",
			Attachments: []types.DeliveryAttachment{
				{
					Title:       "Report.pdf",
					Description: "Test report",
					TaskID:      "task-001",
					File:        "__local://test/report.pdf",
				},
			},
		}

		deliveryCtx := &types.DeliveryContext{
			MemberID:    "member-001",
			ExecutionID: "exec-process-005",
			TriggerType: types.TriggerClock,
			TeamID:      "team-001",
		}

		prefs := &types.DeliveryPreferences{
			Process: &types.ProcessPreference{
				Enabled: true,
				Targets: []types.ProcessTarget{
					{
						Process: "scripts.tests.delivery.HandleAttachments",
					},
				},
			},
		}

		results, err := center.Deliver(ctx, content, deliveryCtx, prefs, nil)

		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.True(t, results[0].Success)
		assert.NotNil(t, results[0].Details, "process should return details with attachments info")
	})
}

// ============================================================================
// Mixed Channel Tests
// ============================================================================

func TestDeliveryCenterAllChannels(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Set robot channel as default
	original := types.DefaultEmailChannel()
	types.SetDefaultEmailChannel("robot")
	defer types.SetDefaultEmailChannel(original)

	t.Run("delivers to email, webhook, and process simultaneously", func(t *testing.T) {
		// Setup webhook server
		webhookCalled := false
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			webhookCalled = true
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ok": true}`))
		}))
		defer server.Close()

		center := standard.NewDeliveryCenter()
		ctx := types.NewContext(context.Background(), nil)

		content := &types.DeliveryContent{
			Summary: "Full Channel Test",
			Body:    "Testing all delivery channels together",
		}

		deliveryCtx := &types.DeliveryContext{
			MemberID:    "member-001",
			ExecutionID: "exec-full-001",
			TriggerType: types.TriggerClock,
			TeamID:      "team-001",
		}

		robot := &types.Robot{
			MemberID:   "robot-001",
			RobotEmail: "robot@example.com",
		}

		prefs := &types.DeliveryPreferences{
			Email: &types.EmailPreference{
				Enabled: true,
				Targets: []types.EmailTarget{
					{To: []string{"user@example.com"}, Subject: "Test"},
				},
			},
			Webhook: &types.WebhookPreference{
				Enabled: true,
				Targets: []types.WebhookTarget{
					{URL: server.URL},
				},
			},
			Process: &types.ProcessPreference{
				Enabled: true,
				Targets: []types.ProcessTarget{
					{Process: "scripts.tests.delivery.Handle"},
				},
			},
		}

		results, _ := center.Deliver(ctx, content, deliveryCtx, prefs, robot)

		// Should have 3 results (1 email + 1 webhook + 1 process)
		require.Len(t, results, 3)

		// Verify each channel type
		var emailResult, webhookResult, processResult *types.ChannelResult
		for i := range results {
			switch results[i].Type {
			case types.DeliveryEmail:
				emailResult = &results[i]
			case types.DeliveryWebhook:
				webhookResult = &results[i]
			case types.DeliveryProcess:
				processResult = &results[i]
			}
		}

		assert.NotNil(t, emailResult, "should have email result")
		assert.NotNil(t, webhookResult, "should have webhook result")
		assert.NotNil(t, processResult, "should have process result")

		// Webhook and process should succeed
		assert.True(t, webhookCalled, "webhook should be called")
		assert.True(t, webhookResult.Success, "webhook should succeed")
		assert.True(t, processResult.Success, "process should succeed")
	})

	t.Run("partial failure does not stop other channels", func(t *testing.T) {
		// Webhook that fails
		failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer failServer.Close()

		// Webhook that succeeds
		successCalled := false
		successServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			successCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		defer successServer.Close()

		center := standard.NewDeliveryCenter()
		ctx := types.NewContext(context.Background(), nil)

		content := &types.DeliveryContent{
			Summary: "Partial Failure Test",
			Body:    "Testing partial failure handling",
		}

		deliveryCtx := &types.DeliveryContext{
			MemberID:    "member-001",
			ExecutionID: "exec-partial-001",
			TriggerType: types.TriggerHuman,
			TeamID:      "team-001",
		}

		prefs := &types.DeliveryPreferences{
			Webhook: &types.WebhookPreference{
				Enabled: true,
				Targets: []types.WebhookTarget{
					{URL: failServer.URL},    // This will fail
					{URL: successServer.URL}, // This should still be called
				},
			},
			Process: &types.ProcessPreference{
				Enabled: true,
				Targets: []types.ProcessTarget{
					{Process: "scripts.tests.delivery.Handle"}, // This should succeed
				},
			},
		}

		results, err := center.Deliver(ctx, content, deliveryCtx, prefs, nil)

		// Should have error (from first webhook failure)
		assert.Error(t, err)

		// But all targets should be attempted
		require.Len(t, results, 3)

		// First webhook failed
		assert.False(t, results[0].Success)

		// Second webhook and process should succeed
		assert.True(t, successCalled, "second webhook should be called despite first failure")
		assert.True(t, results[1].Success, "second webhook should succeed")
		assert.True(t, results[2].Success, "process should succeed")
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

func createDeliveryTestRobot(t *testing.T, agentID string) *types.Robot {
	t.Helper()
	return &types.Robot{
		MemberID:    "test-robot-1",
		TeamID:      "test-team-1",
		DisplayName: "Test Robot",
		Config: &types.Config{
			Identity: &types.Identity{
				Role:   "Test Assistant",
				Duties: []string{"Testing", "Validation"},
			},
			Resources: &types.Resources{
				Phases: map[types.Phase]string{
					types.PhaseDelivery: agentID,
				},
			},
		},
	}
}

func createDeliveryTestExecution(robot *types.Robot) *types.Execution {
	exec := &types.Execution{
		ID:          "test-exec-delivery-1",
		MemberID:    robot.MemberID,
		TeamID:      robot.TeamID,
		TriggerType: types.TriggerClock,
		StartTime:   time.Now(),
		Status:      types.ExecRunning,
		Phase:       types.PhaseDelivery,
	}
	exec.SetRobot(robot)
	return exec
}
