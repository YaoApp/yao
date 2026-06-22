//go:build unit

package task_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/task"
)

func TestDaemonContext_Broadcast(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-broadcast")
	defer dc.Cancel()

	sub, err := dc.Subscribe(&task.SubscribeOpts{Replay: task.ReplayNone})
	require.NoError(t, err)
	defer sub.Cancel()

	dc.Broadcast(&message.Message{Type: "text", Props: map[string]interface{}{"content": "hello"}})

	select {
	case received := <-sub.Ch:
		assert.Equal(t, "text", received.Type)
		assert.NotNil(t, received.Metadata)
		assert.Equal(t, 1, received.Metadata.Sequence)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for broadcast")
	}
}

func TestDaemonContext_CloseSubscribers(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-close")
	defer dc.Cancel()

	sub1, err := dc.Subscribe(&task.SubscribeOpts{Replay: task.ReplayNone})
	require.NoError(t, err)
	sub2, err := dc.Subscribe(&task.SubscribeOpts{Replay: task.ReplayNone})
	require.NoError(t, err)

	dc.CloseSubscribers()

	_, ok1 := <-sub1.Ch
	_, ok2 := <-sub2.Ch
	assert.False(t, ok1, "sub1.Ch should be closed")
	assert.False(t, ok2, "sub2.Ch should be closed")
}

func TestDaemonContext_Subscribe_ReplayAll(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-subscribe-replay")
	defer dc.Cancel()

	for i := 0; i < 3; i++ {
		dc.Broadcast(&message.Message{Type: "text", Props: map[string]interface{}{"i": i}})
	}

	sub, err := dc.Subscribe(&task.SubscribeOpts{Replay: task.ReplayAll})
	require.NoError(t, err)
	defer sub.Cancel()

	for i := 0; i < 3; i++ {
		select {
		case msg := <-sub.Ch:
			assert.Equal(t, "text", msg.Type)
		case <-time.After(time.Second):
			t.Fatalf("timeout waiting for replay message %d", i)
		}
	}

	dc.Broadcast(&message.Message{Type: "text", Props: map[string]interface{}{"live": true}})
	select {
	case msg := <-sub.Ch:
		assert.Equal(t, true, msg.Props["live"])
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for live message")
	}
}

func TestDaemonContext_Subscribe_AfterClose(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-stopping")
	defer dc.Cancel()

	dc.CloseSubscribers()

	_, err := dc.Subscribe(&task.SubscribeOpts{Replay: task.ReplayAll})
	assert.Error(t, err)
}

func TestDaemonContext_ConcurrentBroadcast(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-concurrent")
	defer dc.Cancel()

	sub, err := dc.Subscribe(&task.SubscribeOpts{Replay: task.ReplayNone})
	require.NoError(t, err)
	defer sub.Cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		count := 0
		for range sub.Ch {
			count++
			if count >= 10 {
				return
			}
		}
	}()

	for i := 0; i < 10; i++ {
		dc.Broadcast(&message.Message{Type: "text", Props: map[string]interface{}{"n": i}})
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: not all messages received")
	}
}

func TestDaemonResponseWriter(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-writer")
	defer dc.Cancel()

	sub, err := dc.Subscribe(&task.SubscribeOpts{Replay: task.ReplayNone})
	require.NoError(t, err)
	defer sub.Cancel()

	w := task.ExportNewDaemonResponseWriter(dc)

	data := `data: {"type":"text","props":{"content":"hello"}}` + "\n\n"
	n, err := w.Write([]byte(data))
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)

	select {
	case msg := <-sub.Ch:
		assert.Equal(t, "text", msg.Type)
		assert.Equal(t, "hello", msg.Props["content"])
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message from writer")
	}
}

func TestDaemonContext_Broadcast_WithExistingMetadata(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-metadata")
	defer dc.Cancel()

	sub, err := dc.Subscribe(&task.SubscribeOpts{Replay: task.ReplayNone})
	require.NoError(t, err)
	defer sub.Cancel()

	dc.Broadcast(&message.Message{
		Type:     "text",
		Props:    map[string]interface{}{"content": "pre-meta"},
		Metadata: &message.Metadata{TraceID: "trace-123"},
	})

	select {
	case msg := <-sub.Ch:
		assert.Equal(t, "text", msg.Type)
		assert.Equal(t, "trace-123", msg.Metadata.TraceID)
		assert.Equal(t, 1, msg.Metadata.Sequence)
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestDaemonContext_Broadcast_NoSubscribers(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-no-sub")
	defer dc.Cancel()

	// Broadcast without any subscribers — should not panic
	dc.Broadcast(&message.Message{Type: "text", Props: map[string]interface{}{"content": "ignored"}})

	// Verify ringBuffer is still populated (via ReplayAll)
	sub, err := dc.Subscribe(&task.SubscribeOpts{Replay: task.ReplayAll})
	require.NoError(t, err)
	defer sub.Cancel()

	select {
	case msg := <-sub.Ch:
		assert.Equal(t, "ignored", msg.Props["content"])
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for replayed message")
	}
}

func TestDaemonContext_Broadcast_ChannelFull(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-full-chan")
	defer dc.Cancel()

	sub, err := dc.Subscribe(&task.SubscribeOpts{Replay: task.ReplayNone})
	require.NoError(t, err)
	defer sub.Cancel()

	// Fill the subscriber channel (capacity 64)
	for i := 0; i < 70; i++ {
		dc.Broadcast(&message.Message{Type: "text", Props: map[string]interface{}{"i": i}})
	}

	// Should not panic; some messages may be dropped but ringBuffer has them all
	subAll, err := dc.Subscribe(&task.SubscribeOpts{Replay: task.ReplayAll})
	require.NoError(t, err)
	defer subAll.Cancel()

	count := 0
	for {
		select {
		case <-subAll.Ch:
			count++
		case <-time.After(100 * time.Millisecond):
			goto done
		}
	}
done:
	assert.Equal(t, 70, count, "ringBuffer should contain all 70 messages")
}
