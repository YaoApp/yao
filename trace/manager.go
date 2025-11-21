package trace

import (
	"context"
	"fmt"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/trace/pubsub"
	"github.com/yaoapp/yao/trace/types"
)

// manager implements the Manager interface with channel-based state management
type manager struct {
	ctx          context.Context
	cancel       context.CancelFunc
	traceID      string
	driver       types.Driver
	stateCmdChan chan stateCommand // Single channel for all state mutations
	autoArchive  bool              // Auto-archive on complete/fail
	pubsub       *pubsub.PubSub    // Reference to independent pubsub service (for publishing only, doesn't own it)
}

// NewManager creates a new trace manager instance
// pubsubService: reference to independent pubsub service (manager doesn't own it, just publishes to it)
func NewManager(ctx context.Context, traceID string, driver types.Driver, pubsubService *pubsub.PubSub, option *types.TraceOption) (types.Manager, error) {
	// Create a cancellable context for the manager
	managerCtx, cancel := context.WithCancel(ctx)

	// Determine auto-archive setting
	autoArchive := false
	if option != nil {
		autoArchive = option.AutoArchive
	}

	m := &manager{
		ctx:          managerCtx,
		cancel:       cancel,
		traceID:      traceID,
		driver:       driver,
		stateCmdChan: make(chan stateCommand, 100), // Buffered channel for performance
		autoArchive:  autoArchive,
		pubsub:       pubsubService, // Reference only, doesn't manage lifecycle
	}

	// Start state worker goroutine
	go m.startStateWorker()

	// Try to load existing updates from driver (for resumed traces)
	if existingUpdates, err := driver.LoadUpdates(ctx, traceID, 0); err == nil && len(existingUpdates) > 0 {
		log.Trace("[MANAGER] NewManager: loaded %d existing updates from driver for trace %s", len(existingUpdates), traceID)
		m.stateSetUpdates(existingUpdates)
		// Check if trace was already completed
		for _, update := range existingUpdates {
			if update.Type == types.UpdateTypeComplete {
				log.Trace("[MANAGER] NewManager: trace %s was already completed, marking as completed", traceID)
				m.stateMarkCompleted()
				if data, ok := update.Data.(*types.TraceCompleteData); ok {
					log.Trace("[MANAGER] NewManager: setting trace status to %s", data.Status)
					m.stateSetTraceStatus(data.Status)
				}
				break
			}
		}
	} else {
		if err != nil {
			log.Trace("[MANAGER] NewManager: failed to load existing updates for trace %s: %v", traceID, err)
		} else {
			log.Trace("[MANAGER] NewManager: no existing updates found for trace %s, creating new trace", traceID)
		}
		// New trace - create and broadcast init event
		now := time.Now().UnixMilli()
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

// addUpdateAndBroadcast persists, adds to history, and publishes an update
func (m *manager) addUpdateAndBroadcast(update *types.TraceUpdate) {
	// Persist to driver (synchronous - no race)
	if err := m.driver.SaveUpdate(context.Background(), m.traceID, update); err != nil {
		log.Trace("[MANAGER] addUpdateAndBroadcast: failed to save update type=%s for trace %s: %v", update.Type, m.traceID, err)
	} else {
		log.Trace("[MANAGER] addUpdateAndBroadcast: successfully saved update type=%s for trace %s", update.Type, m.traceID)
	}

	// Add to in-memory history
	m.stateAddUpdate(update)

	// Publish to independent PubSub service (manager just publishes, doesn't manage pubsub lifecycle)
	if m.pubsub != nil {
		m.pubsub.Publish(update)
	}
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

// Add creates next sequential node - auto-joins if currently in parallel state
func (m *manager) Add(input types.TraceInput, option types.TraceNodeOption) (types.Node, error) {
	if err := m.checkContext(); err != nil {
		return nil, err
	}

	now := time.Now().UnixMilli()

	// Check if root exists
	rootNode := m.stateGetRoot()

	if rootNode == nil {
		// Create root node
		rootNode = &types.TraceNode{
			ID:              genNodeID(),
			ParentIDs:       []string{}, // Root has no parents
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

	// Auto-complete parent nodes if enabled (default: true when nil)
	autoCompleteParent := option.AutoCompleteParent == nil || *option.AutoCompleteParent
	if autoCompleteParent {
		for _, current := range currentNodes {
			// Only auto-complete running or pending nodes
			if current.Status == types.StatusRunning || current.Status == types.StatusPending {
				// Use node.Complete() method to properly complete the node with broadcast
				currentNodeInterface := &node{manager: m, data: current}
				if err := currentNodeInterface.Complete(); err != nil {
					// Log error but don't fail the operation
					m.Error("Failed to auto-complete parent node %s: %v", current.ID, err)
				}
			}
		}
	}

	// Collect parent IDs from all current nodes (supports implicit join)
	parentIDs := make([]string, 0, len(currentNodes))
	for _, current := range currentNodes {
		parentIDs = append(parentIDs, current.ID)
	}

	// Create new node with multiple parents (implicit join)
	newNodeData := &types.TraceNode{
		ID:              genNodeID(),
		ParentIDs:       parentIDs, // Multiple parents for implicit join
		Children:        []*types.TraceNode{},
		TraceNodeOption: option,
		Status:          types.StatusRunning,
		Input:           input,
		CreatedAt:       now,
		StartTime:       now,
		UpdatedAt:       now,
	}

	// Add to each parent's children
	for _, parent := range currentNodes {
		parent.Children = append(parent.Children, newNodeData)
		if err := m.driver.SaveNode(m.ctx, m.traceID, parent); err != nil {
			// Log error but continue
			m.Error("Failed to update parent node %s: %v", parent.ID, err)
		}
	}

	// Save new node
	if err := m.driver.SaveNode(m.ctx, m.traceID, newNodeData); err != nil {
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

	now := time.Now().UnixMilli()

	// Get current nodes
	currentNodes := m.stateGetCurrentNodes()

	// Auto-complete parent node if any option has AutoCompleteParent enabled (default: true when nil)
	shouldAutoComplete := false
	for _, input := range parallelInputs {
		if input.Option.AutoCompleteParent == nil || *input.Option.AutoCompleteParent {
			shouldAutoComplete = true
			break
		}
	}

	if shouldAutoComplete {
		for _, current := range currentNodes {
			// Only auto-complete running or pending nodes
			if current.Status == types.StatusRunning || current.Status == types.StatusPending {
				// Use node.Complete() method to properly complete the node with broadcast
				currentNodeInterface := &node{manager: m, data: current}
				if err := currentNodeInterface.Complete(); err != nil {
					// Log error but don't fail the operation
					m.Error("Failed to auto-complete parent node %s: %v", current.ID, err)
				}
			}
		}
	}

	parentNode := currentNodes[0]

	nodeData := make([]*types.TraceNode, 0, len(parallelInputs))
	nodeInterfaces := make([]types.Node, 0, len(parallelInputs))

	// Create multiple child nodes with single parent
	for _, input := range parallelInputs {
		data := &types.TraceNode{
			ID:              genNodeID(),
			ParentIDs:       []string{parentNode.ID}, // Single parent for parallel branches
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
func (m *manager) Info(message string, args ...any) types.Manager {
	m.log("info", message, args...)
	return m
}

// Debug logs debug message to current node(s)
func (m *manager) Debug(message string, args ...any) types.Manager {
	m.log("debug", message, args...)
	return m
}

// Error logs error message to current node(s)
func (m *manager) Error(message string, args ...any) types.Manager {
	m.log("error", message, args...)
	return m
}

// Warn logs warning message to current node(s)
func (m *manager) Warn(message string, args ...any) types.Manager {
	m.log("warn", message, args...)
	return m
}

// log helper method to log messages
func (m *manager) log(level string, message string, args ...any) {
	now := time.Now().UnixMilli()

	// Get current nodes
	nodes := m.stateGetCurrentNodes()

	// Log to all current nodes
	for _, node := range nodes {
		log := &types.TraceLog{
			Timestamp: now,
			Level:     level,
			Message:   message,
			Data:      args,
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

	now := time.Now().UnixMilli()
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

	now := time.Now().UnixMilli()
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

	now := time.Now().UnixMilli()
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
			Duration: now - node.StartTime, // Already in milliseconds
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

	now := time.Now().UnixMilli()
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
				Duration: node.EndTime - node.StartTime, // Already in milliseconds
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
	now := time.Now().UnixMilli()
	totalDuration := int64(0)
	rootNode := m.stateGetRoot()
	if rootNode != nil && rootNode.CreatedAt > 0 {
		totalDuration = now - rootNode.CreatedAt // Already in milliseconds
	}

	// Broadcast completion event
	m.addUpdateAndBroadcast(&types.TraceUpdate{
		Type:      types.UpdateTypeComplete,
		TraceID:   m.traceID,
		Timestamp: now,
		Data:      types.NewTraceCompleteData(m.traceID, totalDuration),
	})

	// Auto-archive if enabled
	if m.autoArchive {
		if err := m.driver.Archive(m.ctx, m.traceID); err != nil {
			// Log error but don't fail the complete operation
			m.Debug("Failed to auto-archive trace", map[string]any{
				"trace_id": m.traceID,
				"error":    err.Error(),
			})
		}
	}

	return nil
}

// CreateSpace creates a new memory space
func (m *manager) CreateSpace(option types.TraceSpaceOption) (*types.TraceSpace, error) {
	if err := m.checkContext(); err != nil {
		return nil, err
	}

	now := time.Now().UnixMilli()

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

	now := time.Now().UnixMilli()

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

	now := time.Now().UnixMilli()

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

	now := time.Now().UnixMilli()

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

	now := time.Now().UnixMilli()

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

// GetEvents retrieves all events since a specific timestamp
// since=0 returns all events from the beginning
func (m *manager) GetEvents(since int64) ([]*types.TraceUpdate, error) {
	if err := m.checkContext(); err != nil {
		return nil, err
	}
	return m.stateGetUpdates(since), nil
}

// GetTraceInfo retrieves the trace info from storage
func (m *manager) GetTraceInfo() (*types.TraceInfo, error) {
	if err := m.checkContext(); err != nil {
		return nil, err
	}
	return m.driver.LoadTraceInfo(m.ctx, m.traceID)
}

// GetAllNodes retrieves all nodes from storage
func (m *manager) GetAllNodes() ([]*types.TraceNode, error) {
	if err := m.checkContext(); err != nil {
		return nil, err
	}

	// Load the root node tree from storage
	rootNode, err := m.driver.LoadTrace(m.ctx, m.traceID)
	if err != nil {
		return nil, err
	}

	if rootNode == nil {
		return []*types.TraceNode{}, nil
	}

	// Flatten the tree to get all nodes
	var allNodes []*types.TraceNode
	var collectNodes func(*types.TraceNode)
	collectNodes = func(node *types.TraceNode) {
		if node == nil {
			return
		}
		allNodes = append(allNodes, node)
		for _, child := range node.Children {
			collectNodes(child)
		}
	}
	collectNodes(rootNode)

	return allNodes, nil
}

// GetNodeByID retrieves a specific node by ID from storage
func (m *manager) GetNodeByID(nodeID string) (*types.TraceNode, error) {
	if err := m.checkContext(); err != nil {
		return nil, err
	}
	return m.driver.LoadNode(m.ctx, m.traceID, nodeID)
}

// GetAllLogs retrieves all logs from storage
func (m *manager) GetAllLogs() ([]*types.TraceLog, error) {
	if err := m.checkContext(); err != nil {
		return nil, err
	}
	return m.driver.LoadLogs(m.ctx, m.traceID, "")
}

// GetLogsByNode retrieves logs for a specific node from storage
func (m *manager) GetLogsByNode(nodeID string) ([]*types.TraceLog, error) {
	if err := m.checkContext(); err != nil {
		return nil, err
	}
	return m.driver.LoadLogs(m.ctx, m.traceID, nodeID)
}

// GetAllSpaces retrieves all spaces from storage
func (m *manager) GetAllSpaces() ([]*types.TraceSpace, error) {
	if err := m.checkContext(); err != nil {
		return nil, err
	}

	// Get all space IDs from driver
	spaceIDs, err := m.driver.ListSpaces(m.ctx, m.traceID)
	if err != nil {
		return nil, err
	}

	// Load all spaces
	spaces := make([]*types.TraceSpace, 0, len(spaceIDs))
	for _, spaceID := range spaceIDs {
		space, err := m.driver.LoadSpace(m.ctx, m.traceID, spaceID)
		if err != nil {
			continue // Skip spaces that fail to load
		}
		if space != nil {
			spaces = append(spaces, space)
		}
	}

	return spaces, nil
}

// GetSpaceByID retrieves a specific space by ID from storage with all its key-value data
func (m *manager) GetSpaceByID(spaceID string) (*types.TraceSpaceData, error) {
	if err := m.checkContext(); err != nil {
		return nil, err
	}

	// Load space metadata
	space, err := m.driver.LoadSpace(m.ctx, m.traceID, spaceID)
	if err != nil {
		return nil, err
	}
	if space == nil {
		return nil, nil
	}

	// Load all keys in the space
	keys, err := m.driver.ListSpaceKeys(m.ctx, m.traceID, spaceID)
	if err != nil {
		return nil, err
	}

	// Load all key-value pairs
	data := make(map[string]any)
	for _, key := range keys {
		value, err := m.driver.GetSpaceKey(m.ctx, m.traceID, spaceID, key)
		if err != nil {
			continue // Skip keys that fail to load
		}
		data[key] = value
	}

	return &types.TraceSpaceData{
		TraceSpace: *space,
		Data:       data,
	}, nil
}
