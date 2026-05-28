package robot

import (
	"context"

	agentcontext "github.com/yaoapp/yao/agent/context"
	robotapi "github.com/yaoapp/yao/agent/robot/api"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	toolrobot "github.com/yaoapp/yao/tools/robot"
)

func init() {
	toolrobot.ListAllRobotsFn = bridgeListAllRobots
	toolrobot.GetRobotResponseFn = bridgeGetRobotResponse
	toolrobot.GetRobotStatusFn = bridgeGetRobotStatus
	toolrobot.CreateRobotFn = bridgeCreateRobot
	toolrobot.UpdateRobotFn = bridgeUpdateRobot
	toolrobot.TriggerFn = bridgeTrigger
	toolrobot.StopExecutionFn = bridgeStopExecution
	toolrobot.ListExecutionsFn = bridgeListExecutions
	toolrobot.GetExecutionFn = bridgeGetExecution
	toolrobot.ListResultsFn = bridgeListResults
}

func buildRobotContext(ctx context.Context, info *toolrobot.AuthInfo) *robottypes.Context {
	var auth *oauthtypes.AuthorizedInfo
	if info != nil {
		auth = &oauthtypes.AuthorizedInfo{
			UserID:   info.UserID,
			TeamID:   info.TeamID,
			TenantID: info.TenantID,
		}
	}
	return robottypes.NewContext(ctx, auth)
}

func bridgeListAllRobots(ctx context.Context, info *toolrobot.AuthInfo, query *toolrobot.ListQuery) (*toolrobot.ListResult, error) {
	rctx := buildRobotContext(ctx, info)
	q := &robotapi.ListQuery{}
	if query != nil {
		q.TeamID = query.TeamID
		q.Keywords = query.Keywords
		q.Page = query.Page
		q.PageSize = query.PageSize
		if query.Status != "" {
			q.Status = robottypes.RobotStatus(query.Status)
		}
	}

	result, err := robotapi.ListAllRobots(rctx, q)
	if err != nil {
		return nil, err
	}

	items := make([]toolrobot.RobotSummary, 0, len(result.Data))
	for _, r := range result.Data {
		items = append(items, toolrobot.RobotSummary{
			MemberID:       r.MemberID,
			DisplayName:    r.DisplayName,
			Bio:            r.Bio,
			Status:         string(r.Status),
			AutonomousMode: r.AutonomousMode,
			Running:        r.RunningCount(),
		})
	}

	return &toolrobot.ListResult{
		Data:     items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	}, nil
}

func bridgeGetRobotResponse(ctx context.Context, info *toolrobot.AuthInfo, memberID string) (*toolrobot.RobotResponse, error) {
	rctx := buildRobotContext(ctx, info)
	resp, err := robotapi.GetRobotResponse(rctx, memberID)
	if err != nil {
		return nil, err
	}
	return &toolrobot.RobotResponse{
		Data:         resp,
		YaoTeamID:    resp.YaoTeamID,
		YaoCreatedBy: resp.YaoCreatedBy,
	}, nil
}

func bridgeGetRobotStatus(ctx context.Context, info *toolrobot.AuthInfo, memberID string) (*toolrobot.RobotState, error) {
	rctx := buildRobotContext(ctx, info)
	state, err := robotapi.GetRobotStatus(rctx, memberID)
	if err != nil {
		return nil, err
	}

	var lastRun, nextRun string
	if state.LastRun != nil {
		lastRun = state.LastRun.Format("2006-01-02T15:04:05Z07:00")
	}
	if state.NextRun != nil {
		nextRun = state.NextRun.Format("2006-01-02T15:04:05Z07:00")
	}

	return &toolrobot.RobotState{
		MemberID:     state.MemberID,
		TeamID:       state.TeamID,
		DisplayName:  state.DisplayName,
		Bio:          state.Bio,
		Status:       string(state.Status),
		Running:      state.Running,
		MaxRunning:   state.MaxRunning,
		RunningIDs:   state.RunningIDs,
		LastRun:      lastRun,
		NextRun:      nextRun,
		YaoTeamID:    state.YaoTeamID,
		YaoCreatedBy: state.YaoCreatedBy,
	}, nil
}

// convertToolConfig converts tool-layer whitelist config to types.Config.
func convertToolConfig(tc *toolrobot.ToolRobotConfig) *robottypes.Config {
	if tc == nil {
		return nil
	}

	cfg := &robottypes.Config{
		DefaultLocale: tc.DefaultLocale,
	}

	if tc.Identity != nil {
		cfg.Identity = &robottypes.Identity{
			Role:   tc.Identity.Role,
			Duties: tc.Identity.Duties,
			Rules:  tc.Identity.Rules,
		}
	}

	if tc.Quota != nil {
		cfg.Quota = &robottypes.Quota{
			Max:      tc.Quota.Max,
			Queue:    tc.Quota.Queue,
			Priority: tc.Quota.Priority,
		}
	}

	if tc.Clock != nil {
		cfg.Clock = &robottypes.Clock{
			Mode:    robottypes.ClockMode(tc.Clock.Mode),
			Times:   tc.Clock.Times,
			Days:    tc.Clock.Days,
			Every:   tc.Clock.Every,
			TZ:      tc.Clock.TZ,
			Timeout: tc.Clock.Timeout,
		}
	}

	if tc.Triggers != nil {
		cfg.Triggers = &robottypes.Triggers{}
		if tc.Triggers.Clock != nil {
			cfg.Triggers.Clock = &robottypes.TriggerSwitch{
				Enabled: tc.Triggers.Clock.Enabled,
				Actions: tc.Triggers.Clock.Actions,
			}
		}
		if tc.Triggers.Intervene != nil {
			cfg.Triggers.Intervene = &robottypes.TriggerSwitch{
				Enabled: tc.Triggers.Intervene.Enabled,
				Actions: tc.Triggers.Intervene.Actions,
			}
		}
		if tc.Triggers.Event != nil {
			cfg.Triggers.Event = &robottypes.TriggerSwitch{
				Enabled: tc.Triggers.Event.Enabled,
				Actions: tc.Triggers.Event.Actions,
			}
		}
	}

	if tc.Executor != nil {
		cfg.Executor = &robottypes.ExecutorConfig{
			Mode:        robottypes.ExecutorMode(tc.Executor.Mode),
			MaxDuration: tc.Executor.MaxDuration,
		}
	}

	return cfg
}

func bridgeCreateRobot(ctx context.Context, info *toolrobot.AuthInfo, req *toolrobot.CreateRequest) (*toolrobot.RobotResponse, error) {
	effectiveTeamID := info.TeamID
	if effectiveTeamID == "" {
		effectiveTeamID = info.UserID
	}

	createReq := &robotapi.CreateRobotRequest{
		TeamID:         effectiveTeamID,
		DisplayName:    req.DisplayName,
		Bio:            req.Bio,
		SystemPrompt:   req.SystemPrompt,
		Workspace:      req.Workspace,
		AutonomousMode: req.AutonomousMode,
		AuthScope: &robotapi.AuthScope{
			CreatedBy: info.UserID,
			TeamID:    effectiveTeamID,
			TenantID:  info.TenantID,
		},
	}

	if len(req.Agents) > 0 {
		agents := make([]interface{}, len(req.Agents))
		for i, a := range req.Agents {
			agents[i] = a
		}
		createReq.Agents = agents
	}

	if req.RobotConfig != nil {
		createReq.RobotConfig = convertToolConfig(req.RobotConfig)
	}

	rctx := buildRobotContext(ctx, info)
	resp, err := robotapi.CreateRobot(rctx, createReq)
	if err != nil {
		return nil, err
	}
	return &toolrobot.RobotResponse{
		Data:         resp,
		YaoTeamID:    resp.YaoTeamID,
		YaoCreatedBy: resp.YaoCreatedBy,
	}, nil
}

func bridgeUpdateRobot(ctx context.Context, info *toolrobot.AuthInfo, memberID string, req *toolrobot.UpdateRequest) (*toolrobot.RobotResponse, error) {
	updateReq := &robotapi.UpdateRobotRequest{
		DisplayName:    req.DisplayName,
		Bio:            req.Bio,
		SystemPrompt:   req.SystemPrompt,
		Workspace:      req.Workspace,
		AutonomousMode: req.AutonomousMode,
	}

	if len(req.Agents) > 0 {
		agents := make([]interface{}, len(req.Agents))
		for i, a := range req.Agents {
			agents[i] = a
		}
		updateReq.Agents = agents
	}

	if req.RobotConfig != nil {
		updateReq.RobotConfig = convertToolConfig(req.RobotConfig)
	}

	rctx := buildRobotContext(ctx, info)
	resp, err := robotapi.UpdateRobot(rctx, memberID, updateReq)
	if err != nil {
		return nil, err
	}
	return &toolrobot.RobotResponse{
		Data:         resp,
		YaoTeamID:    resp.YaoTeamID,
		YaoCreatedBy: resp.YaoCreatedBy,
	}, nil
}

func bridgeTrigger(ctx context.Context, info *toolrobot.AuthInfo, memberID string, req *toolrobot.TriggerRequest) (*toolrobot.TriggerResult, error) {
	triggerReq := &robotapi.TriggerRequest{
		Type:      robottypes.TriggerType(req.Type),
		Source:    robottypes.EventSource(req.Source),
		EventType: req.EventType,
		Data:      req.Data,
	}

	if len(req.Messages) > 0 {
		msgs := make([]agentcontext.Message, len(req.Messages))
		for i, m := range req.Messages {
			msgs[i] = agentcontext.Message{
				Role:    agentcontext.MessageRole(m.Role),
				Content: m.Content,
			}
		}
		triggerReq.Messages = msgs
	}

	rctx := buildRobotContext(ctx, info)
	result, err := robotapi.Trigger(rctx, memberID, triggerReq)
	if err != nil {
		return nil, err
	}
	return &toolrobot.TriggerResult{
		ExecutionID: result.ExecutionID,
		Accepted:    result.Accepted,
		Message:     result.Message,
	}, nil
}

func bridgeStopExecution(ctx context.Context, info *toolrobot.AuthInfo, execID string) error {
	rctx := buildRobotContext(ctx, info)
	return robotapi.StopExecution(rctx, execID)
}

func bridgeListExecutions(ctx context.Context, info *toolrobot.AuthInfo, memberID string, query *toolrobot.ExecutionQuery) (*toolrobot.ExecutionResult, error) {
	rctx := buildRobotContext(ctx, info)
	q := &robotapi.ExecutionQuery{}
	if query != nil {
		if query.Status != "" {
			q.Status = robottypes.ExecStatus(query.Status)
		}
		q.Page = query.Page
		q.PageSize = query.PageSize
	}

	result, err := robotapi.ListExecutions(rctx, memberID, q)
	if err != nil {
		return nil, err
	}
	return &toolrobot.ExecutionResult{
		Data:     result.Data,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	}, nil
}

func bridgeGetExecution(ctx context.Context, info *toolrobot.AuthInfo, execID string) (*toolrobot.ExecutionDetail, error) {
	rctx := buildRobotContext(ctx, info)
	exec, err := robotapi.GetExecution(rctx, execID)
	if err != nil {
		return nil, err
	}
	return &toolrobot.ExecutionDetail{Data: exec}, nil
}

func bridgeListResults(ctx context.Context, info *toolrobot.AuthInfo, memberID string, query *toolrobot.ResultQuery) (*toolrobot.ResultListResponse, error) {
	rctx := buildRobotContext(ctx, info)
	q := &robotapi.ResultQuery{}
	if query != nil {
		q.Page = query.Page
		q.PageSize = query.PageSize
	}
	result, err := robotapi.ListResults(rctx, memberID, q)
	if err != nil {
		return nil, err
	}
	return &toolrobot.ResultListResponse{
		Data:     result.Data,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	}, nil
}
