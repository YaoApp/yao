package types

// Helper functions and methods to create event data

// ToStartData converts TraceNode to NodeStartData (single node)
func (n *TraceNode) ToStartData() *NodeStartData {
	return &NodeStartData{Node: n}
}

// NodesToStartData creates NodeStartData for multiple nodes (parallel operations)
func NodesToStartData(nodes []*TraceNode) *NodeStartData {
	return &NodeStartData{Nodes: nodes}
}

// ToCompleteData converts TraceNode to NodeCompleteData
func (n *TraceNode) ToCompleteData() *NodeCompleteData {
	return &NodeCompleteData{
		NodeID:   n.ID,
		Status:   CompleteStatusSuccess,
		EndTime:  n.EndTime,
		Duration: n.EndTime - n.StartTime, // Already in milliseconds
		Output:   n.Output,
	}
}

// ToFailedData converts TraceNode to NodeFailedData
func (n *TraceNode) ToFailedData(err error) *NodeFailedData {
	return &NodeFailedData{
		NodeID:   n.ID,
		Status:   CompleteStatusFailed,
		EndTime:  n.EndTime,
		Duration: n.EndTime - n.StartTime, // Already in milliseconds
		Error:    err.Error(),
	}
}

// ToMemoryAddData creates MemoryAddData for a space key-value operation
func (s *TraceSpace) ToMemoryAddData(key string, value any, timestamp int64) *MemoryAddData {
	item := MemoryItem{
		ID:        key,
		Type:      s.ID, // Space ID as type
		Content:   value,
		Timestamp: timestamp,
	}
	// Use Label as title if available
	if s.Label != "" {
		item.Title = s.Label
	}
	return &MemoryAddData{
		Type: s.ID,
		Item: item,
	}
}

// NewTraceInitData creates init event data
func NewTraceInitData(traceID string, rootNode *TraceNode, agentName ...string) *TraceInitData {
	data := &TraceInitData{
		TraceID:  traceID,
		RootNode: rootNode,
	}
	if len(agentName) > 0 {
		data.AgentName = agentName[0]
	}
	return data
}

// NewTraceCompleteData creates trace complete event data
func NewTraceCompleteData(traceID string, totalDuration int64) *TraceCompleteData {
	return &TraceCompleteData{
		TraceID:       traceID,
		Status:        TraceStatusCompleted,
		TotalDuration: totalDuration,
	}
}

// NewSpaceDeletedData creates space deleted event data
func NewSpaceDeletedData(spaceID string) *SpaceDeletedData {
	return &SpaceDeletedData{
		SpaceID: spaceID,
	}
}

// NewMemoryDeleteData creates memory delete event data (single key)
func NewMemoryDeleteData(spaceID, key string) *MemoryDeleteData {
	return &MemoryDeleteData{
		SpaceID: spaceID,
		Key:     key,
	}
}

// NewMemoryDeleteAllData creates memory delete event data (all keys cleared)
func NewMemoryDeleteAllData(spaceID string) *MemoryDeleteData {
	return &MemoryDeleteData{
		SpaceID: spaceID,
		Cleared: true,
	}
}
