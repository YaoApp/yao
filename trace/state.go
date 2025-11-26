package trace

import (
	"time"

	"github.com/yaoapp/kun/log"
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
	// Note: subscribers moved to SubscriptionManager (no longer in state)
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

// --- Subscriber Commands (REMOVED - now handled by SubscriptionManager) ---
// Subscriber management has been decoupled from state machine for better separation of concerns

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
		// subscribers removed - now managed by SubscriptionManager
	}

	// Process commands until channel is closed (on Release)
	// Note: We don't exit on context cancellation anymore - state machine should continue
	// running until Release() is called, which closes the channel
	for {
		cmd, ok := <-m.stateCmdChan
		if !ok {
			// Channel closed by Release(), exit cleanly
			return
		}
		cmd.execute(state)

		// Optional: Exit after processing completion (but only after draining)
		// This is mainly for optimization - the channel will be closed by Release() anyway
		if state.completed {
			// Drain remaining commands with timeout
			drainTimer := time.NewTimer(100 * time.Millisecond)
			defer drainTimer.Stop()
		drainLoop:
			for {
				select {
				case cmd, ok := <-m.stateCmdChan:
					if !ok {
						// Channel closed during drain
						break drainLoop
					}
					cmd.execute(state)
				case <-drainTimer.C:
					break drainLoop
				}
			}
			return
		}
	}
}

// Helper methods for manager to send commands

// safeSend checks if context is cancelled before sending to avoid panic on closed channel
func (m *manager) safeSend(cmd stateCommand) (ok bool) {
	// Use defer/recover to handle the case where channel is closed mid-send
	defer func() {
		if r := recover(); r != nil {
			// Channel was closed, silently return false
			ok = false
		}
	}()

	select {
	case <-m.ctx.Done():
		// Context cancelled, channel may be closed
		return false
	case m.stateCmdChan <- cmd:
		return true
	}
}

func (m *manager) stateSetRoot(node *types.TraceNode) {
	m.safeSend(&cmdSetRoot{node: node})
}

func (m *manager) stateGetRoot() *types.TraceNode {
	resp := make(chan *types.TraceNode, 1)
	if !m.safeSend(&cmdGetRoot{resp: resp}) {
		return nil // Context cancelled
	}
	return <-resp
}

func (m *manager) stateSetCurrentNodes(nodes []*types.TraceNode) {
	m.safeSend(&cmdSetCurrentNodes{nodes: nodes})
}

func (m *manager) stateGetCurrentNodes() []*types.TraceNode {
	resp := make(chan []*types.TraceNode, 1)
	if !m.safeSend(&cmdGetCurrentNodes{resp: resp}) {
		return nil // Context cancelled
	}
	return <-resp
}

func (m *manager) stateUpdateRootAndCurrent(root *types.TraceNode, current []*types.TraceNode) {
	m.safeSend(&cmdUpdateRootAndCurrent{root: root, current: current})
}

func (m *manager) stateGetSpace(id string) (*types.TraceSpace, bool) {
	resp := make(chan *types.TraceSpace, 1)
	if !m.safeSend(&cmdGetSpace{id: id, resp: resp}) {
		return nil, false // Context cancelled
	}
	space := <-resp
	return space, space != nil
}

func (m *manager) stateSetSpace(id string, space *types.TraceSpace) {
	m.safeSend(&cmdSetSpace{id: id, space: space})
}

func (m *manager) stateDeleteSpace(id string) {
	m.safeSend(&cmdDeleteSpace{id: id})
}

func (m *manager) stateGetAllSpaces() []*types.TraceSpace {
	resp := make(chan []*types.TraceSpace, 1)
	if !m.safeSend(&cmdGetAllSpaces{resp: resp}) {
		return nil // Context cancelled
	}
	return <-resp
}

func (m *manager) stateSetTraceStatus(status types.TraceStatus) {
	m.safeSend(&cmdSetTraceStatus{status: status})
}

func (m *manager) stateGetTraceStatus() types.TraceStatus {
	resp := make(chan types.TraceStatus, 1)
	if !m.safeSend(&cmdGetTraceStatus{resp: resp}) {
		return types.TraceStatusCancelled // Context cancelled
	}
	return <-resp
}

func (m *manager) stateMarkCompleted() bool {
	resp := make(chan bool, 1)
	if !m.safeSend(&cmdMarkCompleted{resp: resp}) {
		return true // Context cancelled, treat as completed
	}
	return <-resp
}

func (m *manager) stateIsCompleted() bool {
	resp := make(chan bool, 1)
	if !m.safeSend(&cmdIsCompleted{resp: resp}) {
		return true // Context cancelled, treat as completed
	}
	return <-resp
}

func (m *manager) stateAddUpdate(update *types.TraceUpdate) {
	m.safeSend(&cmdAddUpdate{update: update})
}

func (m *manager) stateGetUpdates(since int64) []*types.TraceUpdate {
	resp := make(chan []*types.TraceUpdate, 1)
	if !m.safeSend(&cmdGetUpdates{since: since, resp: resp}) {
		return nil // Context cancelled
	}
	return <-resp
}

func (m *manager) stateSetUpdates(updates []*types.TraceUpdate) {
	log.Trace("[STATE] stateSetUpdates: setting %d updates for trace %s", len(updates), m.traceID)
	m.safeSend(&cmdSetUpdates{updates: updates})
}

// Subscription management methods removed - now handled by SubscriptionManager
// See subscription_manager.go and subscription.go for the new implementation

// stateExecuteSpaceOp executes a space operation serially through state worker
func (m *manager) stateExecuteSpaceOp(spaceID string, fn func() error) error {
	resp := make(chan error, 1)
	m.stateCmdChan <- &cmdSpaceKVOp{spaceID: spaceID, fn: fn, resp: resp}
	return <-resp
}
