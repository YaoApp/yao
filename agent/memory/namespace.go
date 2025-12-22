package memory

import (
	"time"
)

// Ensure Namespace implements NamespaceAccessor
var _ NamespaceAccessor = (*Namespace)(nil)

// GetID returns the namespace identifier
func (ns *Namespace) GetID() string {
	return ns.ID
}

// GetSpace returns the space type of this namespace
func (ns *Namespace) GetSpace() Space {
	return ns.Space
}

// prefixKey adds the namespace prefix to a key
func (ns *Namespace) prefixKey(key string) string {
	return ns.Prefix + key
}

// Get retrieves a value by key
func (ns *Namespace) Get(key string) (interface{}, bool) {
	return ns.Store.Get(ns.prefixKey(key))
}

// Set stores a value with the default TTL for this namespace
func (ns *Namespace) Set(key string, value interface{}, ttl time.Duration) error {
	if ttl == 0 {
		ttl = ns.Default
	}
	return ns.Store.Set(ns.prefixKey(key), value, ttl)
}

// Has checks if a key exists
func (ns *Namespace) Has(key string) bool {
	return ns.Store.Has(ns.prefixKey(key))
}

// Del deletes a key (supports wildcards)
func (ns *Namespace) Del(key string) error {
	return ns.Store.Del(ns.prefixKey(key))
}

// Keys returns all keys in this namespace
// Uses pattern-based query for efficiency
func (ns *Namespace) Keys(pattern ...string) []string {
	// Build pattern with namespace prefix
	var storePattern string
	if len(pattern) > 0 && pattern[0] != "" {
		storePattern = ns.Prefix + pattern[0]
	} else {
		storePattern = ns.Prefix + "*"
	}

	allKeys := ns.Store.Keys(storePattern)
	prefixLen := len(ns.Prefix)

	// Remove prefix from keys
	result := make([]string, 0, len(allKeys))
	for _, key := range allKeys {
		if len(key) >= prefixLen {
			result = append(result, key[prefixLen:])
		}
	}
	return result
}

// Len returns the number of keys in this namespace
// Uses pattern-based query for efficiency
func (ns *Namespace) Len(pattern ...string) int {
	// Build pattern with namespace prefix
	var storePattern string
	if len(pattern) > 0 && pattern[0] != "" {
		storePattern = ns.Prefix + pattern[0]
	} else {
		storePattern = ns.Prefix + "*"
	}

	return ns.Store.Len(storePattern)
}

// Clear deletes all keys in this namespace
func (ns *Namespace) Clear() {
	ns.Store.Del(ns.Prefix + "*")
}

// GetSet retrieves a value and sets a new value if not exists
func (ns *Namespace) GetSet(key string, ttl time.Duration, getValue func(key string) (interface{}, error)) (interface{}, error) {
	if ttl == 0 {
		ttl = ns.Default
	}
	return ns.Store.GetSet(ns.prefixKey(key), ttl, getValue)
}

// GetDel retrieves a value and deletes it atomically
func (ns *Namespace) GetDel(key string) (interface{}, bool) {
	return ns.Store.GetDel(ns.prefixKey(key))
}

// GetMulti retrieves multiple values by keys
func (ns *Namespace) GetMulti(keys []string) map[string]interface{} {
	prefixedKeys := make([]string, len(keys))
	for i, key := range keys {
		prefixedKeys[i] = ns.prefixKey(key)
	}
	result := ns.Store.GetMulti(prefixedKeys)

	// Remove prefix from result keys
	unprefixed := make(map[string]interface{})
	prefixLen := len(ns.Prefix)
	for k, v := range result {
		if len(k) > prefixLen {
			unprefixed[k[prefixLen:]] = v
		} else {
			unprefixed[k] = v
		}
	}
	return unprefixed
}

// SetMulti stores multiple values
func (ns *Namespace) SetMulti(values map[string]interface{}, ttl time.Duration) {
	if ttl == 0 {
		ttl = ns.Default
	}
	prefixed := make(map[string]interface{})
	for k, v := range values {
		prefixed[ns.prefixKey(k)] = v
	}
	ns.Store.SetMulti(prefixed, ttl)
}

// DelMulti deletes multiple keys
func (ns *Namespace) DelMulti(keys []string) {
	prefixedKeys := make([]string, len(keys))
	for i, key := range keys {
		prefixedKeys[i] = ns.prefixKey(key)
	}
	ns.Store.DelMulti(prefixedKeys)
}

// GetSetMulti retrieves multiple values and sets new values if not exists
func (ns *Namespace) GetSetMulti(keys []string, ttl time.Duration, getValue func(key string) (interface{}, error)) map[string]interface{} {
	if ttl == 0 {
		ttl = ns.Default
	}
	prefixedKeys := make([]string, len(keys))
	for i, key := range keys {
		prefixedKeys[i] = ns.prefixKey(key)
	}
	result := ns.Store.GetSetMulti(prefixedKeys, ttl, getValue)

	// Remove prefix from result keys
	unprefixed := make(map[string]interface{})
	prefixLen := len(ns.Prefix)
	for k, v := range result {
		if len(k) > prefixLen {
			unprefixed[k[prefixLen:]] = v
		} else {
			unprefixed[k] = v
		}
	}
	return unprefixed
}

// Incr increments a numeric value
func (ns *Namespace) Incr(key string, delta int64) (int64, error) {
	return ns.Store.Incr(ns.prefixKey(key), delta)
}

// Decr decrements a numeric value
func (ns *Namespace) Decr(key string, delta int64) (int64, error) {
	return ns.Store.Decr(ns.prefixKey(key), delta)
}

// Push appends values to a list
func (ns *Namespace) Push(key string, values ...interface{}) error {
	return ns.Store.Push(ns.prefixKey(key), values...)
}

// Pop removes and returns an element from a list
func (ns *Namespace) Pop(key string, position int) (interface{}, error) {
	return ns.Store.Pop(ns.prefixKey(key), position)
}

// Pull removes the first occurrence of a value from a list
func (ns *Namespace) Pull(key string, value interface{}) error {
	return ns.Store.Pull(ns.prefixKey(key), value)
}

// PullAll removes all occurrences of values from a list
func (ns *Namespace) PullAll(key string, values []interface{}) error {
	return ns.Store.PullAll(ns.prefixKey(key), values)
}

// AddToSet adds values to a set (no duplicates)
func (ns *Namespace) AddToSet(key string, values ...interface{}) error {
	return ns.Store.AddToSet(ns.prefixKey(key), values...)
}

// ArrayLen returns the length of a list
func (ns *Namespace) ArrayLen(key string) int {
	return ns.Store.ArrayLen(ns.prefixKey(key))
}

// ArrayGet retrieves an element from a list by index
func (ns *Namespace) ArrayGet(key string, index int) (interface{}, error) {
	return ns.Store.ArrayGet(ns.prefixKey(key), index)
}

// ArraySet sets an element in a list by index
func (ns *Namespace) ArraySet(key string, index int, value interface{}) error {
	return ns.Store.ArraySet(ns.prefixKey(key), index, value)
}

// ArraySlice returns a slice of a list
func (ns *Namespace) ArraySlice(key string, skip, limit int) ([]interface{}, error) {
	return ns.Store.ArraySlice(ns.prefixKey(key), skip, limit)
}

// ArrayPage returns a page of a list
func (ns *Namespace) ArrayPage(key string, page, pageSize int) ([]interface{}, error) {
	return ns.Store.ArrayPage(ns.prefixKey(key), page, pageSize)
}

// ArrayAll returns all elements of a list
func (ns *Namespace) ArrayAll(key string) ([]interface{}, error) {
	return ns.Store.ArrayAll(ns.prefixKey(key))
}

// Stats returns statistics for this namespace
func (ns *Namespace) Stats() *NamespaceStats {
	return &NamespaceStats{
		Space:    ns.Space,
		ID:       ns.ID,
		KeyCount: ns.Len(),
		StoreID:  ns.StoreID,
	}
}

// Snapshot returns all key-value pairs in this namespace
// Used for recovery/resume functionality
func (ns *Namespace) Snapshot() map[string]interface{} {
	keys := ns.Keys()
	snapshot := make(map[string]interface{}, len(keys))
	for _, key := range keys {
		if value, ok := ns.Get(key); ok {
			snapshot[key] = value
		}
	}
	return snapshot
}
