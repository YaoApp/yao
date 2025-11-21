package trace

import (
	"time"

	"github.com/yaoapp/yao/trace/types"
)

// node implements the Node interface for custom node operations
type node struct {
	manager *manager
	data    *types.TraceNode
}

// Info logs info message (public method, broadcasts event)
func (n *node) Info(message string, args ...any) types.Node {
	n.logWithBroadcast("info", message, args...)
	return n
}

// Debug logs debug message (public method, broadcasts event)
func (n *node) Debug(message string, args ...any) types.Node {
	n.logWithBroadcast("debug", message, args...)
	return n
}

// Error logs error message (public method, broadcasts event)
func (n *node) Error(message string, args ...any) types.Node {
	n.logWithBroadcast("error", message, args...)
	return n
}

// Warn logs warning message (public method, broadcasts event)
func (n *node) Warn(message string, args ...any) types.Node {
	n.logWithBroadcast("warn", message, args...)
	return n
}

// logWithBroadcast logs and broadcasts event (for external calls)
func (n *node) logWithBroadcast(level string, message string, args ...any) {
	log := n.log(level, message, args...)

	// Broadcast event
	n.manager.addUpdateAndBroadcast(&types.TraceUpdate{
		Type:      types.UpdateTypeLogAdded,
		TraceID:   n.manager.traceID,
		NodeID:    n.data.ID,
		Timestamp: log.Timestamp,
		Data:      log,
	})
}

// log logs without broadcasting (for internal Manager calls)
func (n *node) log(level string, message string, args ...any) *types.TraceLog {
	log := &types.TraceLog{
		Timestamp: time.Now().UnixMilli(),
		Level:     level,
		Message:   message,
		Data:      args,
		NodeID:    n.data.ID,
	}
	// Save log (ignore errors for non-critical logging)
	_ = n.manager.driver.SaveLog(n.manager.ctx, n.manager.traceID, log)
	return log
}

// Add creates next sequential node
func (n *node) Add(input types.TraceInput, option types.TraceNodeOption) (types.Node, error) {
	now := time.Now().UnixMilli()

	// Create child node data
	childNodeData := &types.TraceNode{
		ID:              genNodeID(),
		ParentIDs:       []string{n.data.ID}, // Single parent
		Children:        []*types.TraceNode{},
		TraceNodeOption: option,
		Status:          types.StatusRunning,
		Input:           input,
		CreatedAt:       now,
		StartTime:       now,
		UpdatedAt:       now,
	}

	// Add to parent's children
	n.data.Children = append(n.data.Children, childNodeData)

	// Save both nodes
	if err := n.manager.driver.SaveNode(n.manager.ctx, n.manager.traceID, childNodeData); err != nil {
		return nil, err
	}
	if err := n.manager.driver.SaveNode(n.manager.ctx, n.manager.traceID, n.data); err != nil {
		return nil, err
	}

	// Return Node interface
	return &node{
		manager: n.manager,
		data:    childNodeData,
	}, nil
}

// Parallel creates multiple concurrent child nodes
func (n *node) Parallel(parallelInputs []types.TraceParallelInput) ([]types.Node, error) {
	now := time.Now().UnixMilli()
	nodeInterfaces := make([]types.Node, 0, len(parallelInputs))

	// Create multiple child nodes
	for _, input := range parallelInputs {
		childNodeData := &types.TraceNode{
			ID:              genNodeID(),
			ParentIDs:       []string{n.data.ID}, // Single parent for parallel branches
			Children:        []*types.TraceNode{},
			TraceNodeOption: input.Option,
			Status:          types.StatusRunning,
			Input:           input.Input,
			CreatedAt:       now,
			StartTime:       now,
			UpdatedAt:       now,
		}
		n.data.Children = append(n.data.Children, childNodeData)

		// Save node
		if err := n.manager.driver.SaveNode(n.manager.ctx, n.manager.traceID, childNodeData); err != nil {
			return nil, err
		}

		// Create Node interface wrapper
		nodeInterfaces = append(nodeInterfaces, &node{
			manager: n.manager,
			data:    childNodeData,
		})
	}

	// Save parent node
	if err := n.manager.driver.SaveNode(n.manager.ctx, n.manager.traceID, n.data); err != nil {
		return nil, err
	}

	return nodeInterfaces, nil
}

// Join joins multiple nodes into one
func (n *node) Join(nodes []*types.TraceNode, input types.TraceInput, option types.TraceNodeOption) (types.Node, error) {
	now := time.Now().UnixMilli()

	// Collect parent IDs from all nodes
	parentIDs := make([]string, 0, len(nodes))
	for _, node := range nodes {
		if node != nil {
			parentIDs = append(parentIDs, node.ID)
		}
	}

	// Create join node data with multiple parents
	joinNodeData := &types.TraceNode{
		ID:              genNodeID(),
		ParentIDs:       parentIDs, // Multiple parents for explicit join
		Children:        []*types.TraceNode{},
		TraceNodeOption: option,
		Status:          types.StatusRunning,
		Input:           input,
		CreatedAt:       now,
		StartTime:       now,
		UpdatedAt:       now,
	}

	// Save join node
	if err := n.manager.driver.SaveNode(n.manager.ctx, n.manager.traceID, joinNodeData); err != nil {
		return nil, err
	}

	// Return Node interface
	return &node{
		manager: n.manager,
		data:    joinNodeData,
	}, nil
}

// ID returns the node ID
func (n *node) ID() string {
	return n.data.ID
}

// SetOutput sets the node output
func (n *node) SetOutput(output types.TraceOutput) error {
	n.data.Output = output
	n.data.UpdatedAt = time.Now().UnixMilli()
	return n.manager.driver.SaveNode(n.manager.ctx, n.manager.traceID, n.data)
}

// SetMetadata sets node metadata
func (n *node) SetMetadata(key string, value any) error {
	if n.data.Metadata == nil {
		n.data.Metadata = make(map[string]any)
	}
	n.data.Metadata[key] = value
	n.data.UpdatedAt = time.Now().UnixMilli()
	return n.manager.driver.SaveNode(n.manager.ctx, n.manager.traceID, n.data)
}

// SetStatus sets the node status
func (n *node) SetStatus(status string) error {
	n.data.Status = types.NodeStatus(status)
	n.data.UpdatedAt = time.Now().UnixMilli()
	return n.manager.driver.SaveNode(n.manager.ctx, n.manager.traceID, n.data)
}

// Complete marks the node as completed (public method, broadcasts event)
// Optional output parameter: if provided, sets the output before completing
func (n *node) Complete(output ...types.TraceOutput) error {
	if err := n.complete(output...); err != nil {
		return err
	}

	// Broadcast event
	n.manager.addUpdateAndBroadcast(&types.TraceUpdate{
		Type:      types.UpdateTypeNodeComplete,
		TraceID:   n.manager.traceID,
		NodeID:    n.data.ID,
		Timestamp: n.data.EndTime,
		Data:      n.data.ToCompleteData(),
	})

	return nil
}

// complete marks as completed without broadcasting (for Manager calls)
func (n *node) complete(output ...types.TraceOutput) error {
	now := time.Now().UnixMilli()

	// Set output if provided
	if len(output) > 0 {
		n.data.Output = output[0]
	}

	n.data.Status = types.StatusCompleted
	n.data.EndTime = now
	n.data.UpdatedAt = now
	return n.manager.driver.SaveNode(n.manager.ctx, n.manager.traceID, n.data)
}

// Fail marks the node as failed (public method, broadcasts event)
func (n *node) Fail(err error) error {
	// Log error first
	n.Error("Node failed: %v", err)

	if saveErr := n.fail(err); saveErr != nil {
		return saveErr
	}

	// Broadcast event
	n.manager.addUpdateAndBroadcast(&types.TraceUpdate{
		Type:      types.UpdateTypeNodeFailed,
		TraceID:   n.manager.traceID,
		NodeID:    n.data.ID,
		Timestamp: n.data.EndTime,
		Data:      n.data.ToFailedData(err),
	})

	return nil
}

// fail marks as failed without broadcasting (for Manager calls)
func (n *node) fail(err error) error {
	now := time.Now().UnixMilli()

	// Update status
	n.data.Status = types.StatusFailed
	n.data.EndTime = now
	n.data.UpdatedAt = now

	return n.manager.driver.SaveNode(n.manager.ctx, n.manager.traceID, n.data)
}
