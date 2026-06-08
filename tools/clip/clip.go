package clip

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	defaultTTL      = 24 * time.Hour
	cleanupInterval = 10 * time.Minute
	maxDataSize     = 5 * 1024 * 1024 // 5MB
)

// Clip represents a stored content clip.
type Clip struct {
	ID          string            `json:"id"`
	Label       string            `json:"label"`
	Description string            `json:"description"`
	Data        map[string]string `json:"data"`
	CreatedAt   time.Time         `json:"created_at"`
	UserID      string            `json:"user_id"`
	TeamID      string            `json:"team_id"`
}

var (
	store   = make(map[string]*Clip)
	storeMu sync.RWMutex
)

func init() {
	go cleanupLoop()
}

func cleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		storeMu.Lock()
		for id, c := range store {
			if now.Sub(c.CreatedAt) > defaultTTL {
				delete(store, id)
			}
		}
		storeMu.Unlock()
	}
}

func newClipID() string {
	return "clip://" + uuid.New().String()
}

func writeClip(userID, teamID, label, description string, data map[string]string) *Clip {
	c := &Clip{
		ID:          newClipID(),
		Label:       label,
		Description: description,
		Data:        data,
		CreatedAt:   time.Now(),
		UserID:      userID,
		TeamID:      teamID,
	}
	storeMu.Lock()
	store[c.ID] = c
	storeMu.Unlock()
	return c
}

func readClip(userID, teamID, id string) *Clip {
	storeMu.RLock()
	c, ok := store[id]
	storeMu.RUnlock()
	if !ok {
		return nil
	}
	if c.UserID != userID || c.TeamID != teamID {
		return nil
	}
	return c
}

func listClips(userID, teamID string) []*Clip {
	storeMu.RLock()
	defer storeMu.RUnlock()
	var result []*Clip
	for _, c := range store {
		if c.UserID == userID && c.TeamID == teamID {
			result = append(result, c)
		}
	}
	return result
}

// dataSize calculates the total byte size of a data map.
func dataSize(data map[string]string) int {
	total := 0
	for k, v := range data {
		total += len(k) + len(v)
	}
	return total
}
