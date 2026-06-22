package task

import (
	"context"

	"github.com/yaoapp/gou/process"
	tasksvc "github.com/yaoapp/yao/agent/task"
)

// Fn pointers injected by openapi/agent/task/tools_bridge.go init()
var (
	FnList   func(ctx context.Context, auth *process.AuthorizedInfo, q *tasksvc.ListQuery) (*tasksvc.ListResult, error)
	FnCreate func(ctx context.Context, auth *process.AuthorizedInfo, req *tasksvc.CreateReq) (*tasksvc.Task, error)
	FnMove   func(ctx context.Context, auth *process.AuthorizedInfo, chatID string, req *tasksvc.MoveReq) error
	FnRun    func(ctx context.Context, auth *process.AuthorizedInfo, chatID string, req *tasksvc.RunReq) (*tasksvc.RunResult, error)
	FnStop   func(ctx context.Context, auth *process.AuthorizedInfo, chatID string, force bool) error
)
