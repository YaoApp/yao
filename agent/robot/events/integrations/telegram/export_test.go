package telegram

import (
	"context"

	robottypes "github.com/yaoapp/yao/agent/robot/types"
	tgapi "github.com/yaoapp/yao/integrations/telegram"
)

// TestAdapter wraps the Adapter for external test access.
type TestAdapter struct {
	a *Adapter
}

// NewTestAdapter creates a test adapter without starting the poll loop.
func NewTestAdapter() *TestAdapter {
	return &TestAdapter{
		a: &Adapter{
			bots:   make(map[string]*botEntry),
			appIdx: make(map[string]string),
			dedup:  newDedupStore(),
			stopCh: make(chan struct{}),
		},
	}
}

// NewTestAdapterWithBot creates a test adapter with a pre-configured bot entry.
func NewTestAdapterWithBot(memberID, appID, token, host string) *TestAdapter {
	ta := NewTestAdapter()
	var opts []tgapi.BotOption
	if host != "" {
		opts = append(opts, tgapi.WithAPIBase(host))
	}
	ta.a.bots[memberID] = &botEntry{
		robotID: memberID,
		appID:   appID,
		bot:     tgapi.NewBot(token, "", opts...),
	}
	return ta
}

// Close cleans up the test adapter.
func (ta *TestAdapter) Close() {
	close(ta.a.stopCh)
}

// Apply delegates to the internal adapter.
func (ta *TestAdapter) Apply(ctx context.Context, robot *robottypes.Robot) {
	ta.a.Apply(ctx, robot)
}

// Remove delegates to the internal adapter.
func (ta *TestAdapter) Remove(ctx context.Context, robotID string) {
	ta.a.Remove(ctx, robotID)
}

// GetBot returns the bot for a given robot ID, or nil if not found.
func (ta *TestAdapter) GetBot(robotID string) *tgapi.Bot {
	ta.a.mu.RLock()
	defer ta.a.mu.RUnlock()
	entry, ok := ta.a.bots[robotID]
	if !ok {
		return nil
	}
	return entry.bot
}

// GetOffset returns the current polling offset for a robot.
func (ta *TestAdapter) GetOffset(robotID string) int64 {
	ta.a.mu.RLock()
	defer ta.a.mu.RUnlock()
	entry, ok := ta.a.bots[robotID]
	if !ok {
		return 0
	}
	return entry.offset
}

// BotCount returns the number of registered bots.
func (ta *TestAdapter) BotCount() int {
	ta.a.mu.RLock()
	defer ta.a.mu.RUnlock()
	return len(ta.a.bots)
}

// MarkSeen delegates to dedup.
func (ta *TestAdapter) MarkSeen(key string) bool {
	return ta.a.dedup.markSeen(key)
}

// PollAll triggers one poll cycle.
func (ta *TestAdapter) PollAll() {
	ta.a.pollAll()
}

// HandleMessagesForTest processes messages for a specific bot entry.
func (ta *TestAdapter) HandleMessagesForTest(ctx context.Context, robotID string, msgs []*tgapi.ConvertedMessage) {
	ta.a.mu.RLock()
	entry := ta.a.bots[robotID]
	ta.a.mu.RUnlock()
	if entry == nil {
		return
	}

	grouped := groupByChatID(msgs)
	for _, chatMsgs := range grouped {
		ta.a.handleMessages(ctx, entry, chatMsgs)
	}
}
