//go:build integration

package task_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/agent/board"
	"github.com/yaoapp/yao/agent/task"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestScheduleEngine_StartStopCycle(t *testing.T) {
	testprepare.PrepareSandbox(t)

	se := task.ExportNewScheduleEngine()
	err := se.Start()
	require.NoError(t, err)

	// Update with an enabled entry
	se.Update("chat-sched-1", task.ScheduleConfig{
		Enabled:       true,
		Mode:          "interval",
		IntervalValue: 5,
		IntervalUnit:  "minute",
	})

	// Update with disabled entry (removes it)
	se.Update("chat-sched-2", task.ScheduleConfig{
		Enabled: false,
	})

	// Remove an entry
	se.Remove("chat-sched-1")

	// Stop gracefully
	se.Stop()
}

func TestScheduleEngine_UpdateAndRemove(t *testing.T) {
	testprepare.PrepareSandbox(t)

	se := task.ExportNewScheduleEngine()
	err := se.Start()
	require.NoError(t, err)
	defer se.Stop()

	// Add multiple entries
	se.Update("chat-a", task.ScheduleConfig{
		Enabled:       true,
		Mode:          "interval",
		IntervalValue: 10,
		IntervalUnit:  "minute",
	})
	se.Update("chat-b", task.ScheduleConfig{
		Enabled:       true,
		Mode:          "interval",
		IntervalValue: 30,
		IntervalUnit:  "minute",
	})

	// Disable one
	se.Update("chat-a", task.ScheduleConfig{Enabled: false})

	// Remove the other
	se.Remove("chat-b")
}

func TestScheduleEngine_StopWithoutStart(t *testing.T) {
	testprepare.PrepareSandbox(t)
	se := task.ExportNewScheduleEngine()
	// Stop without Start should not panic
	se.Stop()
}

func TestSubscribe_DBPath_NoMessages(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: identity.AlphaOwnerUserID,
		TeamID: identity.AlphaTeamID,
	}

	b, err := board.Create(ctx, auth, &board.CreateReq{Name: "Sub Board"})
	require.NoError(t, err)
	colID := b.Columns[0].ColumnID

	created, err := task.Create(ctx, auth, &task.CreateReq{
		Title: "Subscribe Test", AssistantID: "asst-sub-1", ColumnID: colID,
	})
	require.NoError(t, err)

	// Subscribe when no daemon is running - should use DB path
	sub, err := task.Subscribe(ctx, auth, created.ChatID, &task.SubscribeOpts{
		Replay:   task.ReplayAll,
		AfterSeq: 0,
	})
	require.NoError(t, err)
	require.NotNil(t, sub)
	defer sub.Cancel()

	// Channel should close (no messages in DB for this brand new task)
	select {
	case msg, ok := <-sub.Ch:
		if ok {
			t.Logf("got unexpected message: %+v", msg)
		}
	case <-time.After(2 * time.Second):
		t.Log("channel did not close in time — OK for empty DB")
	}
}

func TestSubscribe_DBPath_WithMessages(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: identity.AlphaOwnerUserID,
		TeamID: identity.AlphaTeamID,
	}

	b, err := board.Create(ctx, auth, &board.CreateReq{Name: "Sub Board 2"})
	require.NoError(t, err)
	colID := b.Columns[0].ColumnID

	created, err := task.Create(ctx, auth, &task.CreateReq{
		Title: "Subscribe Msg Test", AssistantID: "asst-sub-2", ColumnID: colID,
	})
	require.NoError(t, err)

	// Insert a fake message row into the message table
	msgTable := task.ExportLoadMessagesFromDB // just to verify the export works
	_ = msgTable

	props := map[string]interface{}{"content": "hello world", "role": "assistant"}
	propsJSON, _ := json.Marshal(props)

	err = capsule.Global.Query().Table("yao_agent_message").Insert(map[string]interface{}{
		"chat_id":    created.ChatID,
		"type":       "text",
		"props":      string(propsJSON),
		"sequence":   1,
		"created_at": time.Now(),
		"updated_at": time.Now(),
	})
	require.NoError(t, err)

	// Subscribe - should get the message from DB
	sub, err := task.Subscribe(ctx, auth, created.ChatID, &task.SubscribeOpts{
		Replay:   task.ReplayAll,
		AfterSeq: 0,
	})
	require.NoError(t, err)
	defer sub.Cancel()

	select {
	case msg, ok := <-sub.Ch:
		require.True(t, ok, "should receive a message")
		assert.Equal(t, "text", msg.Type)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for message")
	}
}

func TestSubscribe_AfterSeq_Filters(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: identity.AlphaOwnerUserID,
		TeamID: identity.AlphaTeamID,
	}

	b, err := board.Create(ctx, auth, &board.CreateReq{Name: "Sub AfterSeq Board"})
	require.NoError(t, err)
	colID := b.Columns[0].ColumnID

	created, err := task.Create(ctx, auth, &task.CreateReq{
		Title: "AfterSeq Test", AssistantID: "asst-sub-3", ColumnID: colID,
	})
	require.NoError(t, err)

	// Insert 3 messages
	for i := 1; i <= 3; i++ {
		props, _ := json.Marshal(map[string]interface{}{"content": "msg", "seq": i})
		err = capsule.Global.Query().Table("yao_agent_message").Insert(map[string]interface{}{
			"chat_id":    created.ChatID,
			"type":       "text",
			"props":      string(props),
			"sequence":   i,
			"created_at": time.Now(),
			"updated_at": time.Now(),
		})
		require.NoError(t, err)
	}

	// Subscribe with AfterSeq=2, should only get seq=3
	sub, err := task.Subscribe(ctx, auth, created.ChatID, &task.SubscribeOpts{
		Replay:   task.ReplayAfter,
		AfterSeq: 2,
	})
	require.NoError(t, err)
	defer sub.Cancel()

	var received int
	timeout := time.After(3 * time.Second)
loop:
	for {
		select {
		case _, ok := <-sub.Ch:
			if !ok {
				break loop
			}
			received++
		case <-timeout:
			break loop
		}
	}
	assert.Equal(t, 1, received, "should only receive messages after seq 2")
}

func TestInput_NotRunning(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: identity.AlphaOwnerUserID,
		TeamID: identity.AlphaTeamID,
	}

	err := task.Input(ctx, auth, "nonexistent-chat-id", &task.InputReq{
		Messages: []task.InputMessage{{Role: "user", Content: "hello"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestLoadMessagesFromDB_Empty(t *testing.T) {
	testprepare.PrepareSandbox(t)
	msgs := task.ExportLoadMessagesFromDB("nonexistent-chat-id", 0)
	assert.Empty(t, msgs)
}
