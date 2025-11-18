package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/trace/types"
)

// Driver the gou store storage driver implementation
type Driver struct {
	storeName string      // Store name in gou
	store     store.Store // Gou store instance
	prefix    string      // Key prefix for isolation
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

// SaveNode persists a node to store
func (d *Driver) SaveNode(ctx context.Context, traceID string, node *types.TraceNode) error {
	key := d.getKey(traceID, "node", node.ID)

	data, err := json.Marshal(node)
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
	key := d.getKey(traceID, "node", nodeID)

	value, ok := d.store.Get(key)
	if !ok {
		return nil, nil
	}

	dataStr, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("invalid data type in store")
	}

	var node types.TraceNode
	if err := json.Unmarshal([]byte(dataStr), &node); err != nil {
		return nil, fmt.Errorf("failed to unmarshal node: %w", err)
	}

	return &node, nil
}

// LoadTrace loads the entire trace tree from store
func (d *Driver) LoadTrace(ctx context.Context, traceID string) (*types.TraceNode, error) {
	// Load trace info to get root node ID
	info, err := d.LoadTraceInfo(ctx, traceID)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil
	}

	// For now, just return nil - full tree reconstruction can be implemented later
	return nil, nil
}

// SaveSpace persists a space to store
func (d *Driver) SaveSpace(ctx context.Context, traceID string, space *types.TraceSpace) error {
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

// LoadTraceInfo loads trace metadata from store
func (d *Driver) LoadTraceInfo(ctx context.Context, traceID string) (*types.TraceInfo, error) {
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

// Close closes the store driver
func (d *Driver) Close() error {
	// Store connection is managed by gou, no cleanup needed
	return nil
}
