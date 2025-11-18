package trace

import (
	"context"
	"fmt"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/yaoapp/yao/trace/types"
)

// manager implements the Manager interface with channel-based state management
type manager struct {
	ctx          context.Context
	cancel       context.CancelFunc
	traceID      string
	driver       types.Driver
	stateCmdChan chan stateCommand // Single channel for all state mutations
}

// NewManager creates a new trace manager instance
func NewManager(ctx context.Context, traceID string, driver types.Driver) (types.Manager, error) {
	// Create a cancellable context for the manager
	managerCtx, cancel := context.WithCancel(ctx)

	m := &manager{
		ctx:          managerCtx,
		cancel:       cancel,
		traceID:      traceID,
		driver:       driver,
		stateCmdChan: make(chan stateCommand, 100), // Buffered channel for performance
	}

	// Start state worker goroutine
	go m.startStateWorker()

	// Try to load existing updates from driver (for resumed traces)
	if existingUpdates, err := driver.LoadUpdates(ctx, traceID, 0); err == nil && len(existingUpdates) > 0 {
		m.stateSetUpdates(existingUpdates)
		// Check if trace was already completed
		for _, update := range existingUpdates {
			if update.Type == types.UpdateTypeComplete {
				m.stateMarkCompleted()
				if data, ok := update.Data.(*types.TraceCompleteData); ok {
					m.stateSetTraceStatus(data.Status)
				}
				break
			}
		}
	} else {
		// New trace - create and broadcast init event
		now := time.Now().Unix()
		m.addUpdateAndBroadcast(&types.TraceUpdate{
			Type:      types.UpdateTypeInit,
			TraceID:   traceID,
			Timestamp: now,
			Data:      types.NewTraceInitData(traceID, nil),
		})
	}

	return m, nil
}

// genNodeID generates a unique node ID
func genNodeID() string {
	id, _ := gonanoid.Generate("0123456789abcdefghijklmnopqrstuvwxyz", 12)
	return id
}

// addUpdateAndBroadcast persists, adds to history, and broadcasts an update
func (m *manager) addUpdateAndBroadcast(update *types.TraceUpdate) {
	// Persist to driver (synchronous - no race)
	_ = m.driver.SaveUpdate(context.Background(), m.traceID, update)

	// Add to in-memory history
	m.stateAddUpdate(update)

	// Broadcast to subscribers
	m.stateBroadcast(update)
}

// checkContext checks if context is cancelled
func (m *manager) checkContext() error {
	select {
	case <-m.ctx.Done():
		// Context cancelled - just return the error
		// Don't call handleCancellation here to avoid deadlock
		// handleCancellation should be called explicitly when needed
		return m.ctx.Err()
	default:
		return nil
	}
}

// handleCancellation marks nodes and trace as cancelled (called when context is done)
func (m *manager) handleCancellation() {
	// Mark as completed first - this will trigger state worker to exit
	// IMPORTANT: Must mark completed before any state queries to prevent deadlock
	if !m.stateMarkCompleted() {
		return // Already completed
	}

	now := time.Now().Unix()

	// Get current nodes (state worker will process this before exiting)
	nodes := m.stateGetCurrentNodes()

	// Mark only running/pending nodes as cancelled
	for _, node := range nodes {
		if node.Status == types.StatusRunning || node.Status == types.StatusPending {
			node.Status = types.StatusCancelled
			node.EndTime = now
			node.UpdatedAt = now

			// Save node with background context (ignore errors)
			_ = m.driver.SaveNode(context.Background(), m.traceID, node)

			// Broadcast cancelled event
			m.addUpdateAndBroadcast(&types.TraceUpdate{
				Type:      types.UpdateTypeNodeFailed,
				TraceID:   m.traceID,
				NodeID:    node.ID,
				Timestamp: now,
				Data: &types.NodeFailedData{
					NodeID:   node.ID,
					Status:   types.CompleteStatusCancelled,
					EndTime:  now,
					Duration: (now - node.StartTime) * 1000,
					Error:    "context cancelled",
				},
			})
		}
	}

	// Update trace status
	m.stateSetTraceStatus(types.TraceStatusCancelled)

	// Calculate total duration
	totalDuration := int64(0)
	rootNode := m.stateGetRoot()
	if rootNode != nil && rootNode.CreatedAt > 0 {
		totalDuration = (now - rootNode.CreatedAt) * 1000
	}

	// Broadcast trace cancelled event (this will be processed even after state worker starts draining)
	m.addUpdateAndBroadcast(&types.TraceUpdate{
		Type:      types.UpdateTypeComplete,
		TraceID:   m.traceID,
		Timestamp: now,
		Data: &types.TraceCompleteData{
			TraceID:       m.traceID,
			Status:        types.TraceStatusCancelled,
			TotalDuration: totalDuration,
		},
	})
}

// newNode creates a node instance that broadcasts events (for external use)
func (m *manager) newNode(data *types.TraceNode) types.Node {
	return &node{
		manager: m,
		data:    data,
	}
}

// Add creates next sequential node - auto-joins if currently in parallel state
func (m *manager) Add(input types.TraceInput, option types.TraceNodeOption) (types.Node, error) {
	if err := m.checkContext(); err != nil {
		return nil, err
	}

	now := time.Now().Unix()

	// Check if root exists
	rootNode := m.stateGetRoot()

	if rootNode == nil {
		// Create root node
		rootNode = &types.TraceNode{
			ID:              genNodeID(),
			ParentID:        "",
			Children:        []*types.TraceNode{},
			TraceNodeOption: option,
			Status:          types.StatusRunning,
			Input:           input,
			CreatedAt:       now,
			StartTime:       now,
			UpdatedAt:       now,
		}

		// Save root node
		if err := m.driver.SaveNode(m.ctx, m.traceID, rootNode); err != nil {
			return nil, fmt.Errorf("failed to save root node: %w", err)
		}

		// Update state
		m.stateUpdateRootAndCurrent(rootNode, []*types.TraceNode{rootNode})

		// Update trace status
		m.stateSetTraceStatus(types.TraceStatusRunning)

		// Broadcast event
		m.addUpdateAndBroadcast(&types.TraceUpdate{
			Type:      types.UpdateTypeNodeStart,
			TraceID:   m.traceID,
			NodeID:    rootNode.ID,
			Timestamp: now,
			Data:      rootNode.ToStartData(),
		})

		return &node{manager: m, data: rootNode}, nil
	}

	// Get current nodes
	currentNodes := m.stateGetCurrentNodes()

	var parentNode *types.TraceNode
	if len(currentNodes) > 1 {
		// Auto-join: create join node
		parentNode = &types.TraceNode{
			ID:              genNodeID(),
			ParentID:        currentNodes[0].ParentID,
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
		parentNode = currentNodes[0]
	}

	// Create new node
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

	// Update parent's children
	parentNode.Children = append(parentNode.Children, newNodeData)

	// Save nodes
	if err := m.driver.SaveNode(m.ctx, m.traceID, newNodeData); err != nil {
		return nil, err
	}
	if err := m.driver.SaveNode(m.ctx, m.traceID, parentNode); err != nil {
		return nil, err
	}

	// Update current nodes
	m.stateSetCurrentNodes([]*types.TraceNode{newNodeData})

	// Broadcast event
	m.addUpdateAndBroadcast(&types.TraceUpdate{
		Type:      types.UpdateTypeNodeStart,
		TraceID:   m.traceID,
		NodeID:    newNodeData.ID,
		Timestamp: now,
		Data:      newNodeData.ToStartData(),
	})

	return &node{manager: m, data: newNodeData}, nil
}

// Parallel creates multiple concurrent child nodes, returns Node interfaces for direct control
func (m *manager) Parallel(parallelInputs []types.TraceParallelInput) ([]types.Node, error) {
	if err := m.checkContext(); err != nil {
		return nil, err
	}

	if len(parallelInputs) == 0 {
		return nil, fmt.Errorf("parallel inputs cannot be empty")
	}

	// Check if root exists
	if m.stateGetRoot() == nil {
		return nil, fmt.Errorf("root node does not exist, please call Add first before using Parallel")
	}

	now := time.Now().Unix()

	// Get current nodes
	currentNodes := m.stateGetCurrentNodes()
	parentNode := currentNodes[0]

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

		// Create Node interface wrapper
		nodeInterfaces = append(nodeInterfaces, &node{
			manager: m,
			data:    data,
		})
	}

	// Save all nodes in batch - collect errors
	var saveErrors []error
	for _, data := range nodeData {
		if err := m.driver.SaveNode(m.ctx, m.traceID, data); err != nil {
			saveErrors = append(saveErrors, fmt.Errorf("failed to save node %s: %w", data.ID, err))
		}
	}

	// Return error if any node failed to save
	if len(saveErrors) > 0 {
		return nil, fmt.Errorf("failed to save %d node(s): %v", len(saveErrors), saveErrors)
	}

	// Save parent node
	if err := m.driver.SaveNode(m.ctx, m.traceID, parentNode); err != nil {
		return nil, fmt.Errorf("failed to save parent node: %w", err)
	}

	// Set all as current nodes
	m.stateSetCurrentNodes(nodeData)

	// Broadcast parallel nodes as batch
	m.addUpdateAndBroadcast(&types.TraceUpdate{
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

	// Get current nodes
	nodes := m.stateGetCurrentNodes()

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
		m.addUpdateAndBroadcast(&types.TraceUpdate{
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
	nodes := m.stateGetCurrentNodes()
	for _, node := range nodes {
		node.Output = output
		node.UpdatedAt = now
		if err := m.driver.SaveNode(m.ctx, m.traceID, node); err != nil {
			return err
		}

		// Broadcast node update event
		m.addUpdateAndBroadcast(&types.TraceUpdate{
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
	nodes := m.stateGetCurrentNodes()
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
		m.addUpdateAndBroadcast(&types.TraceUpdate{
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
	nodes := m.stateGetCurrentNodes()

	// Determine output value once
	var nodeOutput types.TraceOutput
	if len(output) > 0 {
		nodeOutput = output[0]
	}

	for _, node := range nodes {
		// Set output if provided
		if len(output) > 0 {
			node.Output = nodeOutput
		}

		// Create complete data BEFORE modifying other fields to avoid race
		completeData := &types.NodeCompleteData{
			NodeID:   node.ID,
			Status:   types.CompleteStatusSuccess,
			EndTime:  now,
			Duration: (now - node.StartTime) * 1000,
			Output:   node.Output,
		}

		// Now modify node status
		node.Status = types.StatusCompleted
		node.EndTime = now
		node.UpdatedAt = now
		if err := m.driver.SaveNode(m.ctx, m.traceID, node); err != nil {
			return err
		}

		// Broadcast node complete event with pre-created data
		m.addUpdateAndBroadcast(&types.TraceUpdate{
			Type:      types.UpdateTypeNodeComplete,
			TraceID:   m.traceID,
			NodeID:    node.ID,
			Timestamp: now,
			Data:      completeData,
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

	nodes := m.stateGetCurrentNodes()
	for _, node := range nodes {
		node.Status = types.StatusFailed
		node.EndTime = now
		node.UpdatedAt = now
		if saveErr := m.driver.SaveNode(m.ctx, m.traceID, node); saveErr != nil {
			return saveErr
		}

		// Broadcast node failed event
		m.addUpdateAndBroadcast(&types.TraceUpdate{
			Type:      types.UpdateTypeNodeFailed,
			TraceID:   m.traceID,
			NodeID:    node.ID,
			Timestamp: now,
			Data: &types.NodeFailedData{
				NodeID:   node.ID,
				Status:   types.CompleteStatusFailed,
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
	return m.stateGetRoot(), nil
}

// GetNode returns a node by ID
func (m *manager) GetNode(id string) (*types.TraceNode, error) {
	return m.driver.LoadNode(m.ctx, m.traceID, id)
}

// GetCurrentNodes returns current active nodes
func (m *manager) GetCurrentNodes() ([]*types.TraceNode, error) {
	return m.stateGetCurrentNodes(), nil
}

// MarkComplete marks the entire trace as completed
func (m *manager) MarkComplete() error {
	// Try to mark as completed
	if !m.stateMarkCompleted() {
		return nil // Already completed
	}

	// Update trace status
	m.stateSetTraceStatus(types.TraceStatusCompleted)

	// Calculate total duration from root node
	now := time.Now().Unix()
	totalDuration := int64(0)
	rootNode := m.stateGetRoot()
	if rootNode != nil && rootNode.CreatedAt > 0 {
		totalDuration = (now - rootNode.CreatedAt) * 1000 // Convert to milliseconds
	}

	// Broadcast completion event
	m.addUpdateAndBroadcast(&types.TraceUpdate{
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

	// Cache in memory
	m.stateSetSpace(space.ID, space)

	// Broadcast space created event
	m.addUpdateAndBroadcast(&types.TraceUpdate{
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
	// Check cache first
	if space, ok := m.stateGetSpace(id); ok {
		return space, nil
	}

	// Load from driver
	space, err := m.driver.LoadSpace(m.ctx, m.traceID, id)
	if err != nil {
		return nil, err
	}

	// Cache it
	if space != nil {
		m.stateSetSpace(id, space)
	}

	return space, nil
}

// HasSpace checks if a space exists
func (m *manager) HasSpace(id string) bool {
	// Check cache
	if _, ok := m.stateGetSpace(id); ok {
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

	// Remove from cache
	m.stateDeleteSpace(id)

	// Delete from driver
	if err := m.driver.DeleteSpace(m.ctx, m.traceID, id); err != nil {
		return err
	}

	// Broadcast space deleted event
	m.addUpdateAndBroadcast(&types.TraceUpdate{
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
		// Fallback to cached spaces
		return m.stateGetAllSpaces()
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

	// Set value in driver (through state worker for concurrent safety)
	err = m.stateExecuteSpaceOp(spaceID, func() error {
		if err := m.driver.SetSpaceKey(m.ctx, m.traceID, spaceID, key, value); err != nil {
			return err
		}

		// Update space timestamp
		space.UpdatedAt = now
		if err := m.driver.SaveSpace(m.ctx, m.traceID, space); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Broadcast memory_add event
	m.addUpdateAndBroadcast(&types.TraceUpdate{
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
	var result any
	err := m.stateExecuteSpaceOp(spaceID, func() error {
		var err error
		result, err = m.driver.GetSpaceKey(m.ctx, m.traceID, spaceID, key)
		return err
	})
	return result, err
}

// HasSpaceValue checks if a key exists in a space
func (m *manager) HasSpaceValue(spaceID, key string) bool {
	var result bool
	_ = m.stateExecuteSpaceOp(spaceID, func() error {
		result = m.driver.HasSpaceKey(m.ctx, m.traceID, spaceID, key)
		return nil
	})
	return result
}

// DeleteSpaceValue deletes a value from a space and broadcasts memory_delete event
func (m *manager) DeleteSpaceValue(spaceID, key string) error {
	if err := m.checkContext(); err != nil {
		return err
	}

	now := time.Now().Unix()

	// Delete value from driver (through state worker for concurrent safety)
	err := m.stateExecuteSpaceOp(spaceID, func() error {
		return m.driver.DeleteSpaceKey(m.ctx, m.traceID, spaceID, key)
	})

	if err != nil {
		return err
	}

	// Broadcast memory_delete event
	m.addUpdateAndBroadcast(&types.TraceUpdate{
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

	// Clear values from driver (through state worker for concurrent safety)
	err := m.stateExecuteSpaceOp(spaceID, func() error {
		return m.driver.ClearSpaceKeys(m.ctx, m.traceID, spaceID)
	})

	if err != nil {
		return err
	}

	// Broadcast memory_delete event (for all keys)
	m.addUpdateAndBroadcast(&types.TraceUpdate{
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
	var keys []string
	_ = m.stateExecuteSpaceOp(spaceID, func() error {
		var err error
		keys, err = m.driver.ListSpaceKeys(m.ctx, m.traceID, spaceID)
		return err
	})
	return keys
}

// IsComplete returns whether the trace is completed
func (m *manager) IsComplete() bool {
	return m.stateIsCompleted()
}
