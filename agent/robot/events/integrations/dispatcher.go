package integrations

import (
	"context"
	"fmt"

	agentcontext "github.com/yaoapp/yao/agent/context"
	robotcache "github.com/yaoapp/yao/agent/robot/cache"
	events "github.com/yaoapp/yao/agent/robot/events"
	"github.com/yaoapp/yao/agent/robot/logger"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/event"
	eventtypes "github.com/yaoapp/yao/event/types"
)

var log = logger.New("dispatcher")

// Adapter is the interface each platform adapter implements.
type Adapter interface {
	Apply(ctx context.Context, robot *robottypes.Robot)
	Remove(ctx context.Context, robotID string)
	Reply(ctx context.Context, msg *agentcontext.Message, metadata *events.MessageMetadata) error
	Shutdown()
}

// Dispatcher distributes Robot integration configs to platform adapters.
type Dispatcher struct {
	robotCache *robotcache.Cache
	adapters   map[string]Adapter // key matches Integrations field: "telegram", "discord", etc.
	stopCh     chan struct{}
	subID      string
}

// NewDispatcher creates a Dispatcher.
// Each adapter has a fixed key matching the field name in robottypes.Integrations.
func NewDispatcher(cache *robotcache.Cache, adapters map[string]Adapter) *Dispatcher {
	return &Dispatcher{
		robotCache: cache,
		adapters:   adapters,
		stopCh:     make(chan struct{}),
	}
}

// Start loads all robots and subscribes to config change events.
func (d *Dispatcher) Start(ctx context.Context) error {
	d.loadAll(ctx)

	events.RegisterReplyFunc(d.reply)

	ch := make(chan *eventtypes.Event, 256)
	d.subID = event.Subscribe("robot.config.*", ch)
	go d.watch(ctx, ch)

	log.Info("integration dispatcher: started with %d adapters", len(d.adapters))
	return nil
}

// reply routes a reply to the correct adapter based on channel.
// When channel is empty (e.g. delivery), broadcasts to all adapters.
func (d *Dispatcher) reply(ctx context.Context, msg *agentcontext.Message, metadata *events.MessageMetadata) error {
	if metadata == nil {
		return fmt.Errorf("no metadata in reply")
	}

	if metadata.Channel != "" {
		adapter, ok := d.adapters[metadata.Channel]
		if !ok {
			return fmt.Errorf("no adapter for channel: %s", metadata.Channel)
		}
		return adapter.Reply(ctx, msg, metadata)
	}

	var lastErr error
	for name, adapter := range d.adapters {
		if err := adapter.Reply(ctx, msg, metadata); err != nil {
			log.Error("dispatcher reply: broadcast to %s failed: %v", name, err)
			lastErr = err
		}
	}
	return lastErr
}

// Stop unsubscribes from events and shuts down all adapters.
func (d *Dispatcher) Stop() {
	close(d.stopCh)
	if d.subID != "" {
		event.Unsubscribe(d.subID)
	}
	for name, adapter := range d.adapters {
		adapter.Shutdown()
		log.Info("integration dispatcher: adapter %s shutdown", name)
	}
	log.Info("integration dispatcher: stopped")
}

func (d *Dispatcher) loadAll(ctx context.Context) {
	robots := d.robotCache.ListAll()
	count := 0
	for _, robot := range robots {
		if robot.Config != nil && robot.Config.Integrations != nil && len(parseIntegrations(robot.Config.Integrations)) > 0 {
			d.apply(ctx, robot)
			count++
		}
	}
	log.Info("integration dispatcher: initial load complete, %d robots with integrations", count)
}

// apply parses which integrations the robot has configured,
// and calls the matching adapter for each one.
func (d *Dispatcher) apply(ctx context.Context, robot *robottypes.Robot) {
	if robot.Config == nil || robot.Config.Integrations == nil {
		return
	}
	for _, key := range parseIntegrations(robot.Config.Integrations) {
		if adapter, ok := d.adapters[key]; ok {
			adapter.Apply(ctx, robot)
		}
	}
}

func (d *Dispatcher) remove(ctx context.Context, robotID string) {
	for _, adapter := range d.adapters {
		adapter.Remove(ctx, robotID)
	}
}

// parseIntegrations returns the keys of integrations present in the config.
func parseIntegrations(intg *robottypes.Integrations) []string {
	var keys []string
	if intg.Telegram != nil {
		keys = append(keys, "telegram")
	}
	if intg.Feishu != nil {
		keys = append(keys, "feishu")
	}
	if intg.DingTalk != nil {
		keys = append(keys, "dingtalk")
	}
	if intg.Discord != nil {
		keys = append(keys, "discord")
	}
	if intg.Weixin != nil {
		keys = append(keys, "weixin")
	}
	return keys
}

func (d *Dispatcher) watch(ctx context.Context, ch <-chan *eventtypes.Event) {
	for {
		select {
		case <-d.stopCh:
			return
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			d.dispatch(ctx, ev)
		}
	}
}

func (d *Dispatcher) dispatch(ctx context.Context, ev *eventtypes.Event) {
	var payload events.RobotConfigPayload
	if err := ev.Should(&payload); err != nil {
		log.Error("integration dispatcher: invalid config event: %v", err)
		return
	}

	switch ev.Type {
	case events.RobotConfigCreated, events.RobotConfigUpdated:
		robot := d.robotCache.Get(payload.MemberID)
		if robot == nil {
			rCtx := robottypes.NewContext(ctx, nil)
			loaded, err := d.robotCache.LoadByID(rCtx, payload.MemberID)
			if err != nil {
				log.Warn("integration dispatcher: failed to load robot from DB member=%s: %v", payload.MemberID, err)
				return
			}
			d.robotCache.Add(loaded)
			robot = loaded
			log.Info("integration dispatcher: loaded robot from DB member=%s", payload.MemberID)
		}
		d.apply(ctx, robot)

	case events.RobotConfigDeleted:
		d.remove(ctx, payload.MemberID)
	}
}
