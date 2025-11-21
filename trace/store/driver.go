package store

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/trace/types"
)

// persistNode is a lightweight version of TraceNode for storage
// Only stores IDs of children instead of full child nodes
type persistNode struct {
	ID          string            `json:"ID"`
	ParentIDs   []string          `json:"ParentIDs,omitempty"`
	ChildrenIDs []string          `json:"ChildrenIDs,omitempty"`
	Label       string            `json:"Label,omitempty"`
	Type        string            `json:"Type,omitempty"`
	Icon        string            `json:"Icon,omitempty"`
	Description string            `json:"Description,omitempty"`
	Metadata    map[string]any    `json:"Metadata,omitempty"`
	Status      types.NodeStatus  `json:"Status"`
	Input       types.TraceInput  `json:"Input,omitempty"`
	Output      types.TraceOutput `json:"Output,omitempty"`
	CreatedAt   int64             `json:"CreatedAt"`
	StartTime   int64             `json:"StartTime"`
	EndTime     int64             `json:"EndTime,omitempty"`
	UpdatedAt   int64             `json:"UpdatedAt"`
}

// toPersistNode converts TraceNode to persistNode for storage
func toPersistNode(node *types.TraceNode) *persistNode {
	if node == nil {
		return nil
	}

	// Extract children IDs
	childrenIDs := make([]string, 0, len(node.Children))
	for _, child := range node.Children {
		if child != nil {
			childrenIDs = append(childrenIDs, child.ID)
		}
	}

	return &persistNode{
		ID:          node.ID,
		ParentIDs:   node.ParentIDs,
		ChildrenIDs: childrenIDs,
		Label:       node.Label,
		Type:        node.Type,
		Icon:        node.Icon,
		Description: node.Description,
		Metadata:    node.Metadata,
		Status:      node.Status,
		Input:       node.Input,
		Output:      node.Output,
		CreatedAt:   node.CreatedAt,
		StartTime:   node.StartTime,
		EndTime:     node.EndTime,
		UpdatedAt:   node.UpdatedAt,
	}
}

// fromPersistNode converts persistNode to TraceNode
func fromPersistNode(pn *persistNode) *types.TraceNode {
	if pn == nil {
		return nil
	}

	return &types.TraceNode{
		ID:        pn.ID,
		ParentIDs: pn.ParentIDs,
		Children:  nil, // Children will be loaded separately if needed
		TraceNodeOption: types.TraceNodeOption{
			Label:       pn.Label,
			Type:        pn.Type,
			Icon:        pn.Icon,
			Description: pn.Description,
			Metadata:    pn.Metadata,
		},
		Status:    pn.Status,
		Input:     pn.Input,
		Output:    pn.Output,
		CreatedAt: pn.CreatedAt,
		StartTime: pn.StartTime,
		EndTime:   pn.EndTime,
		UpdatedAt: pn.UpdatedAt,
	}
}

// Driver the gou store storage driver implementation
type Driver struct {
	storeName string      // Store name in gou
	store     store.Store // Gou store instance
	prefix    string      // Key prefix for isolation
	updatesMu sync.Mutex  // Protects concurrent updates
}

// New creates a new store driver
// storeName: the name of the store to use
// prefix: optional key prefix for isolation (default: "__trace")
func New(storeName string, prefix ...string) (*Driver, error) {
	// Get store instance from gou
	st, err := store.Get(storeName)
	if err != nil {
		return nil, fmt.Errorf("failed to get store %s: %w", storeName, err)
	}

	// Set default prefix if not provided
	keyPrefix := "__trace"
	if len(prefix) > 0 && prefix[0] != "" {
		keyPrefix = prefix[0]
	}

	return &Driver{
		storeName: storeName,
		store:     st,
		prefix:    keyPrefix,
	}, nil
}

// getKey generates a key for storage with configurable prefix
// Format: {prefix}:{traceID}:{type}:{id}
// The prefix ensures isolation from other data in shared store
func (d *Driver) getKey(traceID string, parts ...string) string {
	allParts := append([]string{d.prefix, traceID}, parts...)
	return strings.Join(allParts, ":")
}

// getKeyPrefix returns the prefix for all keys of a trace
func (d *Driver) getKeyPrefix(traceID string) string {
	return d.prefix + ":" + traceID + ":"
}

func (d *Driver) getNodeKeyPrefix(traceID string) string {
	return d.getKey(traceID, "node") + ":"
}

// getTraceInfoKey returns the key for trace info
func (d *Driver) getTraceInfoKey(traceID string) string {
	return d.getKey(traceID, "info")
}

// getUpdatesKey returns the key for trace updates
func (d *Driver) getUpdatesKey(traceID string) string {
	return d.getKey(traceID, "updates")
}

// SaveNode persists a node to store
func (d *Driver) SaveNode(ctx context.Context, traceID string, node *types.TraceNode) error {
	// Check if archived - archived traces are read-only
	archived, err := d.IsArchived(ctx, traceID)
	if err != nil {
		return fmt.Errorf("failed to check archive status: %w", err)
	}
	if archived {
		return fmt.Errorf("cannot save node: trace %s is archived (read-only)", traceID)
	}

	key := d.getKey(traceID, "node", node.ID)

	// Convert to persist format (only store children IDs)
	persistData := toPersistNode(node)

	data, err := json.Marshal(persistData)
	if err != nil {
		return fmt.Errorf("failed to marshal node: %w", err)
	}

	if err := d.store.Set(key, string(data), 0); err != nil {
		return fmt.Errorf("failed to save node to store: %w", err)
	}

	return nil
}

// LoadNode loads a node from store
func (d *Driver) LoadNode(ctx context.Context, traceID string, nodeID string) (*types.TraceNode, error) {
	// Check if archived and extract if needed
	archived, err := d.IsArchived(ctx, traceID)
	if err != nil {
		return nil, fmt.Errorf("failed to check archive status: %w", err)
	}
	if archived {
		if err := d.unarchive(ctx, traceID); err != nil {
			return nil, fmt.Errorf("failed to unarchive trace: %w", err)
		}
	}

	key := d.getKey(traceID, "node", nodeID)

	value, ok := d.store.Get(key)
	if !ok {
		return nil, nil
	}

	dataStr, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("invalid data type in store")
	}

	var pn persistNode
	if err := json.Unmarshal([]byte(dataStr), &pn); err != nil {
		return nil, fmt.Errorf("failed to unmarshal node: %w", err)
	}

	// Convert to TraceNode
	node := fromPersistNode(&pn)

	// Load children if needed
	if len(pn.ChildrenIDs) > 0 {
		children := make([]*types.TraceNode, 0, len(pn.ChildrenIDs))
		for _, childID := range pn.ChildrenIDs {
			child, err := d.LoadNode(ctx, traceID, childID)
			if err != nil {
				return nil, fmt.Errorf("failed to load child node %s: %w", childID, err)
			}
			if child != nil {
				children = append(children, child)
			}
		}
		node.Children = children
	}

	return node, nil
}

// LoadTrace loads the entire trace tree from store
func (d *Driver) LoadTrace(ctx context.Context, traceID string) (*types.TraceNode, error) {
	// Check if archived and extract if needed
	archived, err := d.IsArchived(ctx, traceID)
	if err != nil {
		return nil, fmt.Errorf("failed to check archive status: %w", err)
	}
	if archived {
		if err := d.unarchive(ctx, traceID); err != nil {
			return nil, fmt.Errorf("failed to unarchive trace: %w", err)
		}
	}

	// List all node keys
	nodePrefix := d.getNodeKeyPrefix(traceID)
	nodeKeys, err := d.listKeysByPrefix(ctx, nodePrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list node keys: %w", err)
	}

	if len(nodeKeys) == 0 {
		return nil, nil
	}

	// Find root node ID (node with empty ParentIDs) by checking each node
	var rootNodeID string
	for _, key := range nodeKeys {
		// Extract node ID from key (format: prefix:traceID:nodes:nodeID)
		parts := strings.Split(key, ":")
		if len(parts) < 4 {
			continue
		}
		nodeID := parts[len(parts)-1]

		// Read node data to check if it's root
		data, exists := d.store.Get(key)
		if !exists {
			continue
		}

		dataStr, ok := data.(string)
		if !ok {
			continue
		}

		var pn persistNode
		if err := json.Unmarshal([]byte(dataStr), &pn); err != nil {
			continue
		}

		if len(pn.ParentIDs) == 0 {
			rootNodeID = nodeID
			break
		}
	}

	if rootNodeID == "" {
		return nil, fmt.Errorf("no root node found in trace")
	}

	// Load root node (this will recursively load all children)
	return d.LoadNode(ctx, traceID, rootNodeID)
}

// SaveSpace persists a space to store
func (d *Driver) SaveSpace(ctx context.Context, traceID string, space *types.TraceSpace) error {
	// Check if archived - archived traces are read-only
	archived, err := d.IsArchived(ctx, traceID)
	if err != nil {
		return fmt.Errorf("failed to check archive status: %w", err)
	}
	if archived {
		return fmt.Errorf("cannot save space: trace %s is archived (read-only)", traceID)
	}

	key := d.getKey(traceID, "space", space.ID)

	data, err := json.Marshal(space)
	if err != nil {
		return fmt.Errorf("failed to marshal space: %w", err)
	}

	if err := d.store.Set(key, string(data), 0); err != nil {
		return fmt.Errorf("failed to save space to store: %w", err)
	}

	return nil
}

// LoadSpace loads a space from store
func (d *Driver) LoadSpace(ctx context.Context, traceID string, spaceID string) (*types.TraceSpace, error) {
	// Check if archived and extract if needed
	archived, err := d.IsArchived(ctx, traceID)
	if err != nil {
		return nil, fmt.Errorf("failed to check archive status: %w", err)
	}
	if archived {
		if err := d.unarchive(ctx, traceID); err != nil {
			return nil, fmt.Errorf("failed to unarchive trace: %w", err)
		}
	}

	key := d.getKey(traceID, "space", spaceID)

	value, ok := d.store.Get(key)
	if !ok {
		return nil, nil
	}

	dataStr, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("invalid data type in store")
	}

	var space types.TraceSpace
	if err := json.Unmarshal([]byte(dataStr), &space); err != nil {
		return nil, fmt.Errorf("failed to unmarshal space: %w", err)
	}

	return &space, nil
}

// DeleteSpace removes a space from store
func (d *Driver) DeleteSpace(ctx context.Context, traceID string, spaceID string) error {
	// Delete space metadata
	key := d.getKey(traceID, "space", spaceID)
	if err := d.store.Del(key); err != nil {
		return fmt.Errorf("failed to delete space from store: %w", err)
	}

	// Delete space data (all keys)
	dataKey := d.getKey(traceID, "space", spaceID, "data")
	_ = d.store.Del(dataKey) // Ignore error if not exists

	return nil
}

// ListSpaces lists all space IDs for a trace from store
func (d *Driver) ListSpaces(ctx context.Context, traceID string) ([]string, error) {
	// Get all keys from store
	allKeys := d.store.Keys()

	// Filter keys matching pattern: {prefix}:{traceID}:space:*
	prefix := d.getKey(traceID, "space", "")
	spaceIDs := make([]string, 0)

	for _, key := range allKeys {
		if strings.HasPrefix(key, prefix) {
			parts := strings.Split(key, ":")
			// Count parts to find space metadata key
			// Format: {prefix}:{traceID}:space:{spaceID}
			// Parts count depends on prefix (e.g., "__trace" = 4 parts total + 1 = 5)
			expectedParts := strings.Count(d.prefix, ":") + 4
			if len(parts) == expectedParts {
				// This is a space metadata key (not a data key)
				spaceID := parts[len(parts)-1]
				spaceIDs = append(spaceIDs, spaceID)
			}
		}
	}

	return spaceIDs, nil
}

// getSpaceDataKey returns the key for space data storage
func (d *Driver) getSpaceDataKey(traceID, spaceID string) string {
	return d.getKey(traceID, "space", spaceID, "data")
}

// loadSpaceData loads all key-value pairs for a space
func (d *Driver) loadSpaceData(traceID, spaceID string) (map[string]any, error) {
	key := d.getSpaceDataKey(traceID, spaceID)

	value, ok := d.store.Get(key)
	if !ok {
		return make(map[string]any), nil
	}

	dataStr, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("invalid data type in store")
	}

	var kvData map[string]any
	if err := json.Unmarshal([]byte(dataStr), &kvData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal space data: %w", err)
	}

	return kvData, nil
}

// saveSpaceData saves all key-value pairs for a space
func (d *Driver) saveSpaceData(traceID, spaceID string, kvData map[string]any) error {
	key := d.getSpaceDataKey(traceID, spaceID)

	data, err := json.Marshal(kvData)
	if err != nil {
		return fmt.Errorf("failed to marshal space data: %w", err)
	}

	if err := d.store.Set(key, string(data), 0); err != nil {
		return fmt.Errorf("failed to save space data: %w", err)
	}

	return nil
}

// SetSpaceKey stores a value by key in a space
func (d *Driver) SetSpaceKey(ctx context.Context, traceID, spaceID, key string, value any) error {
	// Load existing data
	kvData, err := d.loadSpaceData(traceID, spaceID)
	if err != nil {
		return err
	}

	// Set new value
	kvData[key] = value

	// Save data
	return d.saveSpaceData(traceID, spaceID, kvData)
}

// GetSpaceKey retrieves a value by key from a space
func (d *Driver) GetSpaceKey(ctx context.Context, traceID, spaceID, key string) (any, error) {
	kvData, err := d.loadSpaceData(traceID, spaceID)
	if err != nil {
		return nil, err
	}

	value, exists := kvData[key]
	if !exists {
		return nil, nil
	}

	return value, nil
}

// HasSpaceKey checks if a key exists in a space
func (d *Driver) HasSpaceKey(ctx context.Context, traceID, spaceID, key string) bool {
	kvData, err := d.loadSpaceData(traceID, spaceID)
	if err != nil {
		return false
	}

	_, exists := kvData[key]
	return exists
}

// DeleteSpaceKey removes a key-value pair from a space
func (d *Driver) DeleteSpaceKey(ctx context.Context, traceID, spaceID, key string) error {
	kvData, err := d.loadSpaceData(traceID, spaceID)
	if err != nil {
		return err
	}

	delete(kvData, key)

	return d.saveSpaceData(traceID, spaceID, kvData)
}

// ClearSpaceKeys removes all key-value pairs from a space
func (d *Driver) ClearSpaceKeys(ctx context.Context, traceID, spaceID string) error {
	return d.saveSpaceData(traceID, spaceID, make(map[string]any))
}

// ListSpaceKeys returns all keys in a space
func (d *Driver) ListSpaceKeys(ctx context.Context, traceID, spaceID string) ([]string, error) {
	kvData, err := d.loadSpaceData(traceID, spaceID)
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(kvData))
	for key := range kvData {
		keys = append(keys, key)
	}

	return keys, nil
}

// SaveLog appends a log entry to store
func (d *Driver) SaveLog(ctx context.Context, traceID string, log *types.TraceLog) error {
	// Check if archived - archived traces are read-only
	archived, err := d.IsArchived(ctx, traceID)
	if err != nil {
		return fmt.Errorf("failed to check archive status: %w", err)
	}
	if archived {
		return fmt.Errorf("cannot save log: trace %s is archived (read-only)", traceID)
	}

	// Store logs using ArraySlice approach (store as array in a key)
	key := d.getKey(traceID, "logs", log.NodeID)

	// Marshal log
	data, err := json.Marshal(log)
	if err != nil {
		return fmt.Errorf("failed to marshal log: %w", err)
	}

	// Append to array using Push
	if err := d.store.Push(key, string(data)); err != nil {
		return fmt.Errorf("failed to append log to store: %w", err)
	}

	return nil
}

// LoadLogs loads all logs for a trace or specific node from store
func (d *Driver) LoadLogs(ctx context.Context, traceID string, nodeID string) ([]*types.TraceLog, error) {
	// Check if archived and extract if needed
	archived, err := d.IsArchived(ctx, traceID)
	if err != nil {
		return nil, fmt.Errorf("failed to check archive status: %w", err)
	}
	if archived {
		if err := d.unarchive(ctx, traceID); err != nil {
			return nil, fmt.Errorf("failed to unarchive trace: %w", err)
		}
	}

	var logs []*types.TraceLog

	if nodeID != "" {
		// Load logs for specific node
		key := d.getKey(traceID, "logs", nodeID)
		nodeLogs, err := d.loadLogsFromKey(key)
		if err != nil {
			return nil, err
		}
		logs = append(logs, nodeLogs...)
	} else {
		// Load all logs by iterating all keys
		// Pattern: {prefix}:{traceID}:logs:*
		allKeys := d.store.Keys()
		prefix := d.getKey(traceID, "logs", "")

		for _, key := range allKeys {
			if strings.HasPrefix(key, prefix) {
				nodeLogs, err := d.loadLogsFromKey(key)
				if err != nil {
					return nil, err
				}
				logs = append(logs, nodeLogs...)
			}
		}
	}

	return logs, nil
}

// loadLogsFromKey loads logs from a specific key (array)
func (d *Driver) loadLogsFromKey(key string) ([]*types.TraceLog, error) {
	// Get all items from array
	items, err := d.store.ArrayAll(key)
	if err != nil {
		return []*types.TraceLog{}, nil
	}

	logs := make([]*types.TraceLog, 0, len(items))
	for _, item := range items {
		itemStr, ok := item.(string)
		if !ok {
			continue
		}

		var log types.TraceLog
		if err := json.Unmarshal([]byte(itemStr), &log); err != nil {
			// Skip malformed entries
			continue
		}
		logs = append(logs, &log)
	}

	return logs, nil
}

// SaveTraceInfo persists trace metadata to store
func (d *Driver) SaveTraceInfo(ctx context.Context, info *types.TraceInfo) error {
	// Allow saving trace info even if archived (for updating archive status)
	if info.Archived {
		// If already archived, only allow updating archive-related fields
		existing, err := d.LoadTraceInfo(ctx, info.ID)
		if err == nil && existing != nil && existing.Archived && !info.Archived {
			return fmt.Errorf("cannot unarchive trace: trace %s is archived", info.ID)
		}
	}

	key := d.getKey(info.ID, "info")

	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal trace info: %w", err)
	}

	if err := d.store.Set(key, string(data), 0); err != nil {
		return fmt.Errorf("failed to save trace info to store: %w", err)
	}

	return nil
}

// loadTraceInfoDirect loads trace info without unarchiving (internal use)
func (d *Driver) loadTraceInfoDirect(traceID string) (*types.TraceInfo, error) {
	key := d.getKey(traceID, "info")

	value, ok := d.store.Get(key)
	if !ok {
		return nil, nil
	}

	dataStr, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("invalid data type in store")
	}

	var info types.TraceInfo
	if err := json.Unmarshal([]byte(dataStr), &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal trace info: %w", err)
	}

	return &info, nil
}

// LoadTraceInfo loads trace metadata from store
func (d *Driver) LoadTraceInfo(ctx context.Context, traceID string) (*types.TraceInfo, error) {
	// Check if archived and extract if needed
	archived, err := d.IsArchived(ctx, traceID)
	if err != nil {
		return nil, fmt.Errorf("failed to check archive status: %w", err)
	}
	if archived {
		if err := d.unarchive(ctx, traceID); err != nil {
			return nil, fmt.Errorf("failed to unarchive trace: %w", err)
		}
	}

	return d.loadTraceInfoDirect(traceID)
}

// DeleteTrace removes entire trace from store
func (d *Driver) DeleteTrace(ctx context.Context, traceID string) error {
	// Get all keys
	allKeys := d.store.Keys()
	prefix := d.getKey(traceID, "")

	// Delete all keys matching pattern: {prefix}:{traceID}:*
	for _, key := range allKeys {
		if strings.HasPrefix(key, prefix) {
			_ = d.store.Del(key) // Ignore errors
		}
	}

	return nil
}

// SaveUpdate persists a trace update event to store (append to list)
func (d *Driver) SaveUpdate(ctx context.Context, traceID string, update *types.TraceUpdate) error {
	// Check if archived - archived traces are read-only
	archived, err := d.IsArchived(ctx, traceID)
	if err != nil {
		return fmt.Errorf("failed to check archive status: %w", err)
	}
	if archived {
		return fmt.Errorf("cannot save update: trace %s is archived (read-only)", traceID)
	}

	key := d.getKey(traceID, "updates")

	// Lock to prevent concurrent updates
	d.updatesMu.Lock()
	defer d.updatesMu.Unlock()

	// Load existing updates
	existingUpdates, _ := d.LoadUpdates(ctx, traceID, 0)

	// Append new update
	existingUpdates = append(existingUpdates, update)

	// Marshal all updates
	data, err := json.Marshal(existingUpdates)
	if err != nil {
		return fmt.Errorf("failed to marshal updates: %w", err)
	}

	// Save back to store
	if err := d.store.Set(key, string(data), 0); err != nil {
		return fmt.Errorf("failed to save updates to store: %w", err)
	}

	return nil
}

// LoadUpdates loads trace update events from store
func (d *Driver) LoadUpdates(ctx context.Context, traceID string, since int64) ([]*types.TraceUpdate, error) {
	key := d.getKey(traceID, "updates")

	// Get data from store
	value, ok := d.store.Get(key)
	if !ok {
		return []*types.TraceUpdate{}, nil
	}

	dataStr, ok := value.(string)
	if !ok {
		return []*types.TraceUpdate{}, nil
	}

	// Unmarshal updates array
	var allUpdates []*types.TraceUpdate
	if err := json.Unmarshal([]byte(dataStr), &allUpdates); err != nil {
		return []*types.TraceUpdate{}, nil
	}

	// Filter by timestamp
	filtered := make([]*types.TraceUpdate, 0)
	for _, update := range allUpdates {
		if update.Timestamp >= since {
			filtered = append(filtered, update)
		}
	}

	return filtered, nil
}

// Close closes the store driver
// Archive archives a trace by compressing and merging keys
func (d *Driver) Archive(ctx context.Context, traceID string) error {
	// Check if already archived
	archived, err := d.IsArchived(ctx, traceID)
	if err != nil {
		return fmt.Errorf("failed to check archive status: %w", err)
	}
	if archived {
		return fmt.Errorf("trace %s is already archived", traceID)
	}

	// Step 1: Update trace info to mark as archived BEFORE creating archive
	info, err := d.loadTraceInfoDirect(traceID)
	if err != nil {
		return fmt.Errorf("failed to load trace info: %w", err)
	}
	if info != nil {
		now := time.Now().UnixMilli()
		info.Archived = true
		info.ArchivedAt = &now
		// Save trace info directly without archive check
		key := d.getKey(traceID, "info")
		infoData, err := json.Marshal(info)
		if err != nil {
			return fmt.Errorf("failed to marshal trace info: %w", err)
		}
		if err := d.store.Set(key, string(infoData), 0); err != nil {
			return fmt.Errorf("failed to save trace info: %w", err)
		}
	}

	// Step 2: Collect all keys for this trace
	prefix := d.getKeyPrefix(traceID)
	allKeys := []string{
		d.getTraceInfoKey(traceID),
		d.getUpdatesKey(traceID),
	}

	// Get all node keys
	nodePrefix := prefix + "nodes:"
	nodeKeys, err := d.listKeysByPrefix(ctx, nodePrefix)
	if err != nil {
		return fmt.Errorf("failed to list node keys: %w", err)
	}
	allKeys = append(allKeys, nodeKeys...)

	// Get all space keys
	spacePrefix := prefix + "spaces:"
	spaceKeys, err := d.listKeysByPrefix(ctx, spacePrefix)
	if err != nil {
		return fmt.Errorf("failed to list space keys: %w", err)
	}
	allKeys = append(allKeys, spaceKeys...)

	// Get all log keys
	logPrefix := prefix + "logs:"
	logKeys, err := d.listKeysByPrefix(ctx, logPrefix)
	if err != nil {
		return fmt.Errorf("failed to list log keys: %w", err)
	}
	allKeys = append(allKeys, logKeys...)

	// Step 3: Collect all data into a single map
	archiveData := make(map[string]json.RawMessage)
	for _, key := range allKeys {
		data, ok := d.store.Get(key)
		if !ok {
			continue // Skip missing keys
		}
		// Convert to string then to bytes
		dataStr, ok := data.(string)
		if !ok {
			continue
		}
		archiveData[key] = json.RawMessage(dataStr)
	}

	// Step 4: Marshal to JSON
	jsonData, err := json.Marshal(archiveData)
	if err != nil {
		return fmt.Errorf("failed to marshal archive data: %w", err)
	}

	// Step 5: Compress with gzip
	var compressedBuf bytes.Buffer
	gzipWriter := gzip.NewWriter(&compressedBuf)
	if _, err := gzipWriter.Write(jsonData); err != nil {
		return fmt.Errorf("failed to compress archive: %w", err)
	}
	if err := gzipWriter.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Step 6: Save compressed archive
	archiveKey := d.getArchiveKey(traceID)
	if err := d.store.Set(archiveKey, compressedBuf.String(), 0); err != nil {
		return fmt.Errorf("failed to save archive: %w", err)
	}

	// Step 7: Delete original keys (except trace info and archive)
	for _, key := range allKeys {
		if key == d.getTraceInfoKey(traceID) {
			continue // Keep trace info
		}
		_ = d.store.Del(key) // Ignore errors on delete
	}

	return nil
}

// IsArchived checks if a trace is archived
func (d *Driver) IsArchived(ctx context.Context, traceID string) (bool, error) {
	archiveKey := d.getArchiveKey(traceID)
	exists := d.store.Has(archiveKey)
	return exists, nil
}

// unarchive extracts an archived trace (helper method)
func (d *Driver) unarchive(ctx context.Context, traceID string) error {
	archiveKey := d.getArchiveKey(traceID)

	// Get compressed archive
	compressedData, ok := d.store.Get(archiveKey)
	if !ok {
		return fmt.Errorf("archive not found for trace: %s", traceID)
	}

	// Convert to string then to bytes
	compressedStr, ok := compressedData.(string)
	if !ok {
		return fmt.Errorf("invalid archive data type")
	}

	// Decompress
	gzipReader, err := gzip.NewReader(bytes.NewReader([]byte(compressedStr)))
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	jsonData, err := io.ReadAll(gzipReader)
	if err != nil {
		return fmt.Errorf("failed to decompress archive: %w", err)
	}

	// Unmarshal archive data
	var archiveData map[string]json.RawMessage
	if err := json.Unmarshal(jsonData, &archiveData); err != nil {
		return fmt.Errorf("failed to unmarshal archive: %w", err)
	}

	// Restore all keys
	for key, value := range archiveData {
		if err := d.store.Set(key, string(value), 0); err != nil {
			return fmt.Errorf("failed to restore key %s: %w", key, err)
		}
	}

	return nil
}

// listKeysByPrefix lists all keys with a given prefix (helper method)
func (d *Driver) listKeysByPrefix(ctx context.Context, prefix string) ([]string, error) {
	// Get all keys from store and filter by prefix
	allKeys := d.store.Keys()
	var matchingKeys []string
	for _, key := range allKeys {
		if strings.HasPrefix(key, prefix) {
			matchingKeys = append(matchingKeys, key)
		}
	}
	return matchingKeys, nil
}

// getArchiveKey returns the store key for an archived trace
func (d *Driver) getArchiveKey(traceID string) string {
	return fmt.Sprintf("trace:%s:archive", traceID)
}

func (d *Driver) Close() error {
	// Store connection is managed by gou, no cleanup needed
	return nil
}
