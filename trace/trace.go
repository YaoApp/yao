package trace

import (
	"context"
	"fmt"
	"sync"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/yaoapp/yao/trace/local"
	"github.com/yaoapp/yao/trace/store"
	"github.com/yaoapp/yao/trace/types"
)

// Driver types
const (
	Local = "local" // Local disk storage
	Store = "store" // Gou store storage
)

// Global trace registry
var (
	registry   = make(map[string]*types.TraceInfo)
	registryMu sync.RWMutex
)

// getDriver creates a driver instance based on driver type and options
func getDriver(driver string, options ...any) (types.Driver, error) {
	var drv types.Driver
	var err error

	switch driver {
	case Local:
		basePath := "./traces" // default
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
		storeName := "trace" // default
		if len(options) > 0 {
			if name, ok := options[0].(string); ok {
				storeName = name
			}
		}
		drv, err = store.New(storeName)
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
	// TODO: Implement safe mode with collision detection when needed

	now := time.Now()
	// Date prefix: YYYYMMDD (8 digits)
	prefix := now.Format("20060102")

	// Generate 12-digit unique suffix (timestamp in microseconds + random)
	// Using timestamp ensures uniqueness within the same day
	timestamp := fmt.Sprintf("%06d", now.Unix()%1000000) // 6 digits from timestamp

	const alphabet = "0123456789"
	const length = 6

	random, err := gonanoid.Generate(alphabet, length)
	if err != nil {
		// Fallback to nanoseconds if NanoID generation fails
		random = fmt.Sprintf("%06d", now.Nanosecond()%1000000)
	}

	return prefix + timestamp + random
}

// New creates a new trace manager with specified driver
// Returns: traceID, manager, error
// ctx: context for the trace manager
// driver: Local or Store
// option: trace options (optional)
// driverOptions: driver-specific options (e.g., base path for local, store name for store)
func New(ctx context.Context, driver string, option *types.TraceOption, driverOptions ...any) (string, types.Manager, error) {
	now := time.Now().Unix()

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

	// Create Manager instance with the driver
	manager, err := NewManager(ctx, traceID, drv)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create manager: %w", err)
	}

	// Create trace info
	info := &types.TraceInfo{
		ID:        traceID,
		Driver:    driver,
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

	// Create Manager instance with the driver
	// Note: We need to reconstruct the manager from stored data
	// TODO: Implement proper restoration of manager state from storage
	manager, err := NewManager(ctx, traceID, drv)
	if err != nil {
		drv.Close()
		return "", nil, fmt.Errorf("failed to create manager: %w", err)
	}

	// Update stored info with new manager
	storedInfo.Manager = manager
	storedInfo.UpdatedAt = time.Now().Unix()

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

// Release releases a trace from the registry and closes its resources
// traceID: the trace ID to release
func Release(traceID string) error {
	registryMu.Lock()
	_, exists := registry[traceID]
	if exists {
		delete(registry, traceID)
	}
	registryMu.Unlock()

	if !exists {
		return fmt.Errorf("trace not found in registry: %s", traceID)
	}

	// Close driver resources if manager has a close method
	// (Currently manager doesn't expose driver, but driver has Close method)
	// This is handled when the context is cancelled or program exits

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
