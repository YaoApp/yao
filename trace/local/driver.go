package local

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/trace/types"
)

// persistNode is a lightweight version of TraceNode for storage
// Only stores IDs of children instead of full child nodes
type persistNode struct {
	ID          string            `json:"ID"`
	ParentID    string            `json:"ParentID"`
	ChildrenIDs []string          `json:"ChildrenIDs,omitempty"`
	Label       string            `json:"Label,omitempty"`
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
		ParentID:    node.ParentID,
		ChildrenIDs: childrenIDs,
		Label:       node.Label,
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
		ID:       pn.ID,
		ParentID: pn.ParentID,
		Children: nil, // Children will be loaded separately if needed
		TraceNodeOption: types.TraceNodeOption{
			Label:       pn.Label,
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

// Driver the local disk storage driver implementation
type Driver struct {
	basePath string // Base directory for storing trace files
}

// New creates a new local driver
func New(basePath string) (*Driver, error) {
	// If basePath is empty, use log directory from config
	if basePath == "" {
		if config.Conf.Log != "" {
			// Get directory from log file path
			basePath = filepath.Join(filepath.Dir(config.Conf.Log), "traces")
		} else {
			// Fallback to current directory
			basePath = "./traces"
		}
	}

	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return &Driver{
		basePath: basePath,
	}, nil
}

// getTracePath returns the path for a trace directory
// Format: {basePath}/{YYYYMMDD}/{traceID}/
func (d *Driver) getTracePath(traceID string) string {
	// Extract date prefix from traceID (first 8 digits)
	datePrefix := traceID[:8]
	return filepath.Join(d.basePath, datePrefix, traceID)
}

// ensureTraceDir creates the trace directory if it doesn't exist
func (d *Driver) ensureTraceDir(traceID string) error {
	tracePath := d.getTracePath(traceID)
	return os.MkdirAll(tracePath, 0755)
}

// SaveNode persists a node to disk
func (d *Driver) SaveNode(ctx context.Context, traceID string, node *types.TraceNode) error {
	if err := d.ensureTraceDir(traceID); err != nil {
		return err
	}

	// Create nodes directory
	nodesDir := filepath.Join(d.getTracePath(traceID), "nodes")
	if err := os.MkdirAll(nodesDir, 0755); err != nil {
		return fmt.Errorf("failed to create nodes directory: %w", err)
	}

	// Convert to persist format (only store children IDs)
	persistData := toPersistNode(node)

	// Save node as JSON
	filePath := filepath.Join(nodesDir, node.ID+".json")
	data, err := json.MarshalIndent(persistData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal node: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write node file: %w", err)
	}

	return nil
}

// LoadNode loads a node from disk
func (d *Driver) LoadNode(ctx context.Context, traceID string, nodeID string) (*types.TraceNode, error) {
	filePath := filepath.Join(d.getTracePath(traceID), "nodes", nodeID+".json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read node file: %w", err)
	}

	var pn persistNode
	if err := json.Unmarshal(data, &pn); err != nil {
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

// LoadTrace loads the entire trace tree from disk
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

// SaveSpace persists a space to disk
func (d *Driver) SaveSpace(ctx context.Context, traceID string, space *types.TraceSpace) error {
	if err := d.ensureTraceDir(traceID); err != nil {
		return err
	}

	// Create spaces directory
	spacesDir := filepath.Join(d.getTracePath(traceID), "spaces")
	if err := os.MkdirAll(spacesDir, 0755); err != nil {
		return fmt.Errorf("failed to create spaces directory: %w", err)
	}

	// Save space metadata as JSON
	filePath := filepath.Join(spacesDir, space.ID+".json")
	data, err := json.MarshalIndent(space, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal space: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write space file: %w", err)
	}

	return nil
}

// LoadSpace loads a space from disk
func (d *Driver) LoadSpace(ctx context.Context, traceID string, spaceID string) (*types.TraceSpace, error) {
	filePath := filepath.Join(d.getTracePath(traceID), "spaces", spaceID+".json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read space file: %w", err)
	}

	var space types.TraceSpace
	if err := json.Unmarshal(data, &space); err != nil {
		return nil, fmt.Errorf("failed to unmarshal space: %w", err)
	}

	return &space, nil
}

// DeleteSpace removes a space from disk
func (d *Driver) DeleteSpace(ctx context.Context, traceID string, spaceID string) error {
	// Delete space metadata file
	filePath := filepath.Join(d.getTracePath(traceID), "spaces", spaceID+".json")
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete space file: %w", err)
	}

	// Delete space data directory
	dataDir := filepath.Join(d.getTracePath(traceID), "spaces", spaceID)
	if err := os.RemoveAll(dataDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete space data directory: %w", err)
	}

	return nil
}

// ListSpaces lists all space IDs for a trace from disk
func (d *Driver) ListSpaces(ctx context.Context, traceID string) ([]string, error) {
	spacesDir := filepath.Join(d.getTracePath(traceID), "spaces")

	entries, err := os.ReadDir(spacesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read spaces directory: %w", err)
	}

	var spaceIDs []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			// Remove .json extension to get space ID
			spaceID := strings.TrimSuffix(entry.Name(), ".json")
			spaceIDs = append(spaceIDs, spaceID)
		}
	}

	return spaceIDs, nil
}

// getSpaceDataPath returns the path for space data file
func (d *Driver) getSpaceDataPath(traceID, spaceID string) string {
	return filepath.Join(d.getTracePath(traceID), "spaces", spaceID, "data.json")
}

// loadSpaceData loads all key-value pairs for a space
func (d *Driver) loadSpaceData(traceID, spaceID string) (map[string]any, error) {
	filePath := d.getSpaceDataPath(traceID, spaceID)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]any), nil
		}
		return nil, fmt.Errorf("failed to read space data: %w", err)
	}

	var kvData map[string]any
	if err := json.Unmarshal(data, &kvData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal space data: %w", err)
	}

	return kvData, nil
}

// saveSpaceData saves all key-value pairs for a space
func (d *Driver) saveSpaceData(traceID, spaceID string, kvData map[string]any) error {
	filePath := d.getSpaceDataPath(traceID, spaceID)

	// Create space data directory
	dataDir := filepath.Dir(filePath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create space data directory: %w", err)
	}

	// Save as JSON
	data, err := json.MarshalIndent(kvData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal space data: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write space data file: %w", err)
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

// SaveLog appends a log entry to disk
func (d *Driver) SaveLog(ctx context.Context, traceID string, log *types.TraceLog) error {
	if err := d.ensureTraceDir(traceID); err != nil {
		return err
	}

	// Create logs directory
	logsDir := filepath.Join(d.getTracePath(traceID), "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Append log to node's log file (JSONL format)
	filePath := filepath.Join(logsDir, log.NodeID+".jsonl")

	// Marshal log as single-line JSON
	data, err := json.Marshal(log)
	if err != nil {
		return fmt.Errorf("failed to marshal log: %w", err)
	}

	// Append to file
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write log: %w", err)
	}

	return nil
}

// LoadLogs loads all logs for a trace or specific node from disk
func (d *Driver) LoadLogs(ctx context.Context, traceID string, nodeID string) ([]*types.TraceLog, error) {
	logsDir := filepath.Join(d.getTracePath(traceID), "logs")

	var logs []*types.TraceLog

	if nodeID != "" {
		// Load logs for specific node
		filePath := filepath.Join(logsDir, nodeID+".jsonl")
		nodeLogs, err := d.loadLogFile(filePath)
		if err != nil {
			return nil, err
		}
		logs = append(logs, nodeLogs...)
	} else {
		// Load all logs
		entries, err := os.ReadDir(logsDir)
		if err != nil {
			if os.IsNotExist(err) {
				return []*types.TraceLog{}, nil
			}
			return nil, fmt.Errorf("failed to read logs directory: %w", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".jsonl") {
				filePath := filepath.Join(logsDir, entry.Name())
				nodeLogs, err := d.loadLogFile(filePath)
				if err != nil {
					return nil, err
				}
				logs = append(logs, nodeLogs...)
			}
		}
	}

	return logs, nil
}

// loadLogFile loads logs from a JSONL file
func (d *Driver) loadLogFile(filePath string) ([]*types.TraceLog, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*types.TraceLog{}, nil
		}
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	logs := make([]*types.TraceLog, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		var log types.TraceLog
		if err := json.Unmarshal([]byte(line), &log); err != nil {
			// Skip malformed lines
			continue
		}

		logs = append(logs, &log)
	}

	return logs, nil
}

// SaveTraceInfo persists trace metadata to disk
func (d *Driver) SaveTraceInfo(ctx context.Context, info *types.TraceInfo) error {
	if err := d.ensureTraceDir(info.ID); err != nil {
		return err
	}

	filePath := filepath.Join(d.getTracePath(info.ID), "trace_info.json")

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal trace info: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write trace info file: %w", err)
	}

	return nil
}

// LoadTraceInfo loads trace metadata from disk
func (d *Driver) LoadTraceInfo(ctx context.Context, traceID string) (*types.TraceInfo, error) {
	filePath := filepath.Join(d.getTracePath(traceID), "trace_info.json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read trace info file: %w", err)
	}

	var info types.TraceInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal trace info: %w", err)
	}

	return &info, nil
}

// DeleteTrace removes entire trace from disk
func (d *Driver) DeleteTrace(ctx context.Context, traceID string) error {
	tracePath := d.getTracePath(traceID)

	if err := os.RemoveAll(tracePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete trace directory: %w", err)
	}

	return nil
}

// SaveUpdate persists a trace update event to disk (append-only)
func (d *Driver) SaveUpdate(ctx context.Context, traceID string, update *types.TraceUpdate) error {
	if err := d.ensureTraceDir(traceID); err != nil {
		return err
	}

	filePath := filepath.Join(d.getTracePath(traceID), "updates.jsonl")

	// Marshal update to JSON
	data, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("failed to marshal update: %w", err)
	}

	// Append to file (create if not exists)
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open updates file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write update: %w", err)
	}

	return nil
}

// LoadUpdates loads trace update events from disk
func (d *Driver) LoadUpdates(ctx context.Context, traceID string, since int64) ([]*types.TraceUpdate, error) {
	filePath := filepath.Join(d.getTracePath(traceID), "updates.jsonl")

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*types.TraceUpdate{}, nil
		}
		return nil, fmt.Errorf("failed to read updates file: %w", err)
	}

	// Parse line by line
	lines := strings.Split(string(data), "\n")
	updates := make([]*types.TraceUpdate, 0, len(lines))

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var update types.TraceUpdate
		if err := json.Unmarshal([]byte(line), &update); err != nil {
			// Skip malformed lines
			continue
		}

		// Filter by timestamp
		if update.Timestamp >= since {
			updates = append(updates, &update)
		}
	}

	return updates, nil
}

// Close closes the local driver
func (d *Driver) Close() error {
	// No cleanup needed for local file system
	return nil
}
