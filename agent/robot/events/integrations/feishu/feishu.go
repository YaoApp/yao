package feishu

import (
	"context"
	"sync"

	"github.com/yaoapp/yao/agent/robot/logger"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	fsapi "github.com/yaoapp/yao/integrations/feishu"
)

var log = logger.New("feishu")

// Adapter implements the integrations.Adapter interface for Feishu (Lark).
//
// Architecture:
//   - One event subscription per registered bot via Feishu SDK's long-poll/callback mechanism
//   - One dedup cleaner goroutine removes expired keys every hour
type Adapter struct {
	mu     sync.RWMutex
	bots   map[string]*botEntry // robotID -> *botEntry
	appIdx map[string]string    // appID  -> robotID
	dedup  *dedupStore
	stopCh chan struct{}
}

// botEntry holds the state for one robot's Feishu integration.
type botEntry struct {
	robotID   string
	appID     string
	appSecret string
	bot       *fsapi.Bot
	cancelFn  context.CancelFunc // cancels the event subscription goroutine
}

// NewAdapter creates a new Feishu adapter.
func NewAdapter() *Adapter {
	a := &Adapter{
		bots:   make(map[string]*botEntry),
		appIdx: make(map[string]string),
		dedup:  newDedupStore(),
		stopCh: make(chan struct{}),
	}
	go a.dedup.cleaner(a.stopCh)
	return a
}

// Apply is called by the Dispatcher when a robot config is created or updated.
func (a *Adapter) Apply(ctx context.Context, robot *robottypes.Robot) {
	fsConf := extractConfig(robot)
	log.Debug("Apply robot=%s fsConf=%v", robot.MemberID, fsConf != nil)

	if fsConf == nil || !fsConf.Enabled || fsConf.AppID == "" || fsConf.AppSecret == "" {
		a.removeBot(robot.MemberID)
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if existing, ok := a.bots[robot.MemberID]; ok {
		if existing.appID == fsConf.AppID &&
			existing.appSecret == fsConf.AppSecret {
			return
		}
		a.removeBotLocked(robot.MemberID)
	}

	bot := fsapi.NewBot(fsConf.AppID, fsConf.AppSecret)

	streamCtx, streamCancel := context.WithCancel(context.Background())
	entry := &botEntry{
		robotID:   robot.MemberID,
		appID:     fsConf.AppID,
		appSecret: fsConf.AppSecret,
		bot:       bot,
		cancelFn:  streamCancel,
	}
	a.bots[robot.MemberID] = entry
	a.appIdx[fsConf.AppID] = robot.MemberID

	go a.eventLoop(streamCtx, entry)

	log.Info("feishu adapter: registered robot=%s app=%s", robot.MemberID, fsConf.AppID)
}

// Remove is called by the Dispatcher when a robot is deleted.
func (a *Adapter) Remove(ctx context.Context, robotID string) {
	a.removeBot(robotID)
}

// Shutdown stops all event subscriptions and dedup cleaner.
func (a *Adapter) Shutdown() {
	close(a.stopCh)
	a.mu.Lock()
	for _, entry := range a.bots {
		if entry.cancelFn != nil {
			entry.cancelFn()
		}
	}
	a.mu.Unlock()
	log.Info("feishu adapter: shutdown complete")
}

func (a *Adapter) removeBot(robotID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.removeBotLocked(robotID)
}

func (a *Adapter) removeBotLocked(robotID string) {
	entry, ok := a.bots[robotID]
	if !ok {
		return
	}
	if entry.cancelFn != nil {
		entry.cancelFn()
	}
	if entry.appID != "" {
		delete(a.appIdx, entry.appID)
	}
	delete(a.bots, robotID)
	log.Info("feishu adapter: unregistered robot=%s", robotID)
}

func (a *Adapter) resolveByAppID(appID string) (*botEntry, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	robotID, ok := a.appIdx[appID]
	if !ok {
		return nil, false
	}
	entry, ok := a.bots[robotID]
	return entry, ok
}

func extractConfig(robot *robottypes.Robot) *robottypes.FeishuConfig {
	if robot.Config == nil || robot.Config.Integrations == nil {
		return nil
	}
	return robot.Config.Integrations.Feishu
}
