package robot

import (
	"context"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/robot/api"
	"github.com/yaoapp/yao/agent/robot/types"
)

func init() {
	process.RegisterGroup("robot", map[string]process.Handler{
		"get":             processGet,
		"list":            processList,
		"status":          processStatus,
		"executions":      processExecutions,
		"execution":       processExecution,
		"updateChatTitle": processUpdateChatTitle,
	})
}

// processGet handles robot.Get(memberID).
// args[0]: memberID string
func processGet(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	memberID := p.ArgsString(0)
	ctx := types.NewContext(context.Background(), nil)
	result, err := api.GetRobotResponse(ctx, memberID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return result
}

// processList handles robot.List(filter?).
// args[0]: optional filter map with page, pagesize, status, search (keywords)
func processList(p *process.Process) interface{} {
	p.ValidateArgNums(0)
	ctx := types.NewContext(context.Background(), nil)
	filter := &api.ListQuery{}
	if p.NumOfArgs() > 0 {
		raw := p.ArgsMap(0)
		if v, ok := raw["page"]; ok {
			filter.Page = toInt(v)
		}
		if v, ok := raw["pagesize"]; ok {
			filter.PageSize = toInt(v)
		}
		if v, ok := raw["status"]; ok {
			filter.Status = types.RobotStatus(toString(v))
		}
		if v, ok := raw["search"]; ok {
			filter.Keywords = toString(v)
		}
		if v, ok := raw["team_id"]; ok {
			filter.TeamID = toString(v)
		}
	}
	result, err := api.ListAllRobots(ctx, filter)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return result
}

// processStatus handles robot.Status(memberID).
// args[0]: memberID string
func processStatus(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	memberID := p.ArgsString(0)
	ctx := types.NewContext(context.Background(), nil)
	result, err := api.GetRobotStatus(ctx, memberID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return result
}

// processExecutions handles robot.Executions(memberID, filter?).
// args[0]: memberID string; args[1]: optional filter map
func processExecutions(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	memberID := p.ArgsString(0)
	ctx := types.NewContext(context.Background(), nil)
	filter := &api.ExecutionQuery{}
	if p.NumOfArgs() > 1 {
		raw := p.ArgsMap(1)
		if v, ok := raw["page"]; ok {
			filter.Page = toInt(v)
		}
		if v, ok := raw["pagesize"]; ok {
			filter.PageSize = toInt(v)
		}
		if v, ok := raw["status"]; ok {
			filter.Status = types.ExecStatus(toString(v))
		}
		if v, ok := raw["trigger"]; ok {
			filter.Trigger = types.TriggerType(toString(v))
		}
	}
	result, err := api.ListExecutions(ctx, memberID, filter)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return result
}

// processExecution handles robot.Execution(memberID, executionID).
// args[0]: memberID string; args[1]: executionID string
func processExecution(p *process.Process) interface{} {
	p.ValidateArgNums(2)
	memberID := p.ArgsString(0)
	executionID := p.ArgsString(1)
	ctx := types.NewContext(context.Background(), nil)
	result, err := api.GetExecutionStatus(ctx, executionID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	_ = memberID // reserved for future permission scoping
	return result
}

// processUpdateChatTitle handles robot.UpdateChatTitle(chatID, title).
// args[0]: chatID string; args[1]: title string
func processUpdateChatTitle(p *process.Process) interface{} {
	p.ValidateArgNums(2)
	chatID := p.ArgsString(0)
	title := p.ArgsString(1)

	chatStore := assistant.GetChatStore()
	if chatStore == nil {
		exception.New("chat store not available", 500).Throw()
	}

	if err := chatStore.UpdateChat(chatID, map[string]interface{}{"title": title}); err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return nil
}

func toInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
