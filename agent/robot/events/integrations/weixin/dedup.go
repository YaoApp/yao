package weixin

import (
	"sync"
	"time"
)

const (
	dedupTTL           = 24 * time.Hour
	dedupCleanInterval = time.Hour
)

type dedupStore struct {
	m sync.Map
}

func newDedupStore() *dedupStore {
	return &dedupStore{}
}

func (d *dedupStore) markSeen(key string) bool {
	now := time.Now().Unix()
	_, loaded := d.m.LoadOrStore(key, now)
	return !loaded
}

func (d *dedupStore) cleaner(stopCh <-chan struct{}) {
	ticker := time.NewTicker(dedupCleanInterval)
	defer ticker.Stop()
	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			cutoff := time.Now().Add(-dedupTTL).Unix()
			d.m.Range(func(key, value any) bool {
				if ts, ok := value.(int64); ok && ts < cutoff {
					d.m.Delete(key)
				}
				return true
			})
		}
	}
}
