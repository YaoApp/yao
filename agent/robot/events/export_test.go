package events

import (
	"context"
	"net/http"

	eventtypes "github.com/yaoapp/yao/event/types"
)

// TestHandler wraps robotHandler for external test access.
type TestHandler struct {
	h *robotHandler
}

// NewTestHandler creates a robotHandler for testing.
func NewTestHandler() *TestHandler {
	return &TestHandler{
		h: &robotHandler{
			httpClient: http.DefaultClient,
		},
	}
}

// Handle delegates to the internal robotHandler.Handle.
func (th *TestHandler) Handle(ctx context.Context, ev *eventtypes.Event, resp chan<- eventtypes.Result) {
	th.h.Handle(ctx, ev, resp)
}

// Shutdown delegates to the internal robotHandler.Shutdown.
func (th *TestHandler) Shutdown(ctx context.Context) error {
	return th.h.Shutdown(ctx)
}
