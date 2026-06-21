package board

import (
	"context"

	"github.com/yaoapp/gou/process"
	boardsvc "github.com/yaoapp/yao/agent/board"
)

// Fn pointers injected by openapi/agent/board/tools_bridge.go init()
var (
	FnList   func(ctx context.Context, auth *process.AuthorizedInfo, q *boardsvc.ListQuery) (*boardsvc.ListResult, error)
	FnCreate func(ctx context.Context, auth *process.AuthorizedInfo, req *boardsvc.CreateReq) (*boardsvc.Board, error)
)
