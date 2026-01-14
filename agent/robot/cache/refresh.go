package cache

import (
	"sync"
	"time"

	"github.com/yaoapp/yao/agent/robot/types"
)

// RefreshConfig holds refresh configuration
type RefreshConfig struct {
	Interval time.Duration // full refresh interval (default: 1 hour)
}

// DefaultRefreshConfig returns default refresh configuration
func DefaultRefreshConfig() *RefreshConfig {
	return &RefreshConfig{
		Interval: time.Hour,
	}
}

// refreshState holds the refresh goroutine state
type refreshState struct {
	ticker *time.Ticker
	done   chan struct{}
	mu     sync.Mutex
}

var refresher = &refreshState{}

// Refresh refreshes a single robot's config from database
func (c *Cache) Refresh(ctx *types.Context, memberID string) error {
	robot, err := c.LoadByID(ctx, memberID)
	if err != nil {
		// If robot not found or no longer autonomous, remove from cache
		if err == types.ErrRobotNotFound {
			c.Remove(memberID)
			return nil
		}
		return err
	}

	// Check if robot is still active and autonomous
	if !robot.AutonomousMode {
		c.Remove(memberID)
		return nil
	}

	// Update cache
	c.Add(robot)
	return nil
}

// StartAutoRefresh starts periodic full refresh
func (c *Cache) StartAutoRefresh(ctx *types.Context, config *RefreshConfig) {
	if config == nil {
		config = DefaultRefreshConfig()
	}

	refresher.mu.Lock()
	defer refresher.mu.Unlock()

	// Stop existing refresher if any
	if refresher.done != nil {
		close(refresher.done)
	}

	refresher.ticker = time.NewTicker(config.Interval)
	refresher.done = make(chan struct{})

	go func() {
		for {
			select {
			case <-refresher.done:
				refresher.ticker.Stop()
				return
			case <-refresher.ticker.C:
				// Perform full refresh
				_ = c.Load(ctx)
			}
		}
	}()
}

// StopAutoRefresh stops the periodic refresh
func (c *Cache) StopAutoRefresh() {
	refresher.mu.Lock()
	defer refresher.mu.Unlock()

	if refresher.done != nil {
		close(refresher.done)
		refresher.done = nil
	}
}

// RefreshAll reloads all robots from database
func (c *Cache) RefreshAll(ctx *types.Context) error {
	return c.Load(ctx)
}

// Count returns the number of cached robots
func (c *Cache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.robots)
}

// ListAll returns all cached robots (across all teams)
func (c *Cache) ListAll() []*types.Robot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	robots := make([]*types.Robot, 0, len(c.robots))
	for _, robot := range c.robots {
		robots = append(robots, robot)
	}
	return robots
}

// GetByStatus returns robots with the specified status
func (c *Cache) GetByStatus(status types.RobotStatus) []*types.Robot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var robots []*types.Robot
	for _, robot := range c.robots {
		if robot.Status == status {
			robots = append(robots, robot)
		}
	}
	return robots
}

// GetIdle returns all idle robots ready to execute
func (c *Cache) GetIdle() []*types.Robot {
	return c.GetByStatus(types.RobotIdle)
}

// GetWorking returns all currently working robots
func (c *Cache) GetWorking() []*types.Robot {
	return c.GetByStatus(types.RobotWorking)
}
