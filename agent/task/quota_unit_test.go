//go:build unit

package task_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/task"
)

func TestQuotaManager_TryAcquire(t *testing.T) {
	qm := task.NewTestQuotaManager(2)

	assert.True(t, qm.TryAcquire("team1", ""))
	assert.True(t, qm.TryAcquire("team1", ""))
	assert.False(t, qm.TryAcquire("team1", ""))
}

func TestQuotaManager_Release_SignalsNext(t *testing.T) {
	qm := task.NewTestQuotaManager(1)

	assert.True(t, qm.TryAcquire("team1", ""))
	assert.False(t, qm.TryAcquire("team1", ""))

	entry := task.ExportEnqueue(qm, "team1", "chat-queued", 500)
	qm.Release("team1")

	select {
	case <-entry.Ready():
	default:
		t.Fatal("expected entry.Ready to be closed after Release")
	}
}

func TestQuotaManager_Dequeue(t *testing.T) {
	qm := task.NewTestQuotaManager(1)

	assert.True(t, qm.TryAcquire("team1", ""))
	task.ExportEnqueue(qm, "team1", "chat-a", 500)
	task.ExportEnqueue(qm, "team1", "chat-b", 600)

	qm.Dequeue("team1", "chat-a")

	qm.Release("team1")
	pos := qm.QueuePosition("team1", "chat-b")
	assert.Equal(t, 0, pos, "chat-b should have been dequeued by Release")
}

func TestQuotaManager_PriorityOrder(t *testing.T) {
	qm := task.NewTestQuotaManager(1)
	assert.True(t, qm.TryAcquire("team1", ""))

	task.ExportEnqueue(qm, "team1", "low-priority", 100)
	task.ExportEnqueue(qm, "team1", "high-priority", 900)

	// Release should signal high-priority first
	qm.Release("team1")

	// low-priority should still be in queue
	pos := qm.QueuePosition("team1", "low-priority")
	assert.Equal(t, 1, pos, "low-priority should still be queued")
}

func TestQuotaManager_DefaultLimit(t *testing.T) {
	qm := task.NewTestQuotaManager(3)

	assert.True(t, qm.TryAcquire("new-team", ""))
	assert.True(t, qm.TryAcquire("new-team", ""))
	assert.True(t, qm.TryAcquire("new-team", ""))
	assert.False(t, qm.TryAcquire("new-team", ""))
}

func TestQuotaManager_SamePriority_FIFO(t *testing.T) {
	qm := task.NewTestQuotaManager(1)
	assert.True(t, qm.TryAcquire("team1", ""))

	// Same priority — FIFO order
	task.ExportEnqueue(qm, "team1", "first", 500)
	task.ExportEnqueue(qm, "team1", "second", 500)

	// Release: "first" should get signaled (lower index = earlier insertion)
	qm.Release("team1")

	pos := qm.QueuePosition("team1", "second")
	assert.Equal(t, 1, pos, "second should still be queued")
}

func TestQuotaManager_Release_WhenEmpty(t *testing.T) {
	qm := task.NewTestQuotaManager(2)
	assert.True(t, qm.TryAcquire("team1", ""))

	// Release without any queue entries — should not panic
	qm.Release("team1")

	// Running count should be 0 now
	assert.True(t, qm.TryAcquire("team1", ""))
	assert.True(t, qm.TryAcquire("team1", ""))
	assert.False(t, qm.TryAcquire("team1", ""))
}

func TestQuotaManager_Dequeue_NotFound(t *testing.T) {
	qm := task.NewTestQuotaManager(1)
	// Dequeue from empty team — should not panic
	qm.Dequeue("team1", "nonexistent")
}

func TestQuotaManager_QueuePosition_NotFound(t *testing.T) {
	qm := task.NewTestQuotaManager(1)
	assert.Equal(t, 0, qm.QueuePosition("team1", "nonexistent"))
}

func TestQuotaManager_FallbackLimit(t *testing.T) {
	// Create QM without team1/new-team limit — uses defaultQuotaLimit (9)
	qm := task.NewTestQuotaManagerNoLimits()

	for i := 0; i < 9; i++ {
		assert.True(t, qm.TryAcquire("unknown-team", ""), "acquire %d should succeed", i+1)
	}
	assert.False(t, qm.TryAcquire("unknown-team", ""), "acquire 10 should fail at limit")
}
