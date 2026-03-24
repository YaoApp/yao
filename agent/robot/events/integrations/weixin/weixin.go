package weixin

import (
	"context"
	"sync"

	"github.com/yaoapp/yao/agent/robot/logger"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	weixinapi "github.com/yaoapp/yao/integrations/weixin"
)

var log = logger.New("weixin")

type Adapter struct {
	mu         sync.RWMutex
	bots       map[string]*botEntry
	accountIdx map[string]string // accountID(ilink_bot_id) -> robotID
	dedup      *dedupStore
	stopCh     chan struct{}
}

type botEntry struct {
	robotID     string
	accountID   string
	bot         *weixinapi.Bot
	cancelFn    context.CancelFunc
	ticketCache *typingTicketCache
}

func NewAdapter() *Adapter {
	a := &Adapter{
		bots:       make(map[string]*botEntry),
		accountIdx: make(map[string]string),
		dedup:      newDedupStore(),
		stopCh:     make(chan struct{}),
	}
	go a.dedup.cleaner(a.stopCh)
	return a
}

func (a *Adapter) Apply(ctx context.Context, robot *robottypes.Robot) {
	conf := extractConfig(robot)
	if conf == nil || !conf.Enabled || conf.BotToken == "" {
		a.removeBot(robot.MemberID)
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if existing, ok := a.bots[robot.MemberID]; ok {
		if existing.bot.Token() == conf.BotToken &&
			existing.bot.BaseURL() == resolveBaseURL(conf) &&
			existing.bot.CDNBaseURL() == resolveCDNBaseURL(conf) {
			return
		}
		a.removeBotLocked(robot.MemberID)
	}

	pollCtx, cancel := context.WithCancel(context.Background())
	entry := &botEntry{
		robotID:     robot.MemberID,
		accountID:   conf.AccountID,
		bot:         weixinapi.NewBot(conf.BotToken, resolveBaseURL(conf), resolveCDNBaseURL(conf)),
		cancelFn:    cancel,
		ticketCache: newTypingTicketCache(),
	}
	a.bots[robot.MemberID] = entry
	if conf.AccountID != "" {
		a.accountIdx[conf.AccountID] = robot.MemberID
	}
	go a.pollLoop(pollCtx, entry)

	log.Info("weixin adapter: registered robot=%s accountID=%s", robot.MemberID, conf.AccountID)
}

func (a *Adapter) Remove(ctx context.Context, robotID string) {
	a.removeBot(robotID)
}

func (a *Adapter) Shutdown() {
	close(a.stopCh)
	a.mu.Lock()
	defer a.mu.Unlock()
	for id := range a.bots {
		a.removeBotLocked(id)
	}
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
	entry.cancelFn()
	if entry.accountID != "" {
		delete(a.accountIdx, entry.accountID)
	}
	delete(a.bots, robotID)
}

func resolveBaseURL(conf *robottypes.WeixinConfig) string {
	if conf.APIHost != "" {
		return conf.APIHost
	}
	if conf.BaseURL != "" {
		return conf.BaseURL
	}
	return weixinapi.DefaultBaseURL()
}

func resolveCDNBaseURL(conf *robottypes.WeixinConfig) string {
	if conf.CDNBaseURL != "" {
		return conf.CDNBaseURL
	}
	return weixinapi.DefaultCDNBaseURL()
}

func extractConfig(robot *robottypes.Robot) *robottypes.WeixinConfig {
	if robot.Config == nil || robot.Config.Integrations == nil {
		return nil
	}
	return robot.Config.Integrations.Weixin
}
