package dingtalk

import (
	"context"
	"sync"

	"github.com/yaoapp/yao/agent/robot/logger"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	dtapi "github.com/yaoapp/yao/integrations/dingtalk"
)

var log = logger.New("dingtalk")

// Adapter implements the integrations.Adapter interface for DingTalk.
//
// Architecture:
//   - One DingTalk Stream client per registered bot for real-time message reception
//   - One dedup cleaner goroutine removes expired keys every hour
type Adapter struct {
	mu     sync.RWMutex
	bots   map[string]*botEntry // robotID -> *botEntry
	appIdx map[string]string    // clientID -> robotID
	dedup  *dedupStore
	stopCh chan struct{}
}

// botEntry holds the state for one robot's DingTalk integration.
type botEntry struct {
	robotID      string
	clientID     string
	clientSecret string
	bot          *dtapi.Bot
	cancelFn     context.CancelFunc
}

// NewAdapter creates a new DingTalk adapter.
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
	dtConf := extractConfig(robot)
	log.Debug("Apply robot=%s dtConf=%v", robot.MemberID, dtConf != nil)

	if dtConf == nil || !dtConf.Enabled || dtConf.ClientID == "" || dtConf.ClientSecret == "" {
		a.removeBot(robot.MemberID)
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if existing, ok := a.bots[robot.MemberID]; ok {
		if existing.clientID == dtConf.ClientID &&
			existing.clientSecret == dtConf.ClientSecret {
			return
		}
		a.removeBotLocked(robot.MemberID)
	}

	bot := dtapi.NewBot(dtConf.ClientID, dtConf.ClientSecret)

	streamCtx, streamCancel := context.WithCancel(context.Background())
	entry := &botEntry{
		robotID:      robot.MemberID,
		clientID:     dtConf.ClientID,
		clientSecret: dtConf.ClientSecret,
		bot:          bot,
		cancelFn:     streamCancel,
	}
	a.bots[robot.MemberID] = entry
	a.appIdx[dtConf.ClientID] = robot.MemberID

	go a.streamLoop(streamCtx, entry)

	log.Info("dingtalk adapter: registered robot=%s client=%s", robot.MemberID, dtConf.ClientID)
}

// Remove is called by the Dispatcher when a robot is deleted.
func (a *Adapter) Remove(ctx context.Context, robotID string) {
	a.removeBot(robotID)
}

// Shutdown stops all stream connections and dedup cleaner.
func (a *Adapter) Shutdown() {
	close(a.stopCh)
	a.mu.Lock()
	for _, entry := range a.bots {
		if entry.cancelFn != nil {
			entry.cancelFn()
		}
	}
	a.mu.Unlock()
	log.Info("dingtalk adapter: shutdown complete")
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
	if entry.clientID != "" {
		delete(a.appIdx, entry.clientID)
	}
	delete(a.bots, robotID)
	log.Info("dingtalk adapter: unregistered robot=%s", robotID)
}

func (a *Adapter) resolveByClientID(clientID string) (*botEntry, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	robotID, ok := a.appIdx[clientID]
	if !ok {
		return nil, false
	}
	entry, ok := a.bots[robotID]
	return entry, ok
}

func extractConfig(robot *robottypes.Robot) *robottypes.DingTalkConfig {
	if robot.Config == nil || robot.Config.Integrations == nil {
		return nil
	}
	return robot.Config.Integrations.DingTalk
}
