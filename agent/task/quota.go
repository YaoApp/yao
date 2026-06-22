package task

import (
	"container/heap"
	"sync"
)

// QuotaManager manages per-team concurrent execution limits with priority queue
type QuotaManager struct {
	mu      sync.Mutex
	running map[string]int
	queue   map[string]*priorityQueue
	limits  map[string]int
}

type queueEntry struct {
	chatID   string
	teamID   string
	priority int
	index    int
	ready    chan struct{}
}

// GlobalQuota is the singleton quota manager
var GlobalQuota = &QuotaManager{
	running: make(map[string]int),
	queue:   make(map[string]*priorityQueue),
	limits:  make(map[string]int),
}

const defaultQuotaLimit = 3

// TryAcquire atomically checks and increments running count.
// Returns true if slot acquired, false if at limit.
func (qm *QuotaManager) TryAcquire(teamID, roleID string) bool {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	limit := qm.resolveLimit(teamID, roleID)
	if qm.running[teamID] >= limit {
		return false
	}
	qm.running[teamID]++
	return true
}

// Release decrements running count and signals next queued entry
func (qm *QuotaManager) Release(teamID string) {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	qm.running[teamID]--
	if qm.running[teamID] < 0 {
		qm.running[teamID] = 0
	}
	pq := qm.queue[teamID]
	if pq != nil && pq.Len() > 0 {
		next := heap.Pop(pq).(*queueEntry)
		qm.running[teamID]++
		close(next.ready)
	}
}

// Enqueue adds a task to the priority queue, returns entry with ready channel
func (qm *QuotaManager) Enqueue(teamID, chatID string, priority int) *queueEntry {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	pq := qm.queue[teamID]
	if pq == nil {
		pq = &priorityQueue{}
		heap.Init(pq)
		qm.queue[teamID] = pq
	}
	entry := &queueEntry{
		chatID:   chatID,
		teamID:   teamID,
		priority: priority,
		ready:    make(chan struct{}),
	}
	heap.Push(pq, entry)
	return entry
}

// Dequeue removes a specific entry from the queue (e.g. on cancel)
func (qm *QuotaManager) Dequeue(teamID, chatID string) {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	pq := qm.queue[teamID]
	if pq == nil {
		return
	}
	for i, e := range *pq {
		if e.chatID == chatID {
			heap.Remove(pq, i)
			return
		}
	}
}

// QueuePosition returns current position in queue (1-based), 0 if not found
func (qm *QuotaManager) QueuePosition(teamID, chatID string) int {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	pq := qm.queue[teamID]
	if pq == nil {
		return 0
	}
	for i, e := range *pq {
		if e.chatID == chatID {
			return i + 1
		}
	}
	return 0
}

func (qm *QuotaManager) resolveLimit(teamID, roleID string) int {
	if limit, ok := qm.limits[teamID]; ok {
		return limit
	}
	return defaultQuotaLimit
}

// priorityQueue implements heap.Interface
// Higher priority number = higher priority; same priority uses insertion order
type priorityQueue []*queueEntry

func (pq priorityQueue) Len() int { return len(pq) }
func (pq priorityQueue) Less(i, j int) bool {
	if pq[i].priority != pq[j].priority {
		return pq[i].priority > pq[j].priority
	}
	return pq[i].index < pq[j].index
}
func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}
func (pq *priorityQueue) Push(x interface{}) {
	n := len(*pq)
	entry := x.(*queueEntry)
	entry.index = n
	*pq = append(*pq, entry)
}
func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	entry := old[n-1]
	old[n-1] = nil
	entry.index = -1
	*pq = old[:n-1]
	return entry
}
