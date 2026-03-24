package discord

import (
	"context"
	"sync"

	"github.com/yaoapp/yao/agent/robot/logger"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	dcapi "github.com/yaoapp/yao/integrations/discord"
)

var log = logger.New("discord")

// Adapter implements the integrations.Adapter interface for Discord.
//
// Architecture:
//   - One WebSocket Gateway connection per registered bot via discordgo
//   - One dedup cleaner goroutine removes expired keys every hour
type Adapter struct {
	mu     sync.RWMutex
	bots   map[string]*botEntry // robotID -> *botEntry
	appIdx map[string]string    // appID  -> robotID
	dedup  *dedupStore
	stopCh chan struct{}
}

// botEntry holds the state for one robot's Discord integration.
type botEntry struct {
	robotID  string
	appID    string
	bot      *dcapi.Bot
	cancelFn context.CancelFunc
}

// NewAdapter creates a new Discord adapter.
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
	dcConf := extractConfig(robot)
	log.Debug("Apply robot=%s dcConf=%v", robot.MemberID, dcConf != nil)

	if dcConf == nil || !dcConf.Enabled || dcConf.BotToken == "" {
		a.removeBot(robot.MemberID)
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if existing, ok := a.bots[robot.MemberID]; ok {
		if existing.bot.Token() == dcConf.BotToken &&
			existing.appID == dcConf.AppID {
			return
		}
		a.removeBotLocked(robot.MemberID)
	}

	bot, err := dcapi.NewBot(dcConf.BotToken, dcConf.AppID)
	if err != nil {
		log.Error("discord adapter: create bot failed robot=%s: %v", robot.MemberID, err)
		return
	}

	gwCtx, gwCancel := context.WithCancel(context.Background())
	entry := &botEntry{
		robotID:  robot.MemberID,
		appID:    dcConf.AppID,
		bot:      bot,
		cancelFn: gwCancel,
	}
	a.bots[robot.MemberID] = entry
	if dcConf.AppID != "" {
		a.appIdx[dcConf.AppID] = robot.MemberID
	}

	go a.gatewayLoop(gwCtx, entry)

	log.Info("discord adapter: registered robot=%s app=%s", robot.MemberID, dcConf.AppID)
}

// Remove is called by the Dispatcher when a robot is deleted.
func (a *Adapter) Remove(ctx context.Context, robotID string) {
	a.removeBot(robotID)
}

// Shutdown stops all gateway connections and dedup cleaner.
func (a *Adapter) Shutdown() {
	close(a.stopCh)
	a.mu.Lock()
	for _, entry := range a.bots {
		if entry.cancelFn != nil {
			entry.cancelFn()
		}
		if entry.bot != nil && entry.bot.Session() != nil {
			entry.bot.Session().Close()
		}
	}
	a.mu.Unlock()
	log.Info("discord adapter: shutdown complete")
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
	if entry.bot != nil && entry.bot.Session() != nil {
		entry.bot.Session().Close()
	}
	if entry.appID != "" {
		delete(a.appIdx, entry.appID)
	}
	delete(a.bots, robotID)
	log.Info("discord adapter: unregistered robot=%s", robotID)
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

func extractConfig(robot *robottypes.Robot) *robottypes.DiscordConfig {
	if robot.Config == nil || robot.Config.Integrations == nil {
		return nil
	}
	return robot.Config.Integrations.Discord
}
