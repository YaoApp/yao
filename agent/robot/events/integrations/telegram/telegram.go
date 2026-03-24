package telegram

import (
	"context"
	"sync"

	"github.com/yaoapp/yao/agent/robot/logger"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	tgapi "github.com/yaoapp/yao/integrations/telegram"
)

var log = logger.New("telegram")

// Adapter implements the integrations.Adapter interface for Telegram Bot API.
//
// Architecture:
//   - One polling goroutine (ticker) iterates all registered bots every 60s
//   - One webhook goroutine listens to integration.webhook.telegram events
//   - One dedup cleaner goroutine removes expired keys every hour
type Adapter struct {
	mu      sync.RWMutex
	bots    map[string]*botEntry // robotID -> *botEntry
	appIdx  map[string]string    // appID  -> robotID (webhook routing)
	dedup   *dedupStore
	webhSub string
	stopCh  chan struct{}
}

// botEntry holds the state for one robot's Telegram integration.
type botEntry struct {
	robotID string
	appID   string
	host    string
	bot     *tgapi.Bot // bound to this robot's token
	offset  int64      // polling offset
}

// NewAdapter creates a new Telegram adapter.
func NewAdapter() *Adapter {
	a := &Adapter{
		bots:   make(map[string]*botEntry),
		appIdx: make(map[string]string),
		dedup:  newDedupStore(),
		stopCh: make(chan struct{}),
	}
	go a.dedup.cleaner(a.stopCh)
	go a.pollLoop()
	return a
}

// Apply is called by the Dispatcher when a robot config is created or updated.
func (a *Adapter) Apply(ctx context.Context, robot *robottypes.Robot) {
	tgConf := extractConfig(robot)
	log.Debug("Apply robot=%s tgConf=%v", robot.MemberID, tgConf != nil)
	if tgConf != nil {
		log.Debug("Apply robot=%s enabled=%v token_len=%d host=%q",
			robot.MemberID, tgConf.Enabled, len(tgConf.BotToken), tgConf.Host)
	}

	if tgConf == nil || !tgConf.Enabled || tgConf.BotToken == "" {
		a.removeBot(robot.MemberID)
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if existing, ok := a.bots[robot.MemberID]; ok {
		if existing.bot.Token() == tgConf.BotToken &&
			existing.appID == tgConf.AppID &&
			existing.host == tgConf.Host {
			return
		}
		a.removeBotLocked(robot.MemberID)
	}

	var opts []tgapi.BotOption
	if tgConf.Host != "" {
		opts = append(opts, tgapi.WithAPIBase(tgConf.Host))
	}
	entry := &botEntry{
		robotID: robot.MemberID,
		appID:   tgConf.AppID,
		host:    tgConf.Host,
		bot:     tgapi.NewBot(tgConf.BotToken, tgConf.WebhookSecret, opts...),
	}
	a.bots[robot.MemberID] = entry
	if tgConf.AppID != "" {
		a.appIdx[tgConf.AppID] = robot.MemberID
	}
	log.Info("telegram adapter: registered robot=%s", robot.MemberID)
}

// Remove is called by the Dispatcher when a robot is deleted.
func (a *Adapter) Remove(ctx context.Context, robotID string) {
	a.removeBot(robotID)
}

// Shutdown stops the polling loop, webhook subscription, and dedup cleaner.
func (a *Adapter) Shutdown() {
	close(a.stopCh)
	a.StopWebhookSubscription()
	log.Info("telegram adapter: shutdown complete")
}

// ResolveBot returns the tgapi.Bot for a given appID, used by the webhook
// verification layer. Returns nil if not found.
func (a *Adapter) ResolveBot(appID string) *tgapi.Bot {
	entry, ok := a.resolveByAppID(appID)
	if !ok {
		return nil
	}
	return entry.bot
}

// --- Bot registry ---

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
	if entry.appID != "" {
		delete(a.appIdx, entry.appID)
	}
	delete(a.bots, robotID)
	log.Info("telegram adapter: unregistered robot=%s", robotID)
}

// snapshot returns a copy of all bot entries for safe iteration outside the lock.
func (a *Adapter) snapshot() []*botEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()
	list := make([]*botEntry, 0, len(a.bots))
	for _, entry := range a.bots {
		list = append(list, entry)
	}
	return list
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

func extractConfig(robot *robottypes.Robot) *robottypes.TelegramConfig {
	if robot.Config == nil || robot.Config.Integrations == nil {
		return nil
	}
	return robot.Config.Integrations.Telegram
}
