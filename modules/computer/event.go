package computer

import (
	"context"

	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/event"
	eventtypes "github.com/yaoapp/yao/event/types"
	"github.com/yaoapp/yao/modules/computer/types"
)

const (
	computerEventPrefix = "computer"

	EventComputerCreated = "computer.created"
	EventComputerStatus  = "computer.status"
)

// computerEventHandler keeps the event bus prefix alive. Computer events are
// consumed by the /v1/events subscriber; the handler itself has no side effect.
type computerEventHandler struct{}

func init() {
	event.Register(computerEventPrefix, &computerEventHandler{})
}

func (h *computerEventHandler) Handle(context.Context, *eventtypes.Event, chan<- eventtypes.Result) {}

func (h *computerEventHandler) Shutdown(context.Context) error { return nil }

func publishComputerEvent(ctx context.Context, eventType string, computer *types.Computer) {
	if computer == nil || computer.ID == "" {
		return
	}
	payload := map[string]any{
		"computer_id":      computer.ID,
		"status":           computer.Status,
		"__yao_created_by": computer.UserID,
		"__yao_team_id":    computer.TeamID,
	}
	if _, err := event.Push(ctx, eventType, payload); err != nil {
		// The event bus may not be started in unit tests or during early boot.
		log.Debug("computer event %s skipped: %v", eventType, err)
	}
}

func publishComputerCreated(ctx context.Context, computer *types.Computer) {
	publishComputerEvent(ctx, EventComputerCreated, computer)
}

func publishComputerStatus(ctx context.Context, computer *types.Computer) {
	publishComputerEvent(ctx, EventComputerStatus, computer)
}
