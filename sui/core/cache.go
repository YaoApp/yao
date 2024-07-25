package core

import (
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/kun/log"
)

// Cache the cache
type Cache struct {
	Data          string
	Global        string
	Config        string
	Guard         string
	GuardRedirect string
	HTML          string
	Root          string
	CacheStore    string
	CacheTime     time.Duration
	DataCacheTime time.Duration
	Script        *Script
	Imports       map[string]string
}

const (
	saveCache uint8 = iota
	removeCache
)

type cacheData struct {
	file  string
	cache *Cache
	cmd   uint8
}

// Caches the caches
var Caches = map[string]*Cache{}
var ch = make(chan *cacheData, 1)

func init() {
	go cacheWriter()
}

func cacheWriter() {
	for {
		select {
		case data := <-ch:
			switch data.cmd {
			case saveCache:
				Caches[data.file] = data.cache
			case removeCache:
				delete(Caches, data.file)
			}
		}
	}
}

// SetCache set the cache
func SetCache(file string, cache *Cache) {
	ch <- &cacheData{file, cache, saveCache}
}

// GetCache get the cache
func GetCache(file string) *Cache {
	if cache, has := Caches[file]; has {
		return cache
	}
	return nil
}

// RemoveCache remove the cache
func RemoveCache(file string) {
	ch <- &cacheData{file, nil, removeCache}
	chScript <- &scriptData{file, nil, removeScript}
}

// CleanCache clean the cache
func CleanCache() {
	Caches = map[string]*Cache{}
}

// GetHTML get the html
func (c *Cache) GetHTML(hash string) (string, bool) {

	store, has := store.Pools[c.CacheStore]
	if !has {
		log.Warn(`[SUI] The cache store "%s" is not found`, c.CacheStore)
		return "", false
	}

	v, has := store.Get(hash)
	if !has {
		return "", false
	}

	return v.(string), true
}

// GetData get the data
func (c *Cache) GetData(hash string) (Data, bool) {
	store, has := store.Pools[c.CacheStore]
	if !has {
		log.Warn(`[SUI] The cache store "%s" is not found`, c.CacheStore)
		return Data{}, false
	}

	v, has := store.Get(hash)
	if !has {
		return Data{}, false
	}

	data := Data{}
	err := jsoniter.Unmarshal(v.([]byte), &data)
	if err != nil {
		log.Error(`[SUI] The data is not a valid json: %s`, err.Error())
		return Data{}, false
	}

	return data, true
}

// SetData set the data
func (c *Cache) SetData(hash string, data Data, ttl time.Duration) {
	store, has := store.Pools[c.CacheStore]
	if !has {
		log.Warn(`[SUI] The cache store "%s" is not found`, c.CacheStore)
		return
	}

	raw, err := jsoniter.Marshal(data)
	if err != nil {
		log.Error(`[SUI] The data is not a valid json: %s`, err.Error())
		return
	}

	store.Set(hash, raw, ttl)
}

// SetHTML set the html
func (c *Cache) SetHTML(hash, html string, ttl time.Duration) {
	store, has := store.Pools[c.CacheStore]
	if !has {
		log.Warn(`[SUI] The cache store "%s" is not found`, c.CacheStore)
		return
	}
	store.Set(hash, html, ttl)
}

// DelHTML del the html
func (c *Cache) DelHTML(hash string) {
	store, has := store.Pools[c.CacheStore]
	if !has {
		log.Warn(`[SUI] The cache store "%s" is not found`, c.CacheStore)
		return
	}
	store.Del(hash)
}
