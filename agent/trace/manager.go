package trace

import (
	"context"
	"fmt"
	"sync"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/yaoapp/yao/agent/trace/types"
)

// manager implements the Manager interface with unified business logic
type manager struct {
	ctx          context.Context
	traceID      string
	driver       types.Driver
	rootNode     *types.TraceNode
	currentNodes []*types.TraceNode
	spaces       map[string]*types.TraceSpace
	mu           sync.RWMutex // Protects currentNodes and spaces

	// Subscription mechanism
	updates     []*types.TraceUpdate               // Update history (all events)
	updatesMu   sync.RWMutex                       // Protects updates
	subscribers map[string]chan *types.TraceUpdate // Active subscribers
	subMu       sync.RWMutex                       // Protects subscribers
	completed   bool                               // Trace completion status
}

// NewManager creates a new trace manager instance
func NewManager(ctx context.Context, traceID string, driver types.Driver) (types.Manager, error) {
	// Create root node
	now := time.Now().Unix()
	rootNode := &types.TraceNode{
		ID:        genNodeID(),
		ParentID:  "",
		Children:  []*types.TraceNode{},
		Status:    types.StatusRunning,
		CreatedAt: now,
		StartTime: now,
		UpdatedAt: now,
		TraceNodeOption: types.TraceNodeOption{
			Label: "Root",
			Icon:  "root",
		},
	}

	// Save root node
	if err := driver.SaveNode(ctx, traceID, rootNode); err != nil {
		return nil, fmt.Errorf("failed to save root node: %w", err)
	}

	m := &manager{
		ctx:          ctx,
		traceID:      traceID,
		driver:       driver,
		rootNode:     rootNode,
		currentNodes: []*types.TraceNode{rootNode},
		spaces:       make(map[string]*types.TraceSpace),
		updates:      make([]*types.TraceUpdate, 0, 100),
		subscribers:  make(map[string]chan *types.TraceUpdate),
		completed:    false,
	}

	// Broadcast init event
	m.addUpdate(&types.TraceUpdate{
		Type:      types.UpdateTypeInit,
		TraceID:   traceID,
		Timestamp: now,
		Data:      types.NewTraceInitData(traceID, rootNode),
	})

	// Broadcast root node start event
	m.addUpdate(&types.TraceUpdate{
		Type:      types.UpdateTypeNodeStart,
		TraceID:   traceID,
		NodeID:    rootNode.ID,
		Timestamp: now,
		Data:      rootNode.ToStartData(),
	})

	return m, nil
}

// genNodeID generates a unique node ID
func genNodeID() string {
	id, _ := gonanoid.Generate("0123456789abcdefghijklmnopqrstuvwxyz", 12)
	return id
}

// checkContext checks if context is cancelled
func (m *manager) checkContext() error {
	select {
	case <-m.ctx.Done():
		return m.ctx.Err()
	default:
		return nil
	}
}

// newNode creates a node instance that broadcasts events (for external use)
func (m *manager) newNode(data *types.TraceNode) types.Node {
	return &node{
		manager: m,
		data:    data,
	}
}

// Helper functions for thread-safe access

// getCurrentNodes returns a copy of current nodes (thread-safe read)
func (m *manager) getCurrentNodes() []*types.TraceNode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	nodes := make([]*types.TraceNode, len(m.currentNodes))
	copy(nodes, m.currentNodes)
	return nodes
}

// getSpace returns a space by ID (thread-safe read)
func (m *manager) getSpace(id string) (*types.TraceSpace, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	space, ok := m.spaces[id]
	return space, ok
}

// setSpace stores a space (thread-safe write)
func (m *manager) setSpace(id string, space *types.TraceSpace) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.spaces[id] = space
}

// deleteSpace removes a space (thread-safe write)
func (m *manager) deleteSpace(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.spaces, id)
}

// getAllSpaces returns all spaces (thread-safe read)
func (m *manager) getAllSpaces() []*types.TraceSpace {
	m.mu.RLock()
	defer m.mu.RUnlock()
	spaces := make([]*types.TraceSpace, 0, len(m.spaces))
	for _, space := range m.spaces {
		spaces = append(spaces, space)
	}
	return spaces
}

// Add creates next sequential node - auto-joins if currently in parallel state
func (m *manager) Add(input types.TraceInput, option types.TraceNodeOption) (types.Node, error) {
	if err := m.checkContext(); err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().Unix()

	// If in parallel state (multiple current nodes), auto-join first
	var parentNode *types.TraceNode
	if len(m.currentNodes) > 1 {
		// Auto-join: create join node
		parentNode = &types.TraceNode{
			ID:              genNodeID(),
			ParentID:        m.currentNodes[0].ParentID, // Same parent as parallel nodes
			Children:        []*types.TraceNode{},
			Status:          types.StatusCompleted,
			CreatedAt:       now,
			StartTime:       now,
			EndTime:         now,
			UpdatedAt:       now,
			TraceNodeOption: types.TraceNodeOption{Label: "Join", Icon: "join"},
		}
		// Save join node
		if err := m.driver.SaveNode(m.ctx, m.traceID, parentNode); err != nil {
			return nil, err
		}
	} else {
		parentNode = m.currentNodes[0]
	}

	// Create new node data
	newNodeData := &types.TraceNode{
		ID:              genNodeID(),
		ParentID:        parentNode.ID,
		Children:        []*types.TraceNode{},
		TraceNodeOption: option,
		Status:          types.StatusRunning,
		Input:           input,
		CreatedAt:       now,
		StartTime:       now,
		UpdatedAt:       now,
	}

	// Add to parent's children
	parentNode.Children = append(parentNode.Children, newNodeData)

	// Save nodes
	if err := m.driver.SaveNode(m.ctx, m.traceID, newNodeData); err != nil {
		return nil, err
	}
	if err := m.driver.SaveNode(m.ctx, m.traceID, parentNode); err != nil {
		return nil, err
	}

	// Set as current node
	m.currentNodes = []*types.TraceNode{newNodeData}

	// Broadcast node start event
	m.addUpdate(&types.TraceUpdate{
		Type:      types.UpdateTypeNodeStart,
		TraceID:   m.traceID,
		NodeID:    newNodeData.ID,
		Timestamp: now,
		Data:      newNodeData.ToStartData(),
	})

	// Return Node interface
	return &node{
		manager: m,
		data:    newNodeData,
	}, nil
}

// Parallel creates multiple concurrent child nodes, returns Node interfaces for direct control
func (m *manager) Parallel(parallelInputs []types.TraceParallelInput) ([]types.Node, error) {
	if err := m.checkContext(); err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().Unix()
	parentNode := m.currentNodes[0]
	nodeData := make([]*types.TraceNode, 0, len(parallelInputs))
	nodeInterfaces := make([]types.Node, 0, len(parallelInputs))

	// Create multiple child nodes
	for _, input := range parallelInputs {
		data := &types.TraceNode{
			ID:              genNodeID(),
			ParentID:        parentNode.ID,
			Children:        []*types.TraceNode{},
			TraceNodeOption: input.Option,
			Status:          types.StatusRunning,
			Input:           input.Input,
			CreatedAt:       now,
			StartTime:       now,
			UpdatedAt:       now,
		}
		nodeData = append(nodeData, data)
		parentNode.Children = append(parentNode.Children, data)

		// Save node
		if err := m.driver.SaveNode(m.ctx, m.traceID, data); err != nil {
			return nil, err
		}

		// Create Node interface wrapper
		nodeInterfaces = append(nodeInterfaces, &node{
			manager: m,
			data:    data,
		})
	}

	// Save parent node
	if err := m.driver.SaveNode(m.ctx, m.traceID, parentNode); err != nil {
		return nil, err
	}

	// Set all as current nodes (parallel state)
	m.currentNodes = nodeData

	// Broadcast parallel nodes as batch (frontend supports data.nodes[])
	m.addUpdate(&types.TraceUpdate{
		Type:      types.UpdateTypeNodeStart,
		TraceID:   m.traceID,
		Timestamp: now,
		Data:      types.NodesToStartData(nodeData),
	})

	return nodeInterfaces, nil
}

// Info logs info message to current node(s)
func (m *manager) Info(format string, args ...any) types.Manager {
	m.log("info", format, args...)
	return m
}

// Debug logs debug message to current node(s)
func (m *manager) Debug(format string, args ...any) types.Manager {
	m.log("debug", format, args...)
	return m
}

// Error logs error message to current node(s)
func (m *manager) Error(format string, args ...any) types.Manager {
	m.log("error", format, args...)
	return m
}

// Warn logs warning message to current node(s)
func (m *manager) Warn(format string, args ...any) types.Manager {
	m.log("warn", format, args...)
	return m
}

// log helper method to log messages
func (m *manager) log(level string, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	now := time.Now().Unix()

	// Get current nodes safely
	nodes := m.getCurrentNodes()

	// Log to all current nodes
	for _, node := range nodes {
		log := &types.TraceLog{
			Timestamp: now,
			Level:     level,
			Message:   message,
			NodeID:    node.ID,
		}
		// Save log (ignore errors for non-critical logging)
		_ = m.driver.SaveLog(m.ctx, m.traceID, log)

		// Broadcast log event
		m.addUpdate(&types.TraceUpdate{
			Type:      types.UpdateTypeLogAdded,
			TraceID:   m.traceID,
			NodeID:    node.ID,
			Timestamp: now,
			Data:      log,
		})
	}
}

// SetOutput sets output for current node(s)
func (m *manager) SetOutput(output types.TraceOutput) error {
	if err := m.checkContext(); err != nil {
		return err
	}

	now := time.Now().Unix()
	nodes := m.getCurrentNodes()
	for _, node := range nodes {
		node.Output = output
		node.UpdatedAt = now
		if err := m.driver.SaveNode(m.ctx, m.traceID, node); err != nil {
			return err
		}

		// Broadcast node update event
		m.addUpdate(&types.TraceUpdate{
			Type:      types.UpdateTypeNodeUpdated,
			TraceID:   m.traceID,
			NodeID:    node.ID,
			Timestamp: now,
			Data:      node,
		})
	}
	return nil
}

// SetMetadata sets metadata for current node(s)
func (m *manager) SetMetadata(key string, value any) error {
	if err := m.checkContext(); err != nil {
		return err
	}

	now := time.Now().Unix()
	nodes := m.getCurrentNodes()
	for _, node := range nodes {
		if node.Metadata == nil {
			node.Metadata = make(map[string]any)
		}
		node.Metadata[key] = value
		node.UpdatedAt = now
		if err := m.driver.SaveNode(m.ctx, m.traceID, node); err != nil {
			return err
		}

		// Broadcast node update event
		m.addUpdate(&types.TraceUpdate{
			Type:      types.UpdateTypeNodeUpdated,
			TraceID:   m.traceID,
			NodeID:    node.ID,
			Timestamp: now,
			Data:      node.ToStartData(), // Send updated node
		})
	}
	return nil
}

// Complete marks current node(s) as completed
// Optional output parameter: if provided, sets the output before completing
func (m *manager) Complete(output ...types.TraceOutput) error {
	if err := m.checkContext(); err != nil {
		return err
	}

	now := time.Now().Unix()
	nodes := m.getCurrentNodes()

	// Set output if provided
	if len(output) > 0 {
		for _, node := range nodes {
			node.Output = output[0]
		}
	}

	for _, node := range nodes {
		node.Status = types.StatusCompleted
		node.EndTime = now
		node.UpdatedAt = now
		if err := m.driver.SaveNode(m.ctx, m.traceID, node); err != nil {
			return err
		}

		// Broadcast node complete event
		m.addUpdate(&types.TraceUpdate{
			Type:      types.UpdateTypeNodeComplete,
			TraceID:   m.traceID,
			NodeID:    node.ID,
			Timestamp: now,
			Data:      node.ToCompleteData(),
		})
	}
	return nil
}

// Fail marks current node(s) as failed
func (m *manager) Fail(err error) error {
	if ctxErr := m.checkContext(); ctxErr != nil {
		return ctxErr
	}

	now := time.Now().Unix()
	// Log error first
	m.Error("Node failed: %v", err)

	nodes := m.getCurrentNodes()
	for _, node := range nodes {
		node.Status = types.StatusFailed
		node.EndTime = now
		node.UpdatedAt = now
		if saveErr := m.driver.SaveNode(m.ctx, m.traceID, node); saveErr != nil {
			return saveErr
		}

		// Broadcast node failed event
		m.addUpdate(&types.TraceUpdate{
			Type:      types.UpdateTypeNodeFailed,
			TraceID:   m.traceID,
			NodeID:    node.ID,
			Timestamp: now,
			Data: &types.NodeFailedData{
				NodeID:   node.ID,
				Status:   "failed",
				EndTime:  now,
				Duration: (node.EndTime - node.StartTime) * 1000, // Convert to milliseconds
				Error:    err.Error(),
			},
		})
	}
	return nil
}

// GetRootNode returns the root node
func (m *manager) GetRootNode() (*types.TraceNode, error) {
	return m.rootNode, nil
}

// GetNode returns a node by ID
func (m *manager) GetNode(id string) (*types.TraceNode, error) {
	return m.driver.LoadNode(m.ctx, m.traceID, id)
}

// GetCurrentNodes returns current active nodes
func (m *manager) GetCurrentNodes() ([]*types.TraceNode, error) {
	return m.getCurrentNodes(), nil
}

// MarkComplete marks the entire trace as completed
func (m *manager) MarkComplete() error {
	m.updatesMu.Lock()
	if m.completed {
		m.updatesMu.Unlock()
		return nil // Already completed
	}
	m.completed = true
	m.updatesMu.Unlock()

	// Calculate total duration from root node
	now := time.Now().Unix()
	totalDuration := int64(0)
	if m.rootNode != nil && m.rootNode.CreatedAt > 0 {
		totalDuration = (now - m.rootNode.CreatedAt) * 1000 // Convert to milliseconds
	}

	// Broadcast completion event
	m.addUpdate(&types.TraceUpdate{
		Type:      types.UpdateTypeComplete,
		TraceID:   m.traceID,
		Timestamp: now,
		Data:      types.NewTraceCompleteData(m.traceID, totalDuration),
	})

	return nil
}

// CreateSpace creates a new memory space
func (m *manager) CreateSpace(option types.TraceSpaceOption) (*types.TraceSpace, error) {
	if err := m.checkContext(); err != nil {
		return nil, err
	}

	now := time.Now().Unix()

	// Create space instance
	space := &types.TraceSpace{
		ID:               genNodeID(), // Reuse node ID generator
		TraceSpaceOption: option,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Save to driver
	if err := m.driver.SaveSpace(m.ctx, m.traceID, space); err != nil {
		return nil, err
	}

	// Cache in memory (thread-safe)
	m.setSpace(space.ID, space)

	// Broadcast space created event
	m.addUpdate(&types.TraceUpdate{
		Type:      types.UpdateTypeSpaceCreated,
		TraceID:   m.traceID,
		SpaceID:   space.ID,
		Timestamp: now,
		Data:      space,
	})

	return space, nil
}

// GetSpace returns a space by ID
func (m *manager) GetSpace(id string) (*types.TraceSpace, error) {
	// Check cache first (thread-safe)
	if space, ok := m.getSpace(id); ok {
		return space, nil
	}

	// Load from driver
	space, err := m.driver.LoadSpace(m.ctx, m.traceID, id)
	if err != nil {
		return nil, err
	}

	// Cache it (thread-safe)
	if space != nil {
		m.setSpace(id, space)
	}

	return space, nil
}

// HasSpace checks if a space exists
func (m *manager) HasSpace(id string) bool {
	// Check cache (thread-safe)
	if _, ok := m.getSpace(id); ok {
		return true
	}

	// Check in driver
	space, _ := m.driver.LoadSpace(m.ctx, m.traceID, id)
	return space != nil
}

// DeleteSpace deletes a space
func (m *manager) DeleteSpace(id string) error {
	if err := m.checkContext(); err != nil {
		return err
	}

	now := time.Now().Unix()

	// Remove from cache (thread-safe)
	m.deleteSpace(id)

	// Delete from driver
	if err := m.driver.DeleteSpace(m.ctx, m.traceID, id); err != nil {
		return err
	}

	// Broadcast space deleted event
	m.addUpdate(&types.TraceUpdate{
		Type:      types.UpdateTypeSpaceDeleted,
		TraceID:   m.traceID,
		SpaceID:   id,
		Timestamp: now,
		Data:      types.NewSpaceDeletedData(id),
	})

	return nil
}

// ListSpaces returns all spaces
func (m *manager) ListSpaces() []*types.TraceSpace {
	// Load from driver to ensure we have all spaces
	spaceIDs, err := m.driver.ListSpaces(m.ctx, m.traceID)
	if err != nil {
		// Fallback to cached spaces (thread-safe)
		return m.getAllSpaces()
	}

	// Load all spaces
	spaces := make([]*types.TraceSpace, 0, len(spaceIDs))
	for _, id := range spaceIDs {
		space, err := m.GetSpace(id) // Use GetSpace to leverage cache
		if err == nil && space != nil {
			spaces = append(spaces, space)
		}
	}

	return spaces
}

// SetSpaceValue sets a value in a space and broadcasts memory_add event
func (m *manager) SetSpaceValue(spaceID, key string, value any) error {
	if err := m.checkContext(); err != nil {
		return err
	}

	now := time.Now().Unix()

	// Get space
	space, err := m.GetSpace(spaceID)
	if err != nil || space == nil {
		return fmt.Errorf("space not found: %s", spaceID)
	}

	// Set value in driver
	if err := m.driver.SetSpaceKey(m.ctx, m.traceID, spaceID, key, value); err != nil {
		return err
	}

	// Update space timestamp
	space.UpdatedAt = now
	if err := m.driver.SaveSpace(m.ctx, m.traceID, space); err != nil {
		return err
	}

	// Broadcast memory_add event
	m.addUpdate(&types.TraceUpdate{
		Type:      types.UpdateTypeMemoryAdd,
		TraceID:   m.traceID,
		SpaceID:   spaceID,
		Timestamp: now,
		Data:      space.ToMemoryAddData(key, value, now),
	})

	return nil
}

// GetSpaceValue gets a value from a space
func (m *manager) GetSpaceValue(spaceID, key string) (any, error) {
	return m.driver.GetSpaceKey(m.ctx, m.traceID, spaceID, key)
}

// HasSpaceValue checks if a key exists in a space
func (m *manager) HasSpaceValue(spaceID, key string) bool {
	return m.driver.HasSpaceKey(m.ctx, m.traceID, spaceID, key)
}

// DeleteSpaceValue deletes a value from a space and broadcasts memory_delete event
func (m *manager) DeleteSpaceValue(spaceID, key string) error {
	if err := m.checkContext(); err != nil {
		return err
	}

	now := time.Now().Unix()

	// Delete value from driver
	if err := m.driver.DeleteSpaceKey(m.ctx, m.traceID, spaceID, key); err != nil {
		return err
	}

	// Broadcast memory_delete event
	m.addUpdate(&types.TraceUpdate{
		Type:      types.UpdateTypeMemoryDelete,
		TraceID:   m.traceID,
		SpaceID:   spaceID,
		Timestamp: now,
		Data:      types.NewMemoryDeleteData(spaceID, key),
	})

	return nil
}

// ClearSpaceValues clears all values from a space
func (m *manager) ClearSpaceValues(spaceID string) error {
	if err := m.checkContext(); err != nil {
		return err
	}

	now := time.Now().Unix()

	// Clear values from driver
	if err := m.driver.ClearSpaceKeys(m.ctx, m.traceID, spaceID); err != nil {
		return err
	}

	// Broadcast memory_delete event (for all keys)
	m.addUpdate(&types.TraceUpdate{
		Type:      types.UpdateTypeMemoryDelete,
		TraceID:   m.traceID,
		SpaceID:   spaceID,
		Timestamp: now,
		Data:      types.NewMemoryDeleteAllData(spaceID),
	})

	return nil
}

// ListSpaceKeys returns all keys in a space
func (m *manager) ListSpaceKeys(spaceID string) []string {
	keys, err := m.driver.ListSpaceKeys(m.ctx, m.traceID, spaceID)
	if err != nil {
		return nil
	}
	return keys
}
