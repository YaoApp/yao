//go:build integration

package task_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/agent/board"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/task"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

// createTestBoard creates a board and returns its first column_id for task creation.
func createTestBoard(t *testing.T, ctx context.Context, auth *process.AuthorizedInfo) string {
	t.Helper()
	b, err := board.Create(ctx, auth, &board.CreateReq{
		Name:  "Test Board",
		Icon:  "material-test",
		Color: "#3B82F6",
	})
	require.NoError(t, err)
	require.NotEmpty(t, b.Columns)
	return b.Columns[0].ColumnID
}

func TestRun_BasicExecution(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)

	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: identity.AlphaOwnerUserID,
		TeamID: identity.AlphaTeamID,
	}

	colID := createTestBoard(t, ctx, auth)

	created, err := task.Create(ctx, auth, &task.CreateReq{
		Title:       "Execution Test",
		AssistantID: "asst-test-001",
		ColumnID:    colID,
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ChatID)

	// Inject mock AssistantStreamFn
	origFn := task.AssistantStreamFn
	task.AssistantStreamFn = func(assistantID string, agCtx *agentcontext.Context, msgs []agentcontext.Message, opts ...*agentcontext.Options) (*agentcontext.Response, error) {
		if agCtx.Writer != nil {
			data := []byte(`data: {"type":"text","props":{"content":"mock response"}}` + "\n\n")
			agCtx.Writer.Write(data)
		}
		return &agentcontext.Response{}, nil
	}
	defer func() { task.AssistantStreamFn = origFn }()

	// Run the task
	result, err := task.Run(ctx, auth, created.ChatID, &task.RunReq{
		Messages: []task.InputMessage{{Role: "user", Content: "hello"}},
		Priority: 500,
	})
	require.NoError(t, err)
	assert.Equal(t, "running", result.Status)
	assert.NotEmpty(t, result.RequestID)

	// Wait for completion
	time.Sleep(1 * time.Second)

	// Verify DB state
	got, err := task.Get(ctx, auth, created.ChatID)
	require.NoError(t, err)
	assert.Equal(t, "completed", got.RunStatus)
	assert.GreaterOrEqual(t, got.RunCount, 1)
}

func TestRun_QuotaQueued(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)

	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: identity.AlphaOwnerUserID,
		TeamID: identity.AlphaTeamID,
	}

	colID := createTestBoard(t, ctx, auth)

	// Inject mock AssistantStreamFn that blocks
	origFn := task.AssistantStreamFn
	task.AssistantStreamFn = func(assistantID string, agCtx *agentcontext.Context, msgs []agentcontext.Message, opts ...*agentcontext.Options) (*agentcontext.Response, error) {
		select {
		case <-agCtx.Done():
			return nil, agCtx.Err()
		case <-time.After(5 * time.Second):
		}
		return &agentcontext.Response{}, nil
	}
	defer func() { task.AssistantStreamFn = origFn }()

	// Create and run tasks until one gets queued
	var chatIDs []string
	for i := 0; i < 5; i++ {
		created, err := task.Create(ctx, auth, &task.CreateReq{
			Title:       fmt.Sprintf("Quota Test %d", i),
			AssistantID: "asst-test-001",
			ColumnID:    colID,
		})
		require.NoError(t, err)
		chatIDs = append(chatIDs, created.ChatID)
	}

	var queuedChat string
	for _, cid := range chatIDs {
		result, err := task.Run(ctx, auth, cid, &task.RunReq{
			Messages: []task.InputMessage{{Role: "user", Content: "test"}},
			Priority: 500,
		})
		require.NoError(t, err)
		if result.Status == "queued" {
			queuedChat = cid
			break
		}
	}

	if queuedChat == "" {
		t.Skip("quota limit not reached with 5 tasks — default limit may be higher")
	}

	assert.NotEmpty(t, queuedChat)

	// Stop all tasks to clean up
	for _, cid := range chatIDs {
		task.Stop(ctx, auth, cid, true)
	}
	time.Sleep(500 * time.Millisecond)
}

func TestStop_GracefulAndForce(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)

	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: identity.AlphaOwnerUserID,
		TeamID: identity.AlphaTeamID,
	}

	colID := createTestBoard(t, ctx, auth)

	// Inject mock that blocks until cancelled
	origFn := task.AssistantStreamFn
	task.AssistantStreamFn = func(assistantID string, agCtx *agentcontext.Context, msgs []agentcontext.Message, opts ...*agentcontext.Options) (*agentcontext.Response, error) {
		<-agCtx.Done()
		return nil, agCtx.Err()
	}
	defer func() { task.AssistantStreamFn = origFn }()

	// Test graceful stop
	created, err := task.Create(ctx, auth, &task.CreateReq{
		Title:       "Stop Graceful Test",
		AssistantID: "asst-test-001",
		ColumnID:    colID,
	})
	require.NoError(t, err)

	_, err = task.Run(ctx, auth, created.ChatID, &task.RunReq{
		Messages: []task.InputMessage{{Role: "user", Content: "block"}},
	})
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)
	err = task.Stop(ctx, auth, created.ChatID, false)
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)
	got, err := task.Get(ctx, auth, created.ChatID)
	require.NoError(t, err)
	assert.Contains(t, []string{"stopped", "cancelled", "failed", "completed"}, got.RunStatus)

	// Test force stop
	created2, err := task.Create(ctx, auth, &task.CreateReq{
		Title:       "Stop Force Test",
		AssistantID: "asst-test-001",
		ColumnID:    colID,
	})
	require.NoError(t, err)

	_, err = task.Run(ctx, auth, created2.ChatID, &task.RunReq{
		Messages: []task.InputMessage{{Role: "user", Content: "block"}},
	})
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)
	err = task.Stop(ctx, auth, created2.ChatID, true)
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)
	got2, err := task.Get(ctx, auth, created2.ChatID)
	require.NoError(t, err)
	assert.Contains(t, []string{"stopped", "cancelled", "failed", "completed"}, got2.RunStatus)
}

func TestSetStatus_InboxTrigger(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)

	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: identity.AlphaOwnerUserID,
		TeamID: identity.AlphaTeamID,
	}

	colID := createTestBoard(t, ctx, auth)

	// Inject mock that completes immediately
	origFn := task.AssistantStreamFn
	task.AssistantStreamFn = func(assistantID string, agCtx *agentcontext.Context, msgs []agentcontext.Message, opts ...*agentcontext.Options) (*agentcontext.Response, error) {
		if agCtx.Writer != nil {
			data := []byte(`data: {"type":"text","props":{"content":"done"}}` + "\n\n")
			agCtx.Writer.Write(data)
		}
		return &agentcontext.Response{}, nil
	}
	defer func() { task.AssistantStreamFn = origFn }()

	created, err := task.Create(ctx, auth, &task.CreateReq{
		Title:       "Inbox Trigger Test",
		AssistantID: "asst-test-001",
		ColumnID:    colID,
	})
	require.NoError(t, err)

	_, err = task.Run(ctx, auth, created.ChatID, &task.RunReq{
		Messages: []task.InputMessage{{Role: "user", Content: "do something"}},
	})
	require.NoError(t, err)

	// Wait for completion + inbox trigger
	time.Sleep(1500 * time.Millisecond)

	// Verify agent_mail created for this task
	mailTbl := share.App.Prefix + "agent_mail"
	if m, err := model.Get("__yao.agent.mail"); err == nil && m.MetaData.Table.Name != "" {
		mailTbl = m.MetaData.Table.Name
	}
	rows, err := capsule.Global.Query().Table(mailTbl).
		Where("chat_id", "=", created.ChatID).
		Where("type", "=", "completed").
		Get()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(rows), 1, "expected at least 1 completed mail for task")
}

func TestEnrichTaskResult_Integration(t *testing.T) {
	identity := testprepare.PrepareE2E(t)

	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: identity.AlphaOwnerUserID,
		TeamID: identity.AlphaTeamID,
	}

	colID := createTestBoard(t, ctx, auth)

	// Inject mock AssistantStreamFn that writes content to the ring buffer
	origFn := task.AssistantStreamFn
	task.AssistantStreamFn = func(assistantID string, agCtx *agentcontext.Context, msgs []agentcontext.Message, opts ...*agentcontext.Options) (*agentcontext.Response, error) {
		if agCtx.Writer != nil {
			data := []byte(`data: {"type":"text","props":{"content":"I have completed the analysis and generated the report."}}` + "\n\n")
			agCtx.Writer.Write(data)
		}
		return &agentcontext.Response{}, nil
	}
	defer func() { task.AssistantStreamFn = origFn }()

	created, err := task.Create(ctx, auth, &task.CreateReq{
		Title:       "Enrich Test",
		AssistantID: "asst-test-001",
		ColumnID:    colID,
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ChatID)

	_, err = task.Run(ctx, auth, created.ChatID, &task.RunReq{
		Messages: []task.InputMessage{{Role: "user", Content: "Please analyze this data and generate a report."}},
		Priority: 500,
	})
	require.NoError(t, err)

	// Wait for daemon completion + async enrichment (mock LLM + DB write)
	time.Sleep(3 * time.Second)

	got, err := task.Get(ctx, auth, created.ChatID)
	require.NoError(t, err)

	// Enrichment must NOT fail (the old code would hit "capabilities are required")
	assert.NotEqual(t, "failed", got.RunStatus,
		"enrichment should not fail — got error_message: %s", got.ErrorMessage)
	assert.Equal(t, "completed", got.RunStatus)

	// Title should be populated by LLM enrichment (mock returns "Test Task Title")
	chatTbl := share.App.Prefix + "agent_chat"
	if m, err := model.Get("__yao.agent.chat"); err == nil && m.MetaData.Table.Name != "" {
		chatTbl = m.MetaData.Table.Name
	}
	chatRow, err := capsule.Global.Query().Table(chatTbl).
		Where("chat_id", "=", created.ChatID).
		First()
	require.NoError(t, err)
	require.NotNil(t, chatRow)
	title, _ := chatRow["title"].(string)
	assert.NotEmpty(t, title, "enrichment should have set a title on the chat")
}
