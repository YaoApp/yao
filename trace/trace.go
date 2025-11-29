package trace

import (
	"context"
	"fmt"
	"sync"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/trace/local"
	"github.com/yaoapp/yao/trace/pubsub"
	"github.com/yaoapp/yao/trace/store"
	"github.com/yaoapp/yao/trace/types"
)

// Driver types
const (
	Local = "local" // Local disk storage
	Store = "store" // Gou store storage
)

// Global trace registry and pubsub services
var (
	registry   = make(map[string]*types.TraceInfo)
	registryMu sync.RWMutex

	// Each trace has its own independent pubsub service
	pubsubRegistry   = make(map[string]*pubsub.PubSub)
	pubsubRegistryMu sync.RWMutex
)

// getDriver creates a driver instance based on driver type and options
func getDriver(driver string, options ...any) (types.Driver, error) {
	var drv types.Driver
	var err error

	switch driver {
	case Local:
		basePath := "" // empty means use log directory from config
		if len(options) > 0 {
			if path, ok := options[0].(string); ok {
				basePath = path
			}
		}
		drv, err = local.New(basePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create local driver: %w", err)
		}

	case Store:
		storeName := "__yao.store" // default: use system common store
		prefix := ""               // empty means use driver's default prefix "__trace"

		if len(options) > 0 {
			if name, ok := options[0].(string); ok {
				storeName = name
			}
		}
		if len(options) > 1 {
			if p, ok := options[1].(string); ok {
				prefix = p
			}
		}

		drv, err = store.New(storeName, prefix)
		if err != nil {
			return nil, fmt.Errorf("failed to create store driver: %w", err)
		}

	default:
		return nil, fmt.Errorf("unknown driver: %s", driver)
	}

	return drv, nil
}

// GenTraceID generate a new trace ID with date prefix format: YYYYMMDDnnnnnnnnnnnn
// Format: 20251118123456789012 (8-digit date + 12-digit unique ID)
// The date prefix enables directory-based storage organization (e.g., traces/20251118/)
// safe: optional parameter, reserved for future safe mode implementation (collision detection)
func GenTraceID(safe ...bool) string {
	// Generate trace ID with format: YYYYMMDD + 12-digit NanoID
	// Total length: 20 characters (8 date + 12 random)
	// Using NanoID for better collision resistance in concurrent scenarios

	now := time.Now()
	// Date prefix: YYYYMMDD (8 digits)
	prefix := now.Format("20060102")

	// Generate 12-character random suffix using NanoID with numeric alphabet
	// This provides much better uniqueness than timestamp-based approach
	const alphabet = "0123456789"
	const length = 12

	suffix, err := gonanoid.Generate(alphabet, length)
	if err != nil {
		// Fallback: use nanosecond timestamp if NanoID fails
		nanoTimestamp := now.UnixNano()
		suffix = fmt.Sprintf("%012d", nanoTimestamp%1000000000000) // 12 digits
	}

	return prefix + suffix
}

// New creates a new trace manager with specified driver
// Returns: traceID, manager, error
// ctx: context for the trace manager
// driver: Local or Store
// option: trace options (optional)
// driverOptions: driver-specific options (e.g., base path for local, store name for store)
func New(ctx context.Context, driver string, option *types.TraceOption, driverOptions ...any) (string, types.Manager, error) {
	now := time.Now().UnixMilli()

	// Handle nil option
	if option == nil {
		option = &types.TraceOption{}
	}

	// Generate ID if not provided
	traceID := option.ID
	if traceID == "" {
		traceID = GenTraceID()
	}

	// Check if ID already exists (in registry or storage)
	if IsLoaded(traceID) {
		return "", nil, fmt.Errorf("trace ID already loaded in registry: %s", traceID)
	}

	// Create driver instance
	drv, err := getDriver(driver, driverOptions...)
	if err != nil {
		return "", nil, err
	}

	// Check if exists in storage - if so, load it instead
	exists, err := Exists(ctx, driver, traceID, driverOptions...)
	if err != nil {
		return "", nil, fmt.Errorf("failed to check trace existence: %w", err)
	}
	if exists {
		// Trace exists in storage, load it
		return LoadFromStorage(ctx, driver, traceID, driverOptions...)
	}

	// Create independent PubSub service for this trace
	pubsubService := pubsub.New()

	// Register pubsub service
	pubsubRegistryMu.Lock()
	pubsubRegistry[traceID] = pubsubService
	pubsubRegistryMu.Unlock()

	// Create Manager instance with the driver and pubsub reference
	// Manager uses pubsub only for publishing, doesn't manage its lifecycle
	manager, err := NewManager(ctx, traceID, drv, pubsubService, option)
	if err != nil {
		// Clean up pubsub if manager creation fails
		pubsubRegistryMu.Lock()
		delete(pubsubRegistry, traceID)
		pubsubRegistryMu.Unlock()
		pubsubService.Stop()
		return "", nil, fmt.Errorf("failed to create manager: %w", err)
	}

	// Create trace info
	info := &types.TraceInfo{
		ID:        traceID,
		Driver:    driver,
		Status:    types.TraceStatusPending, // Initial status is pending
		Options:   driverOptions,
		Manager:   manager,
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: option.CreatedBy,
		UpdatedBy: option.CreatedBy, // Initially same as CreatedBy
		TeamID:    option.TeamID,
		TenantID:  option.TenantID,
		Metadata:  option.Metadata,
	}

	// Register trace in global registry
	registryMu.Lock()
	registry[traceID] = info
	registryMu.Unlock()

	// Persist trace info to driver
	if err := drv.SaveTraceInfo(ctx, info); err != nil {
		// If save fails, remove from registry and return error
		registryMu.Lock()
		delete(registry, traceID)
		registryMu.Unlock()
		return "", nil, fmt.Errorf("failed to save trace info: %w", err)
	}

	return traceID, manager, nil
}

// GetPubSub returns the pubsub service for a trace
func GetPubSub(traceID string) *pubsub.PubSub {
	pubsubRegistryMu.RLock()
	defer pubsubRegistryMu.RUnlock()
	return pubsubRegistry[traceID]
}

// Load loads an existing trace by ID from the registry
// Returns: manager, error
// traceID: the trace ID to load
func Load(traceID string) (types.Manager, error) {
	registryMu.RLock()
	info, exists := registry[traceID]
	registryMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("trace not found in registry: %s (use LoadFromStorage to load from persistent storage)", traceID)
	}

	return info.Manager, nil
}

// LoadFromStorage loads a trace from persistent storage and activates it in registry
// This is used to resume a trace that was previously created but not currently loaded
// Returns: traceID, manager, error
// ctx: context for the trace manager
// driver: Local or Store (must match the driver used to create the trace)
// traceID: the trace ID to load
// driverOptions: driver-specific options (e.g., base path for local, store name for store)
func LoadFromStorage(ctx context.Context, driver string, traceID string, driverOptions ...any) (string, types.Manager, error) {
	// Check if already loaded
	if IsLoaded(traceID) {
		registryMu.RLock()
		info := registry[traceID]
		registryMu.RUnlock()
		return traceID, info.Manager, nil
	}

	// Create driver instance
	drv, err := getDriver(driver, driverOptions...)
	if err != nil {
		return "", nil, err
	}

	// Load trace info from storage
	storedInfo, err := drv.LoadTraceInfo(ctx, traceID)
	if err != nil {
		drv.Close()
		return "", nil, fmt.Errorf("failed to load trace info: %w", err)
	}
	if storedInfo == nil {
		drv.Close()
		return "", nil, fmt.Errorf("trace not found in storage: %s", traceID)
	}

	// Create or reuse PubSub service for this trace
	pubsubRegistryMu.Lock()
	pubsubService, exists := pubsubRegistry[traceID]
	if !exists {
		pubsubService = pubsub.New()
		pubsubRegistry[traceID] = pubsubService
	}
	pubsubRegistryMu.Unlock()

	// Create Manager instance with the driver and pubsub reference
	// Note: We need to reconstruct the manager from stored data
	// TODO: Implement proper restoration of manager state from storage
	// For loaded traces, we don't have the original option, so pass nil
	manager, err := NewManager(ctx, traceID, drv, pubsubService, nil)
	if err != nil {
		drv.Close()
		if !exists {
			// Clean up pubsub if we just created it
			pubsubRegistryMu.Lock()
			delete(pubsubRegistry, traceID)
			pubsubRegistryMu.Unlock()
			pubsubService.Stop()
		}
		return "", nil, fmt.Errorf("failed to create manager: %w", err)
	}

	// Update stored info with new manager
	storedInfo.Manager = manager
	storedInfo.UpdatedAt = time.Now().UnixMilli()

	// Register in global registry
	registryMu.Lock()
	registry[traceID] = storedInfo
	registryMu.Unlock()

	return traceID, manager, nil
}

// GetInfo returns the trace metadata from storage
// This function reads from persistent storage and can be used even if the trace is not in registry
// ctx: context for the operation
// driver: Local or Store (must match the driver used to create the trace)
// traceID: the trace ID
// options: driver-specific options (e.g., base path for local, store name for store)
func GetInfo(ctx context.Context, driver string, traceID string, options ...any) (*types.TraceInfo, error) {
	// Try registry first (if trace is active)
	registryMu.RLock()
	info, exists := registry[traceID]
	registryMu.RUnlock()

	if exists {
		// Return a copy to prevent external modification
		infoCopy := *info
		infoCopy.Manager = nil // Don't expose manager in info
		return &infoCopy, nil
	}

	// Not in registry, load from driver
	drv, err := getDriver(driver, options...)
	if err != nil {
		return nil, err
	}
	defer drv.Close()

	// Load from storage
	storedInfo, err := drv.LoadTraceInfo(ctx, traceID)
	if err != nil {
		return nil, fmt.Errorf("failed to load trace info: %w", err)
	}

	if storedInfo == nil {
		return nil, fmt.Errorf("trace not found: %s", traceID)
	}

	// Don't expose manager for stored info (manager is only available for active traces)
	storedInfo.Manager = nil
	return storedInfo, nil
}

// MarkCancelled marks the trace as cancelled without using the context
// This is useful when the HTTP context has been cancelled and we can't use it for trace operations
// traceID: the trace ID to mark as cancelled
// reason: the cancellation reason
func MarkCancelled(traceID string, reason string) error {
	log.Trace("[TRACE] MarkCancelled called: traceID=%s, reason=%s", traceID, reason)

	registryMu.RLock()
	info, exists := registry[traceID]
	registryMu.RUnlock()

	if !exists {
		log.Trace("[TRACE] MarkCancelled: trace not found in registry")
		return fmt.Errorf("trace not found in registry: %s", traceID)
	}

	mgr, ok := info.Manager.(*manager)
	if !ok {
		log.Trace("[TRACE] MarkCancelled: invalid manager type")
		return fmt.Errorf("invalid manager type for trace: %s", traceID)
	}

	log.Trace("[TRACE] MarkCancelled: starting to mark nodes and trace as cancelled")

	// Get independent pubsub service
	ps := GetPubSub(traceID)
	if ps != nil {
		log.Trace("[TRACE] MarkCancelled: current subscriber count: %d", ps.SubscriberCount())
	}

	now := time.Now().UnixMilli()

	// Use background context since the original context is cancelled
	bgCtx := context.Background()

	// Load trace tree from driver (disk)
	log.Trace("[TRACE] MarkCancelled: loading trace tree from driver")
	rootNode, err := mgr.driver.LoadTrace(bgCtx, traceID)
	if err != nil {
		log.Trace("[TRACE] MarkCancelled: failed to load trace tree: %v", err)
		return fmt.Errorf("failed to load trace tree: %w", err)
	}

	if rootNode == nil {
		log.Trace("[TRACE] MarkCancelled: no root node found")
		return fmt.Errorf("no root node found for trace: %s", traceID)
	}

	// Mark incomplete nodes as failed (recursively walk tree)
	log.Trace("[TRACE] MarkCancelled: marking incomplete nodes as failed")
	var markNodesFailed func(node *types.TraceNode)
	markNodesFailed = func(node *types.TraceNode) {
		if node.Status != types.StatusCompleted && node.Status != types.StatusFailed {
			log.Trace("[TRACE] MarkCancelled: marking node %s as failed", node.ID)
			node.Status = types.StatusFailed
			node.EndTime = now
			node.UpdatedAt = now

			// Save node to driver
			if err := mgr.driver.SaveNode(bgCtx, traceID, node); err != nil {
				log.Trace("[TRACE] MarkCancelled: failed to save node %s: %v", node.ID, err)
			}

			// Broadcast node failed event (also saves to disk)
			subscriberCount := 0
			if ps := GetPubSub(traceID); ps != nil {
				subscriberCount = ps.SubscriberCount()
			}
			log.Trace("[TRACE] MarkCancelled: publishing node failed event for node %s to %d subscribers", node.ID, subscriberCount)
			mgr.addUpdateAndBroadcast(&types.TraceUpdate{
				Type:      types.UpdateTypeNodeFailed,
				TraceID:   traceID,
				NodeID:    node.ID,
				Timestamp: now,
				Data: &types.NodeFailedData{
					NodeID:   node.ID,
					Status:   types.CompleteStatusFailed,
					EndTime:  now,
					Duration: now - node.StartTime,
					Error:    reason,
				},
			})
			log.Trace("[TRACE] MarkCancelled: node failed event broadcasted for node %s", node.ID)
		}

		// Process children
		for _, child := range node.Children {
			markNodesFailed(child)
		}
	}
	markNodesFailed(rootNode)

	// Load trace info
	log.Trace("[TRACE] MarkCancelled: loading trace info from driver")
	traceInfo, err := mgr.driver.LoadTraceInfo(bgCtx, traceID)
	if err != nil {
		log.Trace("[TRACE] MarkCancelled: failed to load trace info: %v", err)
		return fmt.Errorf("failed to load trace info: %w", err)
	}

	// Update trace status to cancelled
	log.Trace("[TRACE] MarkCancelled: updating trace status to cancelled")
	traceInfo.Status = types.TraceStatusCancelled
	traceInfo.UpdatedAt = now
	if err := mgr.driver.SaveTraceInfo(bgCtx, traceInfo); err != nil {
		log.Trace("[TRACE] MarkCancelled: failed to save trace info: %v", err)
		return fmt.Errorf("failed to save trace info: %w", err)
	}

	// Set trace status in state machine
	mgr.stateSetTraceStatus(types.TraceStatusCancelled)
	mgr.stateMarkCompleted()

	// Broadcast completion update (saves to disk and publishes to subscribers)
	subscriberCount := 0
	if ps := GetPubSub(traceID); ps != nil {
		subscriberCount = ps.SubscriberCount()
	}
	log.Trace("[TRACE] MarkCancelled: publishing completion update to %d subscribers", subscriberCount)
	totalDuration := int64(0)
	if rootNode.CreatedAt > 0 {
		totalDuration = now - rootNode.CreatedAt
	}

	mgr.addUpdateAndBroadcast(&types.TraceUpdate{
		Type:      types.UpdateTypeComplete,
		TraceID:   traceID,
		Timestamp: now,
		Data: &types.TraceCompleteData{
			TraceID:       traceID,
			Status:        types.TraceStatusCancelled,
			TotalDuration: totalDuration,
		},
	})

	log.Trace("[TRACE] MarkCancelled: completed successfully")
	return nil
}

// Release releases a trace from the registry and closes its resources
// traceID: the trace ID to release
func Release(traceID string) error {
	log.Trace("[TRACE] Release called: traceID=%s", traceID)

	registryMu.Lock()
	info, exists := registry[traceID]
	if exists {
		delete(registry, traceID)
	}
	registryMu.Unlock()

	if !exists {
		log.Trace("[TRACE] Release: trace not found in registry")
		return fmt.Errorf("trace not found in registry: %s", traceID)
	}

	// Stop manager
	if mgr, ok := info.Manager.(*manager); ok {
		// Close state machine channel to stop state worker goroutine
		log.Trace("[TRACE] Release: closing state command channel")
		close(mgr.stateCmdChan)

		// Cancel the manager's context to stop other background operations
		if mgr.cancel != nil {
			log.Trace("[TRACE] Release: cancelling manager context")
			mgr.cancel()
		}
	}

	// Stop independent PubSub service
	pubsubRegistryMu.Lock()
	ps, psExists := pubsubRegistry[traceID]
	if psExists {
		delete(pubsubRegistry, traceID)
	}
	pubsubRegistryMu.Unlock()

	if psExists && ps != nil {
		subscriberCount := ps.SubscriberCount()
		log.Trace("[TRACE] Release: stopping pubsub service with %d active subscribers", subscriberCount)
		ps.Stop()
	}

	log.Trace("[TRACE] Release: completed")
	return nil
}

// IsLoaded checks if a trace is loaded in the registry (active state)
func IsLoaded(traceID string) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()
	_, exists := registry[traceID]
	return exists
}

// Exists checks if a trace exists in persistent storage
// ctx: context for the operation
// driver: Local or Store (must match the driver used to create the trace)
// traceID: the trace ID
// options: driver-specific options (e.g., base path for local, store name for store)
func Exists(ctx context.Context, driver string, traceID string, options ...any) (bool, error) {
	// Check registry first (if loaded)
	if IsLoaded(traceID) {
		return true, nil
	}

	// Check persistent storage
	drv, err := getDriver(driver, options...)
	if err != nil {
		return false, err
	}
	defer drv.Close()

	// Try to load trace info from storage
	info, err := drv.LoadTraceInfo(ctx, traceID)
	if err != nil {
		// If error is not found, return false; otherwise return error
		return false, nil
	}

	return info != nil, nil
}

// List returns all active trace IDs in the registry
func List() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	ids := make([]string, 0, len(registry))
	for id := range registry {
		ids = append(ids, id)
	}
	return ids
}

// Remove deletes a trace and all its associated data (nodes, spaces)
// This is a destructive operation and cannot be undone
// Automatically releases the trace from registry if it exists
// driver: Local or Store (must match the driver used to create the trace)
// traceID: the trace ID to remove
// options: driver-specific options
func Remove(ctx context.Context, driver string, traceID string, options ...any) error {
	// Release from registry first (if exists)
	_ = Release(traceID) // Ignore error if not in registry

	// Create driver instance
	drv, err := getDriver(driver, options...)
	if err != nil {
		return err
	}
	defer drv.Close()

	return drv.DeleteTrace(ctx, traceID)
}
