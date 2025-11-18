package trace

import (
	"time"

	"github.com/yaoapp/yao/trace/types"
)

// State management using channel-based serialization (no locks needed)
// All state mutations go through a single worker goroutine

// managerState holds all mutable state (accessed only by state worker)
type managerState struct {
	rootNode     *types.TraceNode
	currentNodes []*types.TraceNode
	spaces       map[string]*types.TraceSpace
	traceStatus  types.TraceStatus
	completed    bool
	updates      []*types.TraceUpdate
	subscribers  map[string]chan *types.TraceUpdate
}

// State command interface - all commands are processed serially
type stateCommand interface {
	execute(s *managerState)
}

// Commands with response channels for synchronous operations

// --- Root Node Commands ---

type cmdSetRoot struct {
	node *types.TraceNode
}

func (c *cmdSetRoot) execute(s *managerState) {
	s.rootNode = c.node
}

type cmdGetRoot struct {
	resp chan *types.TraceNode
}

func (c *cmdGetRoot) execute(s *managerState) {
	c.resp <- s.rootNode
}

// --- Current Nodes Commands ---

type cmdSetCurrentNodes struct {
	nodes []*types.TraceNode
}

func (c *cmdSetCurrentNodes) execute(s *managerState) {
	s.currentNodes = c.nodes
}

type cmdGetCurrentNodes struct {
	resp chan []*types.TraceNode
}

func (c *cmdGetCurrentNodes) execute(s *managerState) {
	// Return a copy to prevent external mutation
	nodes := make([]*types.TraceNode, len(s.currentNodes))
	copy(nodes, s.currentNodes)
	c.resp <- nodes
}

type cmdUpdateRootAndCurrent struct {
	root    *types.TraceNode
	current []*types.TraceNode
}

func (c *cmdUpdateRootAndCurrent) execute(s *managerState) {
	s.rootNode = c.root
	s.currentNodes = c.current
}

// --- Space Commands ---

type cmdGetSpace struct {
	id   string
	resp chan *types.TraceSpace
}

func (c *cmdGetSpace) execute(s *managerState) {
	c.resp <- s.spaces[c.id]
}

type cmdSetSpace struct {
	id    string
	space *types.TraceSpace
}

func (c *cmdSetSpace) execute(s *managerState) {
	s.spaces[c.id] = c.space
}

type cmdDeleteSpace struct {
	id string
}

func (c *cmdDeleteSpace) execute(s *managerState) {
	delete(s.spaces, c.id)
}

type cmdGetAllSpaces struct {
	resp chan []*types.TraceSpace
}

func (c *cmdGetAllSpaces) execute(s *managerState) {
	spaces := make([]*types.TraceSpace, 0, len(s.spaces))
	for _, space := range s.spaces {
		spaces = append(spaces, space)
	}
	c.resp <- spaces
}

// --- Trace Status Commands ---

type cmdSetTraceStatus struct {
	status types.TraceStatus
}

func (c *cmdSetTraceStatus) execute(s *managerState) {
	s.traceStatus = c.status
}

type cmdGetTraceStatus struct {
	resp chan types.TraceStatus
}

func (c *cmdGetTraceStatus) execute(s *managerState) {
	c.resp <- s.traceStatus
}

// --- Completion Commands ---

type cmdMarkCompleted struct {
	resp chan bool // Returns true if marked, false if already completed
}

func (c *cmdMarkCompleted) execute(s *managerState) {
	if s.completed {
		c.resp <- false
	} else {
		s.completed = true
		c.resp <- true
	}
}

type cmdIsCompleted struct {
	resp chan bool
}

func (c *cmdIsCompleted) execute(s *managerState) {
	c.resp <- s.completed
}

// --- Update Commands ---

type cmdAddUpdate struct {
	update *types.TraceUpdate
}

func (c *cmdAddUpdate) execute(s *managerState) {
	s.updates = append(s.updates, c.update)
}

type cmdGetUpdates struct {
	since int64
	resp  chan []*types.TraceUpdate
}

func (c *cmdGetUpdates) execute(s *managerState) {
	filtered := make([]*types.TraceUpdate, 0)
	for _, update := range s.updates {
		if update.Timestamp >= c.since {
			filtered = append(filtered, update)
		}
	}
	c.resp <- filtered
}

type cmdSetUpdates struct {
	updates []*types.TraceUpdate
}

func (c *cmdSetUpdates) execute(s *managerState) {
	s.updates = c.updates
}

// --- Subscriber Commands ---

type cmdAddSubscriber struct {
	id string
	ch chan *types.TraceUpdate
}

func (c *cmdAddSubscriber) execute(s *managerState) {
	s.subscribers[c.id] = c.ch
}

type cmdRemoveSubscriber struct {
	id string
}

func (c *cmdRemoveSubscriber) execute(s *managerState) {
	delete(s.subscribers, c.id)
}

type cmdGetSubscribers struct {
	resp chan map[string]chan *types.TraceUpdate
}

func (c *cmdGetSubscribers) execute(s *managerState) {
	// Return a copy of the map
	subs := make(map[string]chan *types.TraceUpdate, len(s.subscribers))
	for id, ch := range s.subscribers {
		subs[id] = ch
	}
	c.resp <- subs
}

// --- Broadcast Command (special - sends to all subscribers) ---

type cmdBroadcast struct {
	update *types.TraceUpdate
}

func (c *cmdBroadcast) execute(s *managerState) {
	// Send to all subscribers (non-blocking, with panic recovery)
	for _, ch := range s.subscribers {
		func(channel chan *types.TraceUpdate) {
			defer func() {
				// Recover from panic if channel is closed
				if r := recover(); r != nil {
					// Channel was closed, ignore
				}
			}()
			select {
			case channel <- c.update:
			default:
				// Subscriber is slow, skip (non-blocking)
			}
		}(ch)
	}
}

// --- Space KV Commands (for concurrent safety) ---
// These ensure all operations on a space are serialized through state worker

type cmdSpaceKVOp struct {
	spaceID string
	fn      func() error
	resp    chan error
}

func (c *cmdSpaceKVOp) execute(s *managerState) {
	// Execute the operation (typically a driver call)
	// The function is provided by caller and executed serially here
	err := c.fn()
	c.resp <- err
}

// State worker - processes all commands serially in a single goroutine
func (m *manager) startStateWorker() {
	// Initialize state
	state := &managerState{
		rootNode:     nil,
		currentNodes: []*types.TraceNode{},
		spaces:       make(map[string]*types.TraceSpace),
		traceStatus:  types.TraceStatusPending,
		completed:    false,
		updates:      make([]*types.TraceUpdate, 0, 100),
		subscribers:  make(map[string]chan *types.TraceUpdate),
	}

	// Process commands until context is cancelled or trace is completed
	for {
		select {
		case cmd, ok := <-m.stateCmdChan:
			if !ok {
				// Channel closed
				return
			}
			cmd.execute(state)

			// Exit after processing completion
			if state.completed {
				// Drain remaining commands with timeout
				drainTimer := time.NewTimer(100 * time.Millisecond)
				defer drainTimer.Stop()
			drainLoop:
				for {
					select {
					case cmd := <-m.stateCmdChan:
						cmd.execute(state)
					case <-drainTimer.C:
						break drainLoop
					}
				}
				return
			}
		case <-m.ctx.Done():
			// Context cancelled - continue processing for a short time to handle cancellation
			// Then exit to prevent deadlock
			time.Sleep(10 * time.Millisecond)
			return
		}
	}
}

// Helper methods for manager to send commands

func (m *manager) stateSetRoot(node *types.TraceNode) {
	m.stateCmdChan <- &cmdSetRoot{node: node}
}

func (m *manager) stateGetRoot() *types.TraceNode {
	resp := make(chan *types.TraceNode, 1)
	m.stateCmdChan <- &cmdGetRoot{resp: resp}
	return <-resp
}

func (m *manager) stateSetCurrentNodes(nodes []*types.TraceNode) {
	m.stateCmdChan <- &cmdSetCurrentNodes{nodes: nodes}
}

func (m *manager) stateGetCurrentNodes() []*types.TraceNode {
	resp := make(chan []*types.TraceNode, 1)
	m.stateCmdChan <- &cmdGetCurrentNodes{resp: resp}
	return <-resp
}

func (m *manager) stateUpdateRootAndCurrent(root *types.TraceNode, current []*types.TraceNode) {
	m.stateCmdChan <- &cmdUpdateRootAndCurrent{root: root, current: current}
}

func (m *manager) stateGetSpace(id string) (*types.TraceSpace, bool) {
	resp := make(chan *types.TraceSpace, 1)
	m.stateCmdChan <- &cmdGetSpace{id: id, resp: resp}
	space := <-resp
	return space, space != nil
}

func (m *manager) stateSetSpace(id string, space *types.TraceSpace) {
	m.stateCmdChan <- &cmdSetSpace{id: id, space: space}
}

func (m *manager) stateDeleteSpace(id string) {
	m.stateCmdChan <- &cmdDeleteSpace{id: id}
}

func (m *manager) stateGetAllSpaces() []*types.TraceSpace {
	resp := make(chan []*types.TraceSpace, 1)
	m.stateCmdChan <- &cmdGetAllSpaces{resp: resp}
	return <-resp
}

func (m *manager) stateSetTraceStatus(status types.TraceStatus) {
	m.stateCmdChan <- &cmdSetTraceStatus{status: status}
}

func (m *manager) stateGetTraceStatus() types.TraceStatus {
	resp := make(chan types.TraceStatus, 1)
	m.stateCmdChan <- &cmdGetTraceStatus{resp: resp}
	return <-resp
}

func (m *manager) stateMarkCompleted() bool {
	resp := make(chan bool, 1)
	m.stateCmdChan <- &cmdMarkCompleted{resp: resp}
	return <-resp
}

func (m *manager) stateIsCompleted() bool {
	resp := make(chan bool, 1)
	m.stateCmdChan <- &cmdIsCompleted{resp: resp}
	return <-resp
}

func (m *manager) stateAddUpdate(update *types.TraceUpdate) {
	m.stateCmdChan <- &cmdAddUpdate{update: update}
}

func (m *manager) stateGetUpdates(since int64) []*types.TraceUpdate {
	resp := make(chan []*types.TraceUpdate, 1)
	m.stateCmdChan <- &cmdGetUpdates{since: since, resp: resp}
	return <-resp
}

func (m *manager) stateSetUpdates(updates []*types.TraceUpdate) {
	m.stateCmdChan <- &cmdSetUpdates{updates: updates}
}

func (m *manager) stateAddSubscriber(id string, ch chan *types.TraceUpdate) {
	m.stateCmdChan <- &cmdAddSubscriber{id: id, ch: ch}
}

func (m *manager) stateRemoveSubscriber(id string) {
	m.stateCmdChan <- &cmdRemoveSubscriber{id: id}
}

func (m *manager) stateGetSubscribers() map[string]chan *types.TraceUpdate {
	resp := make(chan map[string]chan *types.TraceUpdate, 1)
	m.stateCmdChan <- &cmdGetSubscribers{resp: resp}
	return <-resp
}

func (m *manager) stateBroadcast(update *types.TraceUpdate) {
	m.stateCmdChan <- &cmdBroadcast{update: update}
}

// stateExecuteSpaceOp executes a space operation serially through state worker
func (m *manager) stateExecuteSpaceOp(spaceID string, fn func() error) error {
	resp := make(chan error, 1)
	m.stateCmdChan <- &cmdSpaceKVOp{spaceID: spaceID, fn: fn, resp: resp}
	return <-resp
}
