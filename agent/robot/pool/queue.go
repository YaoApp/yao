package pool

import (
	"container/heap"
	"sync"
	"time"

	"github.com/yaoapp/yao/agent/robot/types"
)

// QueueItem represents a job waiting in the queue
type QueueItem struct {
	Robot        *types.Robot
	Ctx          *types.Context
	Trigger      types.TriggerType
	Data         interface{}
	ExecutorMode types.ExecutorMode     // optional: override robot's executor mode
	ExecID       string                 // pre-generated execution ID for tracking
	Control      types.ExecutionControl // execution control for pause/resume/stop
	EnqueueTime  time.Time
	Priority     int // calculated priority for sorting
	Index        int // index in heap (managed by container/heap)
}

// PriorityQueue implements a priority queue for robot executions
// Sorted by: robot priority > trigger type priority > wait time
type PriorityQueue struct {
	items      []*QueueItem
	mu         sync.RWMutex
	maxSize    int            // global queue size limit
	robotCount map[string]int // per-robot queue count: memberID -> count
}

// NewPriorityQueue creates a new priority queue
func NewPriorityQueue(maxSize int) *PriorityQueue {
	pq := &PriorityQueue{
		items:      make([]*QueueItem, 0),
		maxSize:    maxSize,
		robotCount: make(map[string]int),
	}
	heap.Init(pq)
	return pq
}

// Enqueue adds an item to the queue
// Returns false if:
// - Global queue is full (maxSize)
// - Robot's queue limit reached (Quota.Queue)
func (pq *PriorityQueue) Enqueue(item *QueueItem) bool {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	// Check 1: Global queue limit
	if pq.maxSize > 0 && len(pq.items) >= pq.maxSize {
		return false // global queue full
	}

	// Check 2: Per-robot queue limit (prevents single robot from hogging the queue)
	if item.Robot != nil {
		memberID := item.Robot.MemberID
		robotQueueLimit := 10 // default
		if item.Robot.Config != nil && item.Robot.Config.Quota != nil {
			robotQueueLimit = item.Robot.Config.Quota.GetQueue()
		}

		if pq.robotCount[memberID] >= robotQueueLimit {
			return false // robot's queue limit reached
		}

		// Increment robot's queue count
		pq.robotCount[memberID]++
	}

	item.Priority = calculatePriority(item)
	item.EnqueueTime = time.Now()
	heap.Push(pq, item)
	return true
}

// Dequeue removes and returns the highest priority item
// Returns nil if queue is empty
func (pq *PriorityQueue) Dequeue() *QueueItem {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if len(pq.items) == 0 {
		return nil
	}

	item := heap.Pop(pq).(*QueueItem)

	// Decrement robot's queue count
	if item.Robot != nil {
		memberID := item.Robot.MemberID
		if pq.robotCount[memberID] > 0 {
			pq.robotCount[memberID]--
		}
		// Clean up if count reaches zero
		if pq.robotCount[memberID] == 0 {
			delete(pq.robotCount, memberID)
		}
	}

	return item
}

// Size returns the number of items in the queue (thread-safe)
func (pq *PriorityQueue) Size() int {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	return len(pq.items)
}

// IsFull returns true if queue has reached max capacity
func (pq *PriorityQueue) IsFull() bool {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	return pq.maxSize > 0 && len(pq.items) >= pq.maxSize
}

// RobotQueuedCount returns the number of queued items for a specific robot
func (pq *PriorityQueue) RobotQueuedCount(memberID string) int {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	return pq.robotCount[memberID]
}

// ==================== heap.Interface implementation ====================
// These methods are called internally by heap.Push/Pop with lock already held

func (pq *PriorityQueue) Len() int { return len(pq.items) }

func (pq *PriorityQueue) Less(i, j int) bool {
	// Higher priority value = higher priority (processed first)
	// If priority is equal, older items (earlier EnqueueTime) come first
	if pq.items[i].Priority == pq.items[j].Priority {
		return pq.items[i].EnqueueTime.Before(pq.items[j].EnqueueTime)
	}
	return pq.items[i].Priority > pq.items[j].Priority
}

func (pq *PriorityQueue) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
	pq.items[i].Index = i
	pq.items[j].Index = j
}

// Push is required by heap.Interface
// Note: This is called by heap.Push(), not directly
func (pq *PriorityQueue) Push(x interface{}) {
	item := x.(*QueueItem)
	item.Index = len(pq.items)
	pq.items = append(pq.items, item)
}

// Pop is required by heap.Interface
// Note: This is called by heap.Pop(), not directly
func (pq *PriorityQueue) Pop() interface{} {
	old := pq.items
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.Index = -1 // mark as removed
	pq.items = old[0 : n-1]
	return item
}

// ==================== Priority Calculation ====================

// calculatePriority calculates the priority score for a queue item
// Priority = robot_priority * 1000 + trigger_priority * 100
// Higher score = higher priority
func calculatePriority(item *QueueItem) int {
	priority := 0

	// 1. Robot priority (from config, 1-10, default 5)
	if item.Robot != nil && item.Robot.Config != nil && item.Robot.Config.Quota != nil {
		robotPriority := item.Robot.Config.Quota.GetPriority()
		priority += robotPriority * 1000
	} else {
		priority += 5000 // default robot priority
	}

	// 2. Trigger type priority
	// Human intervention > Event > Clock
	triggerPriority := getTriggerPriority(item.Trigger)
	priority += triggerPriority * 100

	return priority
}

// getTriggerPriority returns priority value for trigger type
func getTriggerPriority(trigger types.TriggerType) int {
	switch trigger {
	case types.TriggerHuman:
		return 10 // highest priority
	case types.TriggerEvent:
		return 5 // medium priority
	case types.TriggerClock:
		return 1 // lowest priority
	default:
		return 0
	}
}
