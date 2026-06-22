package task

import (
	"github.com/yaoapp/yao/agent/assistant"
	agentcontext "github.com/yaoapp/yao/agent/context"
	tasksvc "github.com/yaoapp/yao/agent/task"
	tasktools "github.com/yaoapp/yao/tools/task"
)

func init() {
	tasktools.FnList = tasksvc.List
	tasktools.FnCreate = tasksvc.Create
	tasktools.FnMove = tasksvc.Move
	tasktools.FnRun = tasksvc.Run
	tasktools.FnStop = tasksvc.Stop

	tasksvc.AssistantStreamFn = func(assistantID string, ctx *agentcontext.Context, msgs []agentcontext.Message, opts ...*agentcontext.Options) (*agentcontext.Response, error) {
		ast, err := assistant.Get(assistantID)
		if err != nil {
			return nil, err
		}
		return ast.Stream(ctx, msgs, opts...)
	}
}
