package integrations

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	agentcontext "github.com/yaoapp/yao/agent/context"
	robotcache "github.com/yaoapp/yao/agent/robot/cache"
	events "github.com/yaoapp/yao/agent/robot/events"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/event"
	eventtypes "github.com/yaoapp/yao/event/types"
)

// mockAdapter records Apply/Remove calls for assertions.
type mockAdapter struct {
	mu      sync.Mutex
	applied []*robottypes.Robot
	removed []string
}

func (m *mockAdapter) Apply(ctx context.Context, robot *robottypes.Robot) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.applied = append(m.applied, robot)
}

func (m *mockAdapter) Remove(ctx context.Context, robotID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.removed = append(m.removed, robotID)
}

func (m *mockAdapter) Reply(ctx context.Context, msg *agentcontext.Message, metadata *events.MessageMetadata) error {
	return nil
}

func (m *mockAdapter) Shutdown() {}

func (m *mockAdapter) getApplied() []*robottypes.Robot {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]*robottypes.Robot, len(m.applied))
	copy(cp, m.applied)
	return cp
}

func (m *mockAdapter) getRemoved() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]string, len(m.removed))
	copy(cp, m.removed)
	return cp
}

// noopHandler satisfies event.Handler so we can register "robot" prefix for Push/Call.
type noopHandler struct{}

func (h *noopHandler) Handle(ctx context.Context, ev *eventtypes.Event, resp chan<- eventtypes.Result) {
	if ev.IsCall {
		resp <- eventtypes.Result{}
	}
}
func (h *noopHandler) Shutdown(ctx context.Context) error { return nil }

var eventOnce sync.Once

func setupEventBus(t *testing.T) {
	t.Helper()
	eventOnce.Do(func() {
		event.Register("robot", &noopHandler{})
	})
	if err := event.Start(); err != nil && err != event.ErrAlreadyStart {
		t.Fatalf("event.Start: %v", err)
	}
	t.Cleanup(func() { _ = event.Stop(context.Background()) })
}

func newRobot(memberID, teamID string, intg *robottypes.Integrations) *robottypes.Robot {
	return &robottypes.Robot{
		MemberID:       memberID,
		TeamID:         teamID,
		AutonomousMode: true,
		Config: &robottypes.Config{
			Integrations: intg,
		},
	}
}

func TestLoadAll_OnlyTelegramConfigured(t *testing.T) {
	setupEventBus(t)
	cache := robotcache.New()

	tgRobot := newRobot("r-tg", "team1", &robottypes.Integrations{
		Telegram: &robottypes.TelegramConfig{Enabled: true, BotToken: "tok"},
	})
	noIntgRobot := newRobot("r-plain", "team1", nil)
	cache.Add(tgRobot)
	cache.Add(noIntgRobot)

	tgAdapter := &mockAdapter{}
	d := NewDispatcher(cache, map[string]Adapter{"telegram": tgAdapter})

	require.NoError(t, d.Start(context.Background()))
	defer d.Stop()

	applied := tgAdapter.getApplied()
	assert.Len(t, applied, 1)
	assert.Equal(t, "r-tg", applied[0].MemberID)
}

func TestLoadAll_NoIntegrations(t *testing.T) {
	setupEventBus(t)
	cache := robotcache.New()

	cache.Add(newRobot("r1", "team1", nil))
	cache.Add(&robottypes.Robot{MemberID: "r2", TeamID: "team1"})

	tgAdapter := &mockAdapter{}
	d := NewDispatcher(cache, map[string]Adapter{"telegram": tgAdapter})

	require.NoError(t, d.Start(context.Background()))
	defer d.Stop()

	assert.Empty(t, tgAdapter.getApplied())
}

func TestLoadAll_MultipleAdapters(t *testing.T) {
	setupEventBus(t)
	cache := robotcache.New()

	// Only Telegram configured, no Discord
	robot := newRobot("r-multi", "team1", &robottypes.Integrations{
		Telegram: &robottypes.TelegramConfig{Enabled: true, BotToken: "tok"},
	})
	cache.Add(robot)

	tgAdapter := &mockAdapter{}
	discordAdapter := &mockAdapter{}
	d := NewDispatcher(cache, map[string]Adapter{
		"telegram": tgAdapter,
		"discord":  discordAdapter,
	})

	require.NoError(t, d.Start(context.Background()))
	defer d.Stop()

	assert.Len(t, tgAdapter.getApplied(), 1)
	assert.Empty(t, discordAdapter.getApplied(), "discord adapter should not be called")
}

func TestConfigCreated_TriggersApply(t *testing.T) {
	setupEventBus(t)
	cache := robotcache.New()

	tgAdapter := &mockAdapter{}
	d := NewDispatcher(cache, map[string]Adapter{"telegram": tgAdapter})

	require.NoError(t, d.Start(context.Background()))
	defer d.Stop()

	assert.Empty(t, tgAdapter.getApplied())

	// Simulate: robot created with Telegram config, added to cache, event pushed
	robot := newRobot("r-new", "team1", &robottypes.Integrations{
		Telegram: &robottypes.TelegramConfig{Enabled: true, BotToken: "new-tok"},
	})
	cache.Add(robot)
	event.Push(context.Background(), events.RobotConfigCreated, events.RobotConfigPayload{
		MemberID: "r-new", TeamID: "team1",
	})

	assert.Eventually(t, func() bool {
		return len(tgAdapter.getApplied()) == 1
	}, 2*time.Second, 50*time.Millisecond)

	assert.Equal(t, "r-new", tgAdapter.getApplied()[0].MemberID)
}

func TestConfigUpdated_TriggersApply(t *testing.T) {
	setupEventBus(t)
	cache := robotcache.New()

	robot := newRobot("r-upd", "team1", &robottypes.Integrations{
		Telegram: &robottypes.TelegramConfig{Enabled: true, BotToken: "old-tok"},
	})
	cache.Add(robot)

	tgAdapter := &mockAdapter{}
	d := NewDispatcher(cache, map[string]Adapter{"telegram": tgAdapter})

	require.NoError(t, d.Start(context.Background()))
	defer d.Stop()

	// Initial load
	assert.Len(t, tgAdapter.getApplied(), 1)

	// Update config in cache
	robot.Config.Integrations.Telegram.BotToken = "new-tok"
	event.Push(context.Background(), events.RobotConfigUpdated, events.RobotConfigPayload{
		MemberID: "r-upd", TeamID: "team1",
	})

	assert.Eventually(t, func() bool {
		return len(tgAdapter.getApplied()) == 2
	}, 2*time.Second, 50*time.Millisecond)

	assert.Equal(t, "new-tok", tgAdapter.getApplied()[1].Config.Integrations.Telegram.BotToken)
}

func TestConfigDeleted_TriggersRemove(t *testing.T) {
	setupEventBus(t)
	cache := robotcache.New()

	robot := newRobot("r-del", "team1", &robottypes.Integrations{
		Telegram: &robottypes.TelegramConfig{Enabled: true, BotToken: "tok"},
	})
	cache.Add(robot)

	tgAdapter := &mockAdapter{}
	d := NewDispatcher(cache, map[string]Adapter{"telegram": tgAdapter})

	require.NoError(t, d.Start(context.Background()))
	defer d.Stop()

	assert.Len(t, tgAdapter.getApplied(), 1)

	event.Push(context.Background(), events.RobotConfigDeleted, events.RobotConfigPayload{
		MemberID: "r-del", TeamID: "team1",
	})

	assert.Eventually(t, func() bool {
		return len(tgAdapter.getRemoved()) == 1
	}, 2*time.Second, 50*time.Millisecond)

	assert.Equal(t, "r-del", tgAdapter.getRemoved()[0])
}

func TestConfigCreated_RobotNotInCache(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	setupEventBus(t)
	cache := robotcache.New()

	tgAdapter := &mockAdapter{}
	d := NewDispatcher(cache, map[string]Adapter{"telegram": tgAdapter})

	require.NoError(t, d.Start(context.Background()))
	defer d.Stop()

	// Push event but don't add robot to cache — triggers LoadByID DB fallback
	event.Push(context.Background(), events.RobotConfigCreated, events.RobotConfigPayload{
		MemberID: "r-ghost", TeamID: "team1",
	})

	time.Sleep(200 * time.Millisecond)
	assert.Empty(t, tgAdapter.getApplied())
}

func TestParseIntegrations(t *testing.T) {
	tests := []struct {
		name     string
		intg     *robottypes.Integrations
		expected []string
	}{
		{"nil", nil, nil},
		{"empty", &robottypes.Integrations{}, nil},
		{"telegram only", &robottypes.Integrations{
			Telegram: &robottypes.TelegramConfig{Enabled: true},
		}, []string{"telegram"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.intg == nil {
				return
			}
			result := parseIntegrations(tt.intg)
			assert.Equal(t, tt.expected, result)
		})
	}
}
