//go:build unit

package task_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/task"
)

func TestWatch_DaemonAlive_ReplayAll(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-watch-replay-all")
	defer dc.Cancel()
	task.ExportRegisterDaemon("test-watch-replay-all", dc)
	defer task.ExportUnregisterDaemon("test-watch-replay-all")

	for i := 0; i < 5; i++ {
		dc.Broadcast(&message.Message{Type: "text", Props: map[string]interface{}{"i": i}})
	}

	stream, err := dc.Watch(&task.WatchOpts{AfterSeq: 0, Limit: 0})
	require.NoError(t, err)
	defer stream.Cancel()

	// Should receive all 5 messages
	for i := 0; i < 5; i++ {
		select {
		case msg := <-stream.Ch:
			assert.Equal(t, "text", msg.Type)
		case <-time.After(time.Second):
			t.Fatalf("timeout waiting for replay msg %d", i)
		}
	}

	// Should receive read_complete marker
	select {
	case msg := <-stream.Ch:
		assert.Equal(t, "event", msg.Type)
		assert.Equal(t, "read_complete", msg.Props["event"])
		assert.Equal(t, false, msg.Props["has_more"])
		assert.Equal(t, true, msg.Props["live"])
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for read_complete")
	}

	// Should receive live messages
	dc.Broadcast(&message.Message{Type: "text", Props: map[string]interface{}{"live": true}})
	select {
	case msg := <-stream.Ch:
		assert.Equal(t, true, msg.Props["live"])
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for live message")
	}
}

func TestWatch_DaemonAlive_Limit(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-watch-limit")
	defer dc.Cancel()
	task.ExportRegisterDaemon("test-watch-limit", dc)
	defer task.ExportUnregisterDaemon("test-watch-limit")

	for i := 0; i < 10; i++ {
		dc.Broadcast(&message.Message{Type: "text", Props: map[string]interface{}{"i": i}})
	}

	stream, err := dc.Watch(&task.WatchOpts{AfterSeq: 0, Limit: 3})
	require.NoError(t, err)
	defer stream.Cancel()

	// Should receive only 3 messages (limited)
	for i := 0; i < 3; i++ {
		select {
		case msg := <-stream.Ch:
			assert.Equal(t, "text", msg.Type)
		case <-time.After(time.Second):
			t.Fatalf("timeout waiting for limited msg %d", i)
		}
	}

	// Should receive read_complete with has_more=true, live=false
	select {
	case msg := <-stream.Ch:
		assert.Equal(t, "event", msg.Type)
		assert.Equal(t, "read_complete", msg.Props["event"])
		assert.Equal(t, true, msg.Props["has_more"])
		assert.Equal(t, false, msg.Props["live"])
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for read_complete")
	}

	// Channel should close (no live mode when truncated)
	select {
	case _, ok := <-stream.Ch:
		assert.False(t, ok, "channel should be closed after limit truncation")
	case <-time.After(time.Second):
		t.Fatal("timeout: channel should have closed")
	}
}

func TestWatch_DaemonAlive_AfterSeq(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-watch-afterseq")
	defer dc.Cancel()
	task.ExportRegisterDaemon("test-watch-afterseq", dc)
	defer task.ExportUnregisterDaemon("test-watch-afterseq")

	for i := 0; i < 5; i++ {
		dc.Broadcast(&message.Message{Type: "text", Props: map[string]interface{}{"i": i}})
	}

	// Request from after seq 3 (should get seq 4, 5)
	stream, err := dc.Watch(&task.WatchOpts{AfterSeq: 3, Limit: 0})
	require.NoError(t, err)
	defer stream.Cancel()

	// Should receive 2 messages (seq 4 and 5)
	for i := 0; i < 2; i++ {
		select {
		case msg := <-stream.Ch:
			assert.Equal(t, "text", msg.Type)
			assert.True(t, msg.Metadata.Sequence > 3)
		case <-time.After(time.Second):
			t.Fatalf("timeout waiting for afterseq msg %d", i)
		}
	}

	// read_complete marker
	select {
	case msg := <-stream.Ch:
		assert.Equal(t, "read_complete", msg.Props["event"])
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for read_complete")
	}
}

func TestWatch_DaemonStopping(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-watch-stopping")
	defer dc.Cancel()

	dc.CloseSubscribers()

	_, err := dc.Watch(&task.WatchOpts{})
	assert.Error(t, err)
}

func TestWSCommand_Parse(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected task.WSCommand
	}{
		{
			name: "read with since and limit",
			json: `{"type":"read","since":50,"limit":100}`,
			expected: task.WSCommand{
				Type:  "read",
				Since: 50,
				Limit: 100,
			},
		},
		{
			name: "run with messages",
			json: `{"type":"run","messages":[{"role":"user","content":"hello"}],"assistant_id":"asst-1"}`,
			expected: task.WSCommand{
				Type:        "run",
				Messages:    []task.InputMessage{{Role: "user", Content: "hello"}},
				AssistantID: "asst-1",
			},
		},
		{
			name: "retry with extra guidance",
			json: `{"type":"retry","messages":[{"role":"user","content":"try again carefully"}]}`,
			expected: task.WSCommand{
				Type:     "retry",
				Messages: []task.InputMessage{{Role: "user", Content: "try again carefully"}},
			},
		},
		{
			name:     "repeat",
			json:     `{"type":"repeat"}`,
			expected: task.WSCommand{Type: "repeat"},
		},
		{
			name:     "stop",
			json:     `{"type":"stop"}`,
			expected: task.WSCommand{Type: "stop"},
		},
		{
			name:     "cancel",
			json:     `{"type":"cancel"}`,
			expected: task.WSCommand{Type: "cancel"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cmd task.WSCommand
			err := json.Unmarshal([]byte(tt.json), &cmd)
			require.NoError(t, err)
			assert.Equal(t, tt.expected.Type, cmd.Type)
			assert.Equal(t, tt.expected.Since, cmd.Since)
			assert.Equal(t, tt.expected.Limit, cmd.Limit)
			if tt.expected.AssistantID != "" {
				assert.Equal(t, tt.expected.AssistantID, cmd.AssistantID)
			}
			if tt.expected.Messages != nil {
				assert.Equal(t, len(tt.expected.Messages), len(cmd.Messages))
			}
		})
	}
}

func TestRunDaemon_SingleRound_Exit(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-single-round")
	defer dc.Cancel()
	task.ExportRegisterDaemon("test-single-round", dc)

	stream, err := dc.Watch(&task.WatchOpts{})
	require.NoError(t, err)
	defer stream.Cancel()

	// Broadcast some messages then close (simulating single-round daemon exit)
	dc.Broadcast(&message.Message{Type: "text", Props: map[string]interface{}{"content": "working..."}})
	dc.Broadcast(&message.Message{Type: "done", Props: map[string]interface{}{}})
	dc.CloseSubscribers()

	// Drain all messages
	var received []*message.Message
	timeout := time.After(2 * time.Second)
	for {
		select {
		case msg, ok := <-stream.Ch:
			if !ok {
				goto done
			}
			received = append(received, msg)
		case <-timeout:
			goto done
		}
	}
done:
	// Should have: read_complete + text + done (at minimum)
	assert.True(t, len(received) >= 2, "expected at least read_complete + messages, got %d", len(received))

	// Verify daemon is unregisterable (test simulates manual unregister since real runDaemon would do it)
	task.ExportUnregisterDaemon("test-single-round")
	_, exists := task.ExportGetDaemon("test-single-round")
	assert.False(t, exists)
}
