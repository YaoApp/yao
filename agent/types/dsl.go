package types

import "github.com/yaoapp/gou/store"

// GetCacheStore get the cache store
func (dsl *DSL) GetCacheStore() (store.Store, error) {
	if dsl.Cache == "" {
		return store.Get("__yao.agent.cache")
	}
	return store.Get(dsl.Cache)
}
