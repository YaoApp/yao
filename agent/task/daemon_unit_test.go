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

	// Watch now sends read_complete before live messages
	var received *message.Message
	timeout := time.After(time.Second)
	for {
		select {
		case msg := <-sub.Ch:
			if msg.Type == "event" && msg.Props["event"] == "read_complete" {
				continue // skip read_complete marker
			}
			received = msg
			goto check
		case <-timeout:
			t.Fatal("timeout waiting for broadcast")
			return
		}
	}
check:
	assert.Equal(t, "text", received.Type)
	assert.NotNil(t, received.Metadata)
	assert.Equal(t, 1, received.Metadata.Sequence)
}

func TestDaemonContext_CloseSubscribers(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-close")
	defer dc.Cancel()

	sub1, err := dc.Subscribe(&task.SubscribeOpts{Replay: task.ReplayNone})
	require.NoError(t, err)
	sub2, err := dc.Subscribe(&task.SubscribeOpts{Replay: task.ReplayNone})
	require.NoError(t, err)

	dc.CloseSubscribers()

	// Drain until channels close (may receive read_complete before closure)
	drainUntilClosed := func(ch <-chan *message.Message) bool {
		for {
			select {
			case _, ok := <-ch:
				if !ok {
					return true
				}
			case <-time.After(2 * time.Second):
				return false
			}
		}
	}
	assert.True(t, drainUntilClosed(sub1.Ch), "sub1.Ch should eventually close")
	assert.True(t, drainUntilClosed(sub2.Ch), "sub2.Ch should eventually close")
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

	// Receive 3 replayed text messages
	for i := 0; i < 3; i++ {
		select {
		case msg := <-sub.Ch:
			assert.Equal(t, "text", msg.Type)
		case <-time.After(time.Second):
			t.Fatalf("timeout waiting for replay message %d", i)
		}
	}

	// Receive read_complete marker
	select {
	case msg := <-sub.Ch:
		assert.Equal(t, "event", msg.Type)
		assert.Equal(t, "read_complete", msg.Props["event"])
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for read_complete")
	}

	// Receive live message
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
		textCount := 0
		for msg := range sub.Ch {
			if msg.Type == "text" {
				textCount++
				if textCount >= 10 {
					return
				}
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

	// Skip read_complete, then get text message
	for {
		select {
		case msg := <-sub.Ch:
			if msg.Type == "event" {
				continue
			}
			assert.Equal(t, "text", msg.Type)
			assert.Equal(t, "hello", msg.Props["content"])
			return
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for message from writer")
			return
		}
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

	for {
		select {
		case msg := <-sub.Ch:
			if msg.Type == "event" {
				continue
			}
			assert.Equal(t, "text", msg.Type)
			assert.Equal(t, "trace-123", msg.Metadata.TraceID)
			assert.Equal(t, 1, msg.Metadata.Sequence)
			return
		case <-time.After(time.Second):
			t.Fatal("timeout")
			return
		}
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

	textCount := 0
	for {
		select {
		case msg := <-subAll.Ch:
			if msg.Type == "text" {
				textCount++
			}
		case <-time.After(100 * time.Millisecond):
			goto done
		}
	}
done:
	assert.Equal(t, 70, textCount, "ringBuffer should contain all 70 text messages")
}
