//go:build integration

package task_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/event"
	eventtypes "github.com/yaoapp/yao/event/types"

	_ "github.com/yaoapp/yao/agent/task" // ensure init() registers handlers
)

func TestEventPush_TaskPrefix_NoErrNoHandler(t *testing.T) {
	event.Reset()
	require.NoError(t, event.Start())
	defer event.Stop(context.Background())

	ch := make(chan *eventtypes.Event, 10)
	subID := event.Subscribe("task.*", ch)
	defer event.Unsubscribe(subID)

	payload := map[string]any{"chat_id": "c1", "__yao_team_id": "t1"}
	_, err := event.Push(context.Background(), "task.created", payload)
	require.NoError(t, err, "event.Push should not return ErrNoHandler")

	select {
	case ev := <-ch:
		assert.Equal(t, "task.created", ev.Type)
		m := ev.Payload.(map[string]any)
		assert.Equal(t, "c1", m["chat_id"])
		assert.Equal(t, "t1", m["__yao_team_id"])
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for task.created event")
	}
}

func TestEventPush_BoardPrefix_NoErrNoHandler(t *testing.T) {
	event.Reset()
	require.NoError(t, event.Start())
	defer event.Stop(context.Background())

	ch := make(chan *eventtypes.Event, 10)
	subID := event.Subscribe("board.*", ch)
	defer event.Unsubscribe(subID)

	payload := map[string]any{"board_id": "b1", "__yao_team_id": "t1"}
	_, err := event.Push(context.Background(), "board.deleted", payload)
	require.NoError(t, err, "event.Push(board.*) should not return ErrNoHandler")

	select {
	case ev := <-ch:
		assert.Equal(t, "board.deleted", ev.Type)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for board.deleted event")
	}
}

func TestEventPush_MailPrefix_NoErrNoHandler(t *testing.T) {
	event.Reset()
	require.NoError(t, event.Start())
	defer event.Stop(context.Background())

	ch := make(chan *eventtypes.Event, 10)
	subID := event.Subscribe("mail.*", ch)
	defer event.Unsubscribe(subID)

	payload := map[string]any{"mail_id": "m1", "__yao_created_by": "u1"}
	_, err := event.Push(context.Background(), "mail.new", payload)
	require.NoError(t, err, "event.Push(mail.*) should not return ErrNoHandler")

	select {
	case ev := <-ch:
		assert.Equal(t, "mail.new", ev.Type)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for mail.new event")
	}
}

func TestEventPush_PayloadContainsTeamID(t *testing.T) {
	event.Reset()
	require.NoError(t, event.Start())
	defer event.Stop(context.Background())

	ch := make(chan *eventtypes.Event, 10)
	subID := event.Subscribe("task.*", ch)
	defer event.Unsubscribe(subID)

	payload := map[string]any{
		"chat_id":       "c1",
		"__yao_team_id": "team-abc",
		"title":         "Test Task",
	}
	_, err := event.Push(context.Background(), "task.created", payload)
	require.NoError(t, err)

	select {
	case ev := <-ch:
		m := ev.Payload.(map[string]any)
		assert.Equal(t, "team-abc", m["__yao_team_id"], "__yao_team_id must be present in payload")
	case <-time.After(3 * time.Second):
		t.Fatal("timed out")
	}
}
