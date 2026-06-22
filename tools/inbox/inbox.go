package inbox

import (
	"context"

	"github.com/yaoapp/gou/process"
	inboxsvc "github.com/yaoapp/yao/agent/inbox"
)

// Fn pointers injected by openapi/agent/inbox/tools_bridge.go init()
var (
	FnList func(ctx context.Context, auth *process.AuthorizedInfo, q *inboxsvc.ListQuery) (*inboxsvc.ListResult, error)
	FnRead func(ctx context.Context, auth *process.AuthorizedInfo, mailID string) error
)
