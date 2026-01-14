package cache

import (
	"sync"

	"github.com/yaoapp/yao/agent/robot/types"
)

// Cache implements types.Cache interface
// Thread-safe in-memory cache for Robot instances
type Cache struct {
	robots map[string]*types.Robot // memberID -> Robot
	byTeam map[string][]string     // teamID -> memberIDs
	mu     sync.RWMutex
}

// New creates a new cache instance
func New() *Cache {
	return &Cache{
		robots: make(map[string]*types.Robot),
		byTeam: make(map[string][]string),
	}
}

// Get returns a robot by member ID
// Stub: returns nil (will be implemented in Phase 3)
func (c *Cache) Get(memberID string) *types.Robot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.robots[memberID]
}

// List returns all robots for a team
func (c *Cache) List(teamID string) []*types.Robot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	memberIDs := c.byTeam[teamID]
	robots := make([]*types.Robot, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		if robot := c.robots[memberID]; robot != nil {
			robots = append(robots, robot)
		}
	}
	return robots
}

// Note: Refresh is implemented in refresh.go

// Add adds or updates a robot in cache
func (c *Cache) Add(robot *types.Robot) {
	if robot == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.robots[robot.MemberID] = robot

	// Update team index
	if _, exists := c.byTeam[robot.TeamID]; !exists {
		c.byTeam[robot.TeamID] = []string{}
	}

	// Check if member ID already in team list
	found := false
	for _, id := range c.byTeam[robot.TeamID] {
		if id == robot.MemberID {
			found = true
			break
		}
	}
	if !found {
		c.byTeam[robot.TeamID] = append(c.byTeam[robot.TeamID], robot.MemberID)
	}
}

// Remove removes a robot from cache
func (c *Cache) Remove(memberID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	robot := c.robots[memberID]
	if robot == nil {
		return
	}

	delete(c.robots, memberID)

	// Remove from team index
	teamMembers := c.byTeam[robot.TeamID]
	for i, id := range teamMembers {
		if id == memberID {
			c.byTeam[robot.TeamID] = append(teamMembers[:i], teamMembers[i+1:]...)
			break
		}
	}
}
