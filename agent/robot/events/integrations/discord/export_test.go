package discord

import (
	"context"

	robottypes "github.com/yaoapp/yao/agent/robot/types"
	dcapi "github.com/yaoapp/yao/integrations/discord"
)

// TestAdapter wraps the Adapter for external test access.
type TestAdapter struct {
	a *Adapter
}

// NewTestAdapter creates a test adapter without starting background goroutines.
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
func NewTestAdapterWithBot(memberID, botToken, appID string) *TestAdapter {
	ta := NewTestAdapter()
	bot, _ := dcapi.NewBot(botToken, appID)
	ta.a.bots[memberID] = &botEntry{
		robotID: memberID,
		appID:   appID,
		bot:     bot,
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
func (ta *TestAdapter) GetBot(robotID string) *dcapi.Bot {
	ta.a.mu.RLock()
	defer ta.a.mu.RUnlock()
	entry, ok := ta.a.bots[robotID]
	if !ok {
		return nil
	}
	return entry.bot
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

// HandleMessagesForTest processes messages for a specific bot entry.
func (ta *TestAdapter) HandleMessagesForTest(ctx context.Context, robotID string, cms []*dcapi.ConvertedMessage) {
	ta.a.mu.RLock()
	entry := ta.a.bots[robotID]
	ta.a.mu.RUnlock()
	if entry == nil {
		return
	}
	ta.a.handleMessages(ctx, entry, cms)
}
