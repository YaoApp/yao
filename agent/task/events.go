package task

import (
	"context"

	"github.com/yaoapp/yao/event"
	eventtypes "github.com/yaoapp/yao/event/types"
)

func init() {
	event.Register("task", &kanbanHandler{})
	event.Register("board", &kanbanHandler{})
	event.Register("mail", &kanbanHandler{})
}

// kanbanHandler is a no-op handler that enables event.Push to reach subscribers.
// Push flow: getHandler -> smgr.notify -> pool.dispatch
// smgr.notify delivers to dynamic subscribers before handler dispatch.
type kanbanHandler struct{}

func (h *kanbanHandler) Handle(ctx context.Context, ev *eventtypes.Event, resp chan<- eventtypes.Result) {
	resp <- eventtypes.Result{}
}

func (h *kanbanHandler) Shutdown(ctx context.Context) error { return nil }
