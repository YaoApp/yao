package robot

import (
	"context"
	"fmt"
	"testing"

	"github.com/yaoapp/gou/process"
)

func mockAuth(userID, teamID string) *process.AuthorizedInfo {
	return &process.AuthorizedInfo{
		UserID: userID,
		TeamID: teamID,
	}
}

func mockProc(auth *process.AuthorizedInfo, args ...interface{}) *process.Process {
	return &process.Process{
		Args:       args,
		Authorized: auth,
		Context:    context.Background(),
	}
}

func hasError(result interface{}) (string, bool) {
	m, ok := result.(map[string]any)
	if !ok {
		return "", false
	}
	e, ok := m["error"].(string)
	return e, ok
}

// ==================== CreateHandler tests ====================

func TestCreateHandler_NoAuth(t *testing.T) {
	proc := mockProc(nil, map[string]interface{}{"display_name": "Bot"})
	result := CreateHandler(proc)
	msg, ok := hasError(result)
	if !ok || msg != "unauthorized" {
		t.Errorf("expected 'unauthorized', got %v", result)
	}
}

func TestCreateHandler_NoFn(t *testing.T) {
	old := CreateRobotFn
	CreateRobotFn = nil
	defer func() { CreateRobotFn = old }()

	proc := mockProc(mockAuth("u1", "t1"), map[string]interface{}{"display_name": "Bot"})
	result := CreateHandler(proc)
	msg, ok := hasError(result)
	if !ok || msg != "robot API not initialized" {
		t.Errorf("expected 'robot API not initialized', got %v", result)
	}
}

func TestCreateHandler_MissingDisplayName(t *testing.T) {
	CreateRobotFn = func(ctx context.Context, auth *AuthInfo, req *CreateRequest) (*RobotResponse, error) {
		t.Fatal("should not be called")
		return nil, nil
	}
	defer func() { CreateRobotFn = nil }()

	proc := mockProc(mockAuth("u1", "t1"), map[string]interface{}{"bio": "no name"})
	result := CreateHandler(proc)
	msg, ok := hasError(result)
	if !ok || msg != "display_name is required" {
		t.Errorf("expected 'display_name is required', got %v", result)
	}
}

func TestCreateHandler_WhitelistFilters(t *testing.T) {
	var captured *CreateRequest
	CreateRobotFn = func(ctx context.Context, auth *AuthInfo, req *CreateRequest) (*RobotResponse, error) {
		captured = req
		return &RobotResponse{Data: map[string]string{"member_id": "new-1"}}, nil
	}
	defer func() { CreateRobotFn = nil }()

	proc := mockProc(mockAuth("u1", "t1"), map[string]interface{}{
		"display_name":   "Good Bot",
		"bio":            "A good bot",
		"mcp_servers":    []string{"evil-server"},
		"cost_limit":     99999.0,
		"language_model": "gpt-4",
		"status":         "active",
		"robot_email":    "evil@evil.com",
	})

	result := CreateHandler(proc)
	if _, isErr := hasError(result); isErr {
		t.Fatalf("unexpected error: %v", result)
	}

	if captured == nil {
		t.Fatal("CreateRobotFn was not called")
	}
	if captured.DisplayName != "Good Bot" {
		t.Errorf("DisplayName = %q, want 'Good Bot'", captured.DisplayName)
	}
	if captured.Bio != "A good bot" {
		t.Errorf("Bio = %q, want 'A good bot'", captured.Bio)
	}
}

func TestCreateHandler_WithRobotConfig(t *testing.T) {
	var captured *CreateRequest
	CreateRobotFn = func(ctx context.Context, auth *AuthInfo, req *CreateRequest) (*RobotResponse, error) {
		captured = req
		return &RobotResponse{Data: "ok"}, nil
	}
	defer func() { CreateRobotFn = nil }()

	proc := mockProc(mockAuth("u1", "t1"), map[string]interface{}{
		"display_name": "Config Bot",
		"robot_config": map[string]interface{}{
			"identity": map[string]interface{}{
				"role":   "analyst",
				"duties": []interface{}{"analyze"},
			},
			"quota": map[string]interface{}{
				"max": float64(5),
			},
			"integrations": map[string]interface{}{
				"telegram": map[string]interface{}{
					"bot_token": "STOLEN",
				},
			},
		},
	})

	result := CreateHandler(proc)
	if _, isErr := hasError(result); isErr {
		t.Fatalf("unexpected error: %v", result)
	}
	if captured.RobotConfig == nil {
		t.Fatal("RobotConfig should not be nil")
	}
	if captured.RobotConfig.Identity == nil || captured.RobotConfig.Identity.Role != "analyst" {
		t.Errorf("Identity.Role = %v, want 'analyst'", captured.RobotConfig.Identity)
	}
	if captured.RobotConfig.Quota == nil || captured.RobotConfig.Quota.Max != 5 {
		t.Errorf("Quota.Max = %v, want 5", captured.RobotConfig.Quota)
	}
}

func TestCreateHandler_AuthPropagation(t *testing.T) {
	var capturedAuth *AuthInfo
	CreateRobotFn = func(ctx context.Context, auth *AuthInfo, req *CreateRequest) (*RobotResponse, error) {
		capturedAuth = auth
		return &RobotResponse{Data: "ok"}, nil
	}
	defer func() { CreateRobotFn = nil }()

	proc := mockProc(&process.AuthorizedInfo{
		UserID:   "user-abc",
		TeamID:   "team-xyz",
		TenantID: "tenant-1",
	}, map[string]interface{}{"display_name": "Bot"})

	CreateHandler(proc)
	if capturedAuth == nil {
		t.Fatal("auth not propagated")
	}
	if capturedAuth.UserID != "user-abc" {
		t.Errorf("UserID = %q", capturedAuth.UserID)
	}
	if capturedAuth.TeamID != "team-xyz" {
		t.Errorf("TeamID = %q", capturedAuth.TeamID)
	}
	if capturedAuth.TenantID != "tenant-1" {
		t.Errorf("TenantID = %q", capturedAuth.TenantID)
	}
}

// ==================== UpdateHandler tests ====================

func TestUpdateHandler_NoAuth(t *testing.T) {
	proc := mockProc(nil, map[string]interface{}{"member_id": "rob-1"})
	result := UpdateHandler(proc)
	msg, _ := hasError(result)
	if msg != "unauthorized" {
		t.Errorf("expected 'unauthorized', got %v", result)
	}
}

func TestUpdateHandler_MissingMemberID(t *testing.T) {
	UpdateRobotFn = func(ctx context.Context, auth *AuthInfo, memberID string, req *UpdateRequest) (*RobotResponse, error) {
		return nil, nil
	}
	defer func() { UpdateRobotFn = nil }()

	proc := mockProc(mockAuth("u1", "t1"), map[string]interface{}{"bio": "update"})
	result := UpdateHandler(proc)
	msg, _ := hasError(result)
	if msg != "member_id is required" {
		t.Errorf("expected 'member_id is required', got %v", result)
	}
}

func TestUpdateHandler_PermissionCheck(t *testing.T) {
	GetRobotResponseFn = func(ctx context.Context, auth *AuthInfo, memberID string) (*RobotResponse, error) {
		return &RobotResponse{
			Data:         map[string]string{"member_id": memberID},
			YaoTeamID:    "team-other",
			YaoCreatedBy: "user-other",
		}, nil
	}
	UpdateRobotFn = func(ctx context.Context, auth *AuthInfo, memberID string, req *UpdateRequest) (*RobotResponse, error) {
		t.Fatal("should not be called — permission denied")
		return nil, nil
	}
	defer func() { GetRobotResponseFn = nil; UpdateRobotFn = nil }()

	proc := mockProc(&process.AuthorizedInfo{
		UserID:      "u1",
		TeamID:      "t1",
		Constraints: process.DataConstraints{OwnerOnly: true},
	}, map[string]interface{}{"member_id": "rob-1", "bio": "hacked"})

	result := UpdateHandler(proc)
	msg, _ := hasError(result)
	if msg != "permission denied" {
		t.Errorf("expected 'permission denied', got %v", result)
	}
}

func TestUpdateHandler_WhitelistFilters(t *testing.T) {
	GetRobotResponseFn = func(ctx context.Context, auth *AuthInfo, memberID string) (*RobotResponse, error) {
		return &RobotResponse{
			Data:         "ok",
			YaoTeamID:    "t1",
			YaoCreatedBy: "u1",
		}, nil
	}
	var captured *UpdateRequest
	var capturedMemberID string
	UpdateRobotFn = func(ctx context.Context, auth *AuthInfo, memberID string, req *UpdateRequest) (*RobotResponse, error) {
		captured = req
		capturedMemberID = memberID
		return &RobotResponse{Data: "updated"}, nil
	}
	defer func() { GetRobotResponseFn = nil; UpdateRobotFn = nil }()

	proc := mockProc(mockAuth("u1", "t1"), map[string]interface{}{
		"member_id":      "rob-1",
		"bio":            "new bio",
		"mcp_servers":    []string{"evil"},
		"cost_limit":     9999.0,
		"language_model": "gpt-4",
	})

	result := UpdateHandler(proc)
	if _, isErr := hasError(result); isErr {
		t.Fatalf("unexpected error: %v", result)
	}
	if capturedMemberID != "rob-1" {
		t.Errorf("memberID = %q, want 'rob-1'", capturedMemberID)
	}
	if captured.Bio == nil || *captured.Bio != "new bio" {
		t.Errorf("Bio = %v, want 'new bio'", captured.Bio)
	}
}

// ==================== ExecutionCreateHandler tests ====================

func TestExecCreate_NoAuth(t *testing.T) {
	proc := mockProc(nil, map[string]interface{}{"member_id": "rob-1"})
	result := ExecutionCreateHandler(proc)
	msg, _ := hasError(result)
	if msg != "unauthorized" {
		t.Errorf("expected 'unauthorized', got %v", result)
	}
}

func TestExecCreate_MissingMemberID(t *testing.T) {
	TriggerFn = func(ctx context.Context, auth *AuthInfo, memberID string, req *TriggerRequest) (*TriggerResult, error) {
		return nil, nil
	}
	defer func() { TriggerFn = nil }()

	proc := mockProc(mockAuth("u1", "t1"), map[string]interface{}{"messages": []interface{}{}})
	result := ExecutionCreateHandler(proc)
	msg, _ := hasError(result)
	if msg != "member_id is required" {
		t.Errorf("expected 'member_id is required', got %v", result)
	}
}

func TestExecCreate_DefaultsToHuman(t *testing.T) {
	GetRobotResponseFn = func(ctx context.Context, auth *AuthInfo, memberID string) (*RobotResponse, error) {
		return &RobotResponse{Data: "ok", YaoTeamID: "t1", YaoCreatedBy: "u1"}, nil
	}
	var capturedReq *TriggerRequest
	TriggerFn = func(ctx context.Context, auth *AuthInfo, memberID string, req *TriggerRequest) (*TriggerResult, error) {
		capturedReq = req
		return &TriggerResult{ExecutionID: "exec-1", Accepted: true}, nil
	}
	defer func() { GetRobotResponseFn = nil; TriggerFn = nil }()

	proc := mockProc(mockAuth("u1", "t1"), map[string]interface{}{
		"member_id": "rob-1",
		"messages":  []interface{}{map[string]interface{}{"role": "user", "content": "hello"}},
	})

	result := ExecutionCreateHandler(proc)
	if _, isErr := hasError(result); isErr {
		t.Fatalf("unexpected error: %v", result)
	}
	if capturedReq.Type != "human" {
		t.Errorf("Type = %q, want 'human' (default)", capturedReq.Type)
	}
	if len(capturedReq.Messages) != 1 {
		t.Fatalf("Messages len = %d, want 1", len(capturedReq.Messages))
	}
	if capturedReq.Messages[0].Content != "hello" {
		t.Errorf("Messages[0].Content = %q, want 'hello'", capturedReq.Messages[0].Content)
	}
}

func TestExecCreate_TriggerTypeMapping(t *testing.T) {
	GetRobotResponseFn = func(ctx context.Context, auth *AuthInfo, memberID string) (*RobotResponse, error) {
		return &RobotResponse{Data: "ok", YaoTeamID: "t1", YaoCreatedBy: "u1"}, nil
	}
	var capturedReq *TriggerRequest
	TriggerFn = func(ctx context.Context, auth *AuthInfo, memberID string, req *TriggerRequest) (*TriggerResult, error) {
		capturedReq = req
		return &TriggerResult{ExecutionID: "exec-2", Accepted: true}, nil
	}
	defer func() { GetRobotResponseFn = nil; TriggerFn = nil }()

	proc := mockProc(mockAuth("u1", "t1"), map[string]interface{}{
		"member_id":    "rob-1",
		"trigger_type": "event",
		"source":       "webhook",
		"event_type":   "lead.created",
	})

	result := ExecutionCreateHandler(proc)
	if _, isErr := hasError(result); isErr {
		t.Fatalf("unexpected error: %v", result)
	}
	if capturedReq.Type != "event" {
		t.Errorf("Type = %q, want 'event' (from trigger_type)", capturedReq.Type)
	}
	if capturedReq.Source != "webhook" {
		t.Errorf("Source = %q, want 'webhook'", capturedReq.Source)
	}
	if capturedReq.EventType != "lead.created" {
		t.Errorf("EventType = %q, want 'lead.created'", capturedReq.EventType)
	}
}

func TestExecCreate_InvalidTriggerType(t *testing.T) {
	TriggerFn = func(ctx context.Context, auth *AuthInfo, memberID string, req *TriggerRequest) (*TriggerResult, error) {
		t.Fatal("should not be called for invalid trigger type")
		return nil, nil
	}
	defer func() { TriggerFn = nil }()

	proc := mockProc(mockAuth("u1", "t1"), map[string]interface{}{
		"member_id":    "rob-1",
		"trigger_type": "invalid",
	})

	result := ExecutionCreateHandler(proc)
	msg, ok := hasError(result)
	if !ok || msg != "trigger_type must be 'human' or 'event'" {
		t.Errorf("expected trigger_type error, got %v", result)
	}
}

func TestExecCreate_ResponseShape(t *testing.T) {
	GetRobotResponseFn = func(ctx context.Context, auth *AuthInfo, memberID string) (*RobotResponse, error) {
		return &RobotResponse{Data: "ok", YaoTeamID: "t1", YaoCreatedBy: "u1"}, nil
	}
	TriggerFn = func(ctx context.Context, auth *AuthInfo, memberID string, req *TriggerRequest) (*TriggerResult, error) {
		return &TriggerResult{ExecutionID: "exec-123", Accepted: true, Message: "queued"}, nil
	}
	defer func() { GetRobotResponseFn = nil; TriggerFn = nil }()

	proc := mockProc(mockAuth("u1", "t1"), map[string]interface{}{"member_id": "rob-1"})
	result := ExecutionCreateHandler(proc)

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("result is not map, got %T", result)
	}
	if m["execution_id"] != "exec-123" {
		t.Errorf("execution_id = %v", m["execution_id"])
	}
	if m["accepted"] != true {
		t.Errorf("accepted = %v", m["accepted"])
	}
	if m["status"] != "submitted" {
		t.Errorf("status = %v", m["status"])
	}
	if m["message"] != "queued" {
		t.Errorf("message = %v", m["message"])
	}
}

func TestExecCreate_WhitelistFilters(t *testing.T) {
	GetRobotResponseFn = func(ctx context.Context, auth *AuthInfo, memberID string) (*RobotResponse, error) {
		return &RobotResponse{Data: "ok", YaoTeamID: "t1", YaoCreatedBy: "u1"}, nil
	}
	var capturedReq *TriggerRequest
	TriggerFn = func(ctx context.Context, auth *AuthInfo, memberID string, req *TriggerRequest) (*TriggerResult, error) {
		capturedReq = req
		return &TriggerResult{ExecutionID: "exec-1", Accepted: true}, nil
	}
	defer func() { GetRobotResponseFn = nil; TriggerFn = nil }()

	proc := mockProc(mockAuth("u1", "t1"), map[string]interface{}{
		"member_id":     "rob-1",
		"messages":      []interface{}{map[string]interface{}{"role": "user", "content": "hi"}},
		"action":        "task.add",
		"executor_mode": "sandbox",
		"plan_at":       "2025-01-01T00:00:00Z",
		"locale":        "zh",
	})

	result := ExecutionCreateHandler(proc)
	if _, isErr := hasError(result); isErr {
		t.Fatalf("unexpected error: %v", result)
	}

	if capturedReq.Source != "" {
		t.Errorf("Source should be empty but got %q", capturedReq.Source)
	}
	if capturedReq.EventType != "" {
		t.Errorf("EventType should be empty but got %q", capturedReq.EventType)
	}
}

// ==================== ListHandler tests ====================

func TestListHandler_NoAuth(t *testing.T) {
	proc := mockProc(nil, map[string]interface{}{})
	result := ListHandler(proc)
	msg, _ := hasError(result)
	if msg != "unauthorized" {
		t.Errorf("expected 'unauthorized', got %v", result)
	}
}

func TestListHandler_Success(t *testing.T) {
	ListAllRobotsFn = func(ctx context.Context, auth *AuthInfo, query *ListQuery) (*ListResult, error) {
		if query.TeamID != "t1" {
			t.Errorf("TeamID = %q, want 't1' (from auth fallback)", query.TeamID)
		}
		return &ListResult{
			Data: []RobotSummary{
				{MemberID: "r-1", DisplayName: "Bot 1", Status: "idle"},
			},
			Total: 1, Page: 1, PageSize: 20,
		}, nil
	}
	defer func() { ListAllRobotsFn = nil }()

	proc := mockProc(mockAuth("u1", "t1"), map[string]interface{}{})
	result := ListHandler(proc)
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("result is not map, got %T", result)
	}
	if m["total"] != 1 {
		t.Errorf("total = %v", m["total"])
	}
}

func TestListHandler_WithFilters(t *testing.T) {
	var capturedQuery *ListQuery
	ListAllRobotsFn = func(ctx context.Context, auth *AuthInfo, query *ListQuery) (*ListResult, error) {
		capturedQuery = query
		return &ListResult{Data: []RobotSummary{}, Total: 0, Page: 1, PageSize: 10}, nil
	}
	defer func() { ListAllRobotsFn = nil }()

	proc := mockProc(mockAuth("u1", "t1"), map[string]interface{}{
		"status":   "working",
		"keywords": "sales",
		"page":     float64(2),
		"pagesize": float64(10),
	})
	ListHandler(proc)

	if capturedQuery.Status != "working" {
		t.Errorf("Status = %q", capturedQuery.Status)
	}
	if capturedQuery.Keywords != "sales" {
		t.Errorf("Keywords = %q", capturedQuery.Keywords)
	}
	if capturedQuery.Page != 2 {
		t.Errorf("Page = %d", capturedQuery.Page)
	}
	if capturedQuery.PageSize != 10 {
		t.Errorf("PageSize = %d", capturedQuery.PageSize)
	}
}

func TestListHandler_Error(t *testing.T) {
	ListAllRobotsFn = func(ctx context.Context, auth *AuthInfo, query *ListQuery) (*ListResult, error) {
		return nil, fmt.Errorf("db connection failed")
	}
	defer func() { ListAllRobotsFn = nil }()

	proc := mockProc(mockAuth("u1", "t1"), map[string]interface{}{})
	result := ListHandler(proc)
	msg, ok := hasError(result)
	if !ok || msg != "db connection failed" {
		t.Errorf("expected error propagation, got %v", result)
	}
}

// ==================== StatusHandler tests ====================

func TestStatusHandler_Success(t *testing.T) {
	GetRobotStatusFn = func(ctx context.Context, auth *AuthInfo, memberID string) (*RobotState, error) {
		return &RobotState{
			MemberID:     "rob-1",
			TeamID:       "t1",
			DisplayName:  "Bot 1",
			Status:       "working",
			Running:      2,
			MaxRunning:   5,
			RunningIDs:   []string{"exec-a", "exec-b"},
			YaoTeamID:    "t1",
			YaoCreatedBy: "u1",
		}, nil
	}
	defer func() { GetRobotStatusFn = nil }()

	proc := mockProc(mockAuth("u1", "t1"), "rob-1")
	result := StatusHandler(proc)
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("result is not map, got %T", result)
	}
	if m["running"] != 2 {
		t.Errorf("running = %v", m["running"])
	}
	if m["max_running"] != 5 {
		t.Errorf("max_running = %v", m["max_running"])
	}
	ids, ok := m["running_ids"].([]string)
	if !ok || len(ids) != 2 {
		t.Errorf("running_ids = %v", m["running_ids"])
	}
}

func TestStatusHandler_MissingMemberID(t *testing.T) {
	proc := mockProc(mockAuth("u1", "t1"), "")
	result := StatusHandler(proc)
	msg, _ := hasError(result)
	if msg != "member_id is required" {
		t.Errorf("expected 'member_id is required', got %v", result)
	}
}
