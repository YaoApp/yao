//go:build unit

package pool_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/robot/pool"
	"github.com/yaoapp/yao/agent/robot/types"
)

// ==================== Priority Queue Basic Tests ====================

func TestQueueNewPriorityQueue(t *testing.T) {
	t.Run("create with positive size", func(t *testing.T) {
		pq := pool.NewPriorityQueue(100)
		assert.NotNil(t, pq)
		assert.Equal(t, 0, pq.Size())
		assert.False(t, pq.IsFull())
	})

	t.Run("create with zero size (unlimited)", func(t *testing.T) {
		pq := pool.NewPriorityQueue(0)
		assert.NotNil(t, pq)
		assert.False(t, pq.IsFull())
	})
}

func TestQueueEnqueueDequeue(t *testing.T) {
	pq := pool.NewPriorityQueue(100)
	robot := createTestRobot("robot_1", "team_1", 5, 10, 5)
	ctx := createTestContext()

	t.Run("enqueue single item", func(t *testing.T) {
		item := &pool.QueueItem{
			Robot:   robot,
			Ctx:     ctx,
			Trigger: types.TriggerClock,
			Data:    "test_data",
		}
		ok := pq.Enqueue(item)
		assert.True(t, ok)
		assert.Equal(t, 1, pq.Size())
	})

	t.Run("dequeue single item", func(t *testing.T) {
		item := pq.Dequeue()
		assert.NotNil(t, item)
		assert.Equal(t, "robot_1", item.Robot.MemberID)
		assert.Equal(t, "test_data", item.Data)
		assert.Equal(t, 0, pq.Size())
	})

	t.Run("dequeue from empty queue", func(t *testing.T) {
		item := pq.Dequeue()
		assert.Nil(t, item)
	})
}

func TestQueueSize(t *testing.T) {
	pq := pool.NewPriorityQueue(100)
	robot := createTestRobot("robot_1", "team_1", 5, 10, 5)

	assert.Equal(t, 0, pq.Size())

	for i := 0; i < 5; i++ {
		pq.Enqueue(&pool.QueueItem{Robot: robot, Trigger: types.TriggerClock})
	}
	assert.Equal(t, 5, pq.Size())

	pq.Dequeue()
	pq.Dequeue()
	assert.Equal(t, 3, pq.Size())
}

// ==================== Global Queue Limit Tests ====================

func TestQueueGlobalLimit(t *testing.T) {
	pq := pool.NewPriorityQueue(5)

	for i := 0; i < 10; i++ {
		robot := createTestRobot("robot_"+string(rune('A'+i)), "team_1", 5, 10, 5)
		item := &pool.QueueItem{Robot: robot, Trigger: types.TriggerClock}
		ok := pq.Enqueue(item)

		if i < 5 {
			assert.True(t, ok, "Should accept item %d", i)
		} else {
			assert.False(t, ok, "Should reject item %d (queue full)", i)
		}
	}

	assert.Equal(t, 5, pq.Size())
	assert.True(t, pq.IsFull())
}

func TestQueueUnlimitedSize(t *testing.T) {
	pq := pool.NewPriorityQueue(0)

	for i := 0; i < 100; i++ {
		robot := createTestRobot("robot_"+string(rune('A'+i%26)), "team_1", 5, 1000, 5)
		item := &pool.QueueItem{Robot: robot, Trigger: types.TriggerClock}
		ok := pq.Enqueue(item)
		assert.True(t, ok)
	}

	assert.Equal(t, 100, pq.Size())
	assert.False(t, pq.IsFull())
}

// ==================== Per-Robot Queue Limit Tests ====================

func TestQueuePerRobotLimit(t *testing.T) {
	pq := pool.NewPriorityQueue(100)
	robot := createTestRobot("robot_limited", "team_1", 5, 3, 5)

	successCount := 0
	for i := 0; i < 10; i++ {
		item := &pool.QueueItem{Robot: robot, Trigger: types.TriggerClock}
		if pq.Enqueue(item) {
			successCount++
		}
	}

	assert.Equal(t, 3, successCount)
	assert.Equal(t, 3, pq.Size())
	assert.Equal(t, 3, pq.RobotQueuedCount("robot_limited"))
}

func TestQueueMultipleRobotsIndependentLimits(t *testing.T) {
	pq := pool.NewPriorityQueue(100)

	robotA := createTestRobot("robot_A", "team_1", 5, 2, 5)
	robotB := createTestRobot("robot_B", "team_1", 5, 3, 5)

	for i := 0; i < 5; i++ {
		pq.Enqueue(&pool.QueueItem{Robot: robotA, Trigger: types.TriggerClock})
	}
	assert.Equal(t, 2, pq.RobotQueuedCount("robot_A"))

	for i := 0; i < 5; i++ {
		pq.Enqueue(&pool.QueueItem{Robot: robotB, Trigger: types.TriggerClock})
	}
	assert.Equal(t, 3, pq.RobotQueuedCount("robot_B"))

	assert.Equal(t, 5, pq.Size())
}

func TestQueueRobotCountAfterDequeue(t *testing.T) {
	pq := pool.NewPriorityQueue(100)
	robot := createTestRobot("robot_1", "team_1", 5, 10, 5)

	for i := 0; i < 3; i++ {
		pq.Enqueue(&pool.QueueItem{Robot: robot, Trigger: types.TriggerClock})
	}
	assert.Equal(t, 3, pq.RobotQueuedCount("robot_1"))

	pq.Dequeue()
	assert.Equal(t, 2, pq.RobotQueuedCount("robot_1"))
	pq.Dequeue()
	assert.Equal(t, 1, pq.RobotQueuedCount("robot_1"))

	pq.Dequeue()
	assert.Equal(t, 0, pq.RobotQueuedCount("robot_1"))
}

func TestQueueNilRobot(t *testing.T) {
	pq := pool.NewPriorityQueue(100)

	item := &pool.QueueItem{
		Robot:   nil,
		Trigger: types.TriggerClock,
	}
	ok := pq.Enqueue(item)
	assert.True(t, ok)
	assert.Equal(t, 1, pq.Size())

	dequeued := pq.Dequeue()
	assert.NotNil(t, dequeued)
	assert.Nil(t, dequeued.Robot)
}

func TestQueueDefaultRobotQueueLimit(t *testing.T) {
	pq := pool.NewPriorityQueue(100)

	robot := &types.Robot{
		MemberID: "robot_no_config",
		TeamID:   "team_1",
	}

	successCount := 0
	for i := 0; i < 15; i++ {
		item := &pool.QueueItem{Robot: robot, Trigger: types.TriggerClock}
		if pq.Enqueue(item) {
			successCount++
		}
	}

	assert.Equal(t, 10, successCount)
}

// ==================== Priority Tests ====================

func TestQueuePriorityByRobotPriority(t *testing.T) {
	pq := pool.NewPriorityQueue(100)

	robotLow := createTestRobot("robot_low", "team_1", 5, 10, 1)
	robotMed := createTestRobot("robot_med", "team_1", 5, 10, 5)
	robotHigh := createTestRobot("robot_high", "team_1", 5, 10, 10)

	pq.Enqueue(&pool.QueueItem{Robot: robotLow, Trigger: types.TriggerClock})
	pq.Enqueue(&pool.QueueItem{Robot: robotMed, Trigger: types.TriggerClock})
	pq.Enqueue(&pool.QueueItem{Robot: robotHigh, Trigger: types.TriggerClock})

	item1 := pq.Dequeue()
	assert.Equal(t, "robot_high", item1.Robot.MemberID)

	item2 := pq.Dequeue()
	assert.Equal(t, "robot_med", item2.Robot.MemberID)

	item3 := pq.Dequeue()
	assert.Equal(t, "robot_low", item3.Robot.MemberID)
}

func TestQueuePriorityByTriggerType(t *testing.T) {
	pq := pool.NewPriorityQueue(100)

	robot := createTestRobot("robot_1", "team_1", 5, 10, 5)

	pq.Enqueue(&pool.QueueItem{Robot: robot, Trigger: types.TriggerClock})
	pq.Enqueue(&pool.QueueItem{Robot: robot, Trigger: types.TriggerEvent})
	pq.Enqueue(&pool.QueueItem{Robot: robot, Trigger: types.TriggerHuman})

	item1 := pq.Dequeue()
	assert.Equal(t, types.TriggerHuman, item1.Trigger)

	item2 := pq.Dequeue()
	assert.Equal(t, types.TriggerEvent, item2.Trigger)

	item3 := pq.Dequeue()
	assert.Equal(t, types.TriggerClock, item3.Trigger)
}

func TestQueuePriorityRobotOverTrigger(t *testing.T) {
	pq := pool.NewPriorityQueue(100)

	robotLow := createTestRobot("robot_low", "team_1", 5, 10, 1)
	robotHigh := createTestRobot("robot_high", "team_1", 5, 10, 10)

	pq.Enqueue(&pool.QueueItem{Robot: robotLow, Trigger: types.TriggerHuman})
	pq.Enqueue(&pool.QueueItem{Robot: robotHigh, Trigger: types.TriggerClock})

	item1 := pq.Dequeue()
	assert.Equal(t, "robot_high", item1.Robot.MemberID)

	item2 := pq.Dequeue()
	assert.Equal(t, "robot_low", item2.Robot.MemberID)
}

func TestQueuePriorityByEnqueueTime(t *testing.T) {
	pq := pool.NewPriorityQueue(100)

	robot := createTestRobot("robot_1", "team_1", 5, 10, 5)

	pq.Enqueue(&pool.QueueItem{Robot: robot, Trigger: types.TriggerClock, Data: "first"})
	time.Sleep(1 * time.Millisecond)
	pq.Enqueue(&pool.QueueItem{Robot: robot, Trigger: types.TriggerClock, Data: "second"})
	time.Sleep(1 * time.Millisecond)
	pq.Enqueue(&pool.QueueItem{Robot: robot, Trigger: types.TriggerClock, Data: "third"})

	item1 := pq.Dequeue()
	assert.Equal(t, "first", item1.Data)

	item2 := pq.Dequeue()
	assert.Equal(t, "second", item2.Data)

	item3 := pq.Dequeue()
	assert.Equal(t, "third", item3.Data)
}

// ==================== Concurrency Tests ====================

func TestQueueConcurrentEnqueue(t *testing.T) {
	pq := pool.NewPriorityQueue(1000)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			robot := createTestRobot("robot_"+string(rune('A'+id)), "team_1", 5, 100, 5)
			for j := 0; j < 50; j++ {
				pq.Enqueue(&pool.QueueItem{Robot: robot, Trigger: types.TriggerClock})
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	assert.Equal(t, 500, pq.Size())
}

func TestQueueConcurrentDequeue(t *testing.T) {
	pq := pool.NewPriorityQueue(1000)

	for i := 0; i < 500; i++ {
		robot := createTestRobot("robot_"+string(rune('A'+i%10)), "team_1", 5, 100, 5)
		pq.Enqueue(&pool.QueueItem{Robot: robot, Trigger: types.TriggerClock})
	}

	dequeued := make(chan *pool.QueueItem, 500)
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			for {
				item := pq.Dequeue()
				if item == nil {
					break
				}
				dequeued <- item
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
	close(dequeued)

	count := 0
	for range dequeued {
		count++
	}

	assert.Equal(t, 500, count)
	assert.Equal(t, 0, pq.Size())
}

func TestQueueConcurrentEnqueueDequeue(t *testing.T) {
	pq := pool.NewPriorityQueue(100)

	done := make(chan bool)

	go func() {
		for i := 0; i < 200; i++ {
			robot := createTestRobot("robot_"+string(rune('A'+i%10)), "team_1", 5, 50, 5)
			pq.Enqueue(&pool.QueueItem{Robot: robot, Trigger: types.TriggerClock})
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	dequeueCount := 0
	go func() {
		for i := 0; i < 200; i++ {
			if pq.Dequeue() != nil {
				dequeueCount++
			}
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	<-done
	<-done

	assert.GreaterOrEqual(t, dequeueCount, 1)
}

// ==================== Edge Cases ====================

func TestQueueIsFull(t *testing.T) {
	t.Run("not full initially", func(t *testing.T) {
		pq := pool.NewPriorityQueue(5)
		assert.False(t, pq.IsFull())
	})

	t.Run("full when at max", func(t *testing.T) {
		pq := pool.NewPriorityQueue(3)
		for i := 0; i < 3; i++ {
			robot := createTestRobot("robot_"+string(rune('A'+i)), "team_1", 5, 10, 5)
			pq.Enqueue(&pool.QueueItem{Robot: robot, Trigger: types.TriggerClock})
		}
		assert.True(t, pq.IsFull())
	})

	t.Run("not full after dequeue", func(t *testing.T) {
		pq := pool.NewPriorityQueue(3)
		for i := 0; i < 3; i++ {
			robot := createTestRobot("robot_"+string(rune('A'+i)), "team_1", 5, 10, 5)
			pq.Enqueue(&pool.QueueItem{Robot: robot, Trigger: types.TriggerClock})
		}
		pq.Dequeue()
		assert.False(t, pq.IsFull())
	})

	t.Run("never full when unlimited", func(t *testing.T) {
		pq := pool.NewPriorityQueue(0)
		for i := 0; i < 100; i++ {
			robot := createTestRobot("robot_"+string(rune('A'+i%26)), "team_1", 5, 1000, 5)
			pq.Enqueue(&pool.QueueItem{Robot: robot, Trigger: types.TriggerClock})
		}
		assert.False(t, pq.IsFull())
	})
}

func TestQueueRobotQueuedCount(t *testing.T) {
	pq := pool.NewPriorityQueue(100)

	t.Run("zero for unknown robot", func(t *testing.T) {
		assert.Equal(t, 0, pq.RobotQueuedCount("unknown_robot"))
	})

	t.Run("correct count for robot", func(t *testing.T) {
		robot := createTestRobot("robot_1", "team_1", 5, 10, 5)
		pq.Enqueue(&pool.QueueItem{Robot: robot, Trigger: types.TriggerClock})
		pq.Enqueue(&pool.QueueItem{Robot: robot, Trigger: types.TriggerClock})
		assert.Equal(t, 2, pq.RobotQueuedCount("robot_1"))
	})

	t.Run("zero after all dequeued", func(t *testing.T) {
		robot := createTestRobot("robot_2", "team_1", 5, 10, 5)
		pq.Enqueue(&pool.QueueItem{Robot: robot, Trigger: types.TriggerClock})
		pq.Dequeue()
		pq.Dequeue()
		pq.Dequeue()
		assert.Equal(t, 0, pq.RobotQueuedCount("robot_2"))
	})
}

func TestQueueEnqueueSetsEnqueueTime(t *testing.T) {
	pq := pool.NewPriorityQueue(100)
	robot := createTestRobot("robot_1", "team_1", 5, 10, 5)

	before := time.Now()
	pq.Enqueue(&pool.QueueItem{Robot: robot, Trigger: types.TriggerClock})
	after := time.Now()

	item := pq.Dequeue()
	assert.True(t, item.EnqueueTime.After(before) || item.EnqueueTime.Equal(before))
	assert.True(t, item.EnqueueTime.Before(after) || item.EnqueueTime.Equal(after))
}
