package api

import (
	"fmt"

	"github.com/yaoapp/yao/agent/robot/types"
)

// ==================== Trigger API ====================
// These functions handle robot execution triggers

// Trigger starts a robot execution with the specified trigger type and request
// This is the main entry point for triggering robot execution
func Trigger(ctx *types.Context, memberID string, req *TriggerRequest) (*TriggerResult, error) {
	if memberID == "" {
		return nil, fmt.Errorf("member_id is required")
	}
	if req == nil {
		return nil, fmt.Errorf("trigger request is required")
	}

	mgr, err := getManager()
	if err != nil {
		return nil, err
	}

	switch req.Type {
	case types.TriggerHuman:
		return triggerHuman(ctx, mgr, memberID, req)
	case types.TriggerEvent:
		return triggerEvent(ctx, mgr, memberID, req)
	case types.TriggerClock:
		return triggerManual(ctx, mgr, memberID, req)
	default:
		return nil, fmt.Errorf("invalid trigger type: %s", req.Type)
	}
}

// TriggerManual manually triggers a robot execution (for testing or debugging)
// This bypasses normal trigger validation and directly submits to the pool
func TriggerManual(ctx *types.Context, memberID string, triggerType types.TriggerType, data interface{}) (*TriggerResult, error) {
	if memberID == "" {
		return nil, fmt.Errorf("member_id is required")
	}

	mgr, err := getManager()
	if err != nil {
		return nil, err
	}

	execID, err := mgr.TriggerManual(ctx, memberID, triggerType, data)
	if err != nil {
		return &TriggerResult{
			Accepted: false,
			Message:  err.Error(),
		}, nil
	}

	return &TriggerResult{
		Accepted:    true,
		ExecutionID: execID,
		Message:     fmt.Sprintf("Manual trigger (%s) submitted", triggerType),
	}, nil
}

// Intervene processes a human intervention request
// Human intervention skips P0 (inspiration) and goes directly to P1 (goals)
func Intervene(ctx *types.Context, memberID string, req *TriggerRequest) (*TriggerResult, error) {
	if memberID == "" {
		return nil, fmt.Errorf("member_id is required")
	}
	if req == nil {
		return nil, fmt.Errorf("intervention request is required")
	}

	mgr, err := getManager()
	if err != nil {
		return nil, err
	}

	return triggerHuman(ctx, mgr, memberID, req)
}

// HandleEvent processes an event trigger request
// Event trigger skips P0 (inspiration) and goes directly to P1 (goals)
func HandleEvent(ctx *types.Context, memberID string, req *TriggerRequest) (*TriggerResult, error) {
	if memberID == "" {
		return nil, fmt.Errorf("member_id is required")
	}
	if req == nil {
		return nil, fmt.Errorf("event request is required")
	}

	mgr, err := getManager()
	if err != nil {
		return nil, err
	}

	return triggerEvent(ctx, mgr, memberID, req)
}

// ==================== Internal Trigger Functions ====================

// triggerHuman handles human intervention trigger
func triggerHuman(ctx *types.Context, mgr managerInterface, memberID string, req *TriggerRequest) (*TriggerResult, error) {
	// Build intervention request
	interveneReq := &types.InterveneRequest{
		MemberID:     memberID,
		TeamID:       ctx.TeamID(),
		Action:       req.Action,
		Messages:     req.Messages,
		PlanTime:     req.PlanAt,
		ExecutorMode: req.ExecutorMode,
	}

	// Call manager's Intervene
	result, err := mgr.Intervene(ctx, interveneReq)
	if err != nil {
		return &TriggerResult{
			Accepted: false,
			Message:  err.Error(),
		}, nil
	}

	return &TriggerResult{
		Accepted:    true,
		ExecutionID: result.ExecutionID,
		Message:     result.Message,
	}, nil
}

// triggerEvent handles event trigger
func triggerEvent(ctx *types.Context, mgr managerInterface, memberID string, req *TriggerRequest) (*TriggerResult, error) {
	// Build event request
	eventReq := &types.EventRequest{
		MemberID:     memberID,
		Source:       string(req.Source),
		EventType:    req.EventType,
		Data:         req.Data,
		ExecutorMode: req.ExecutorMode,
	}

	// Call manager's HandleEvent
	result, err := mgr.HandleEvent(ctx, eventReq)
	if err != nil {
		return &TriggerResult{
			Accepted: false,
			Message:  err.Error(),
		}, nil
	}

	return &TriggerResult{
		Accepted:    true,
		ExecutionID: result.ExecutionID,
		Message:     result.Message,
	}, nil
}

// triggerManual handles manual/clock trigger
func triggerManual(ctx *types.Context, mgr managerInterface, memberID string, req *TriggerRequest) (*TriggerResult, error) {
	// For clock trigger, pass clock context if available
	var data interface{}
	if req.Data != nil {
		data = req.Data
	}

	execID, err := mgr.TriggerManual(ctx, memberID, req.Type, data)
	if err != nil {
		return &TriggerResult{
			Accepted: false,
			Message:  err.Error(),
		}, nil
	}

	return &TriggerResult{
		Accepted:    true,
		ExecutionID: execID,
		Message:     fmt.Sprintf("Trigger (%s) submitted", req.Type),
	}, nil
}

// managerInterface defines the methods we need from manager
// This allows for easier testing with mocks
type managerInterface interface {
	TriggerManual(ctx *types.Context, memberID string, trigger types.TriggerType, data interface{}) (string, error)
	Intervene(ctx *types.Context, req *types.InterveneRequest) (*types.ExecutionResult, error)
	HandleEvent(ctx *types.Context, req *types.EventRequest) (*types.ExecutionResult, error)
	PauseExecution(ctx *types.Context, execID string) error
	ResumeExecution(ctx *types.Context, execID string) error
	StopExecution(ctx *types.Context, execID string) error
}
