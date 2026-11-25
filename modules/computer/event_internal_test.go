package computer

import (
	"context"
	"testing"

	eventtypes "github.com/yaoapp/yao/event/types"
	"github.com/yaoapp/yao/modules/computer/types"
)

func TestComputerEventHandlerIsNoop(t *testing.T) {
	h := &computerEventHandler{}
	resp := make(chan eventtypes.Result, 1)
	h.Handle(context.Background(), &eventtypes.Event{Type: EventComputerStatus}, resp)
	select {
	case <-resp:
		t.Fatal("push handler must not write a response")
	default:
	}
	if err := h.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

func TestPublishComputerEventSkipsIncompleteComputers(t *testing.T) {
	publishComputerCreated(context.Background(), nil)
	publishComputerStatus(context.Background(), &types.Computer{})
	publishComputerStatus(context.Background(), &types.Computer{ID: "cmp-1", Status: types.StatusPending})
}
