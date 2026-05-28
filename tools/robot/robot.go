package robot

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/process"
)

// Function variables set during app init to avoid circular imports.
var (
	ListAllRobotsFn    func(ctx context.Context, auth *AuthInfo, query *ListQuery) (*ListResult, error)
	GetRobotResponseFn func(ctx context.Context, auth *AuthInfo, memberID string) (*RobotResponse, error)
	GetRobotStatusFn   func(ctx context.Context, auth *AuthInfo, memberID string) (*RobotState, error)
	CreateRobotFn      func(ctx context.Context, auth *AuthInfo, req *CreateRequest) (*RobotResponse, error)
	UpdateRobotFn      func(ctx context.Context, auth *AuthInfo, memberID string, req *UpdateRequest) (*RobotResponse, error)
	TriggerFn          func(ctx context.Context, auth *AuthInfo, memberID string, req *TriggerRequest) (*TriggerResult, error)
	StopExecutionFn    func(ctx context.Context, auth *AuthInfo, execID string) error
	ListExecutionsFn   func(ctx context.Context, auth *AuthInfo, memberID string, query *ExecutionQuery) (*ExecutionResult, error)
	GetExecutionFn     func(ctx context.Context, auth *AuthInfo, execID string) (*ExecutionDetail, error)
	ListResultsFn      func(ctx context.Context, auth *AuthInfo, memberID string, query *ResultQuery) (*ResultListResponse, error)
)

// AuthInfo carries auth context from process to API layer
type AuthInfo struct {
	UserID    string
	TeamID    string
	TenantID  string
	OwnerOnly bool
	TeamOnly  bool
}

func authFromProcess(proc *process.Process) *AuthInfo {
	auth := proc.Authorized
	if auth == nil {
		return nil
	}
	return &AuthInfo{
		UserID:    auth.UserID,
		TeamID:    auth.TeamID,
		TenantID:  auth.TenantID,
		OwnerOnly: auth.Constraints.OwnerOnly,
		TeamOnly:  auth.Constraints.TeamOnly,
	}
}

func getEffectiveTeamID(info *AuthInfo) string {
	if info == nil {
		return ""
	}
	if info.TeamID != "" {
		return info.TeamID
	}
	return info.UserID
}

func buildListFilter(info *AuthInfo, requestedTeamID string) string {
	if info == nil {
		return requestedTeamID
	}
	if !info.TeamOnly && !info.OwnerOnly {
		if requestedTeamID != "" {
			return requestedTeamID
		}
		return getEffectiveTeamID(info)
	}
	if info.TeamOnly && info.TeamID != "" {
		return info.TeamID
	}
	if info.OwnerOnly {
		return info.UserID
	}
	return requestedTeamID
}

func canRead(info *AuthInfo, robotTeamID, robotCreatedBy string) bool {
	if info == nil {
		return false
	}
	if !info.TeamOnly && !info.OwnerOnly {
		return true
	}
	if robotCreatedBy != "" && robotCreatedBy == info.UserID {
		return true
	}
	if info.TeamOnly && robotTeamID != "" && robotTeamID == info.TeamID {
		return true
	}
	if info.OwnerOnly {
		return false
	}
	return false
}

func canWrite(info *AuthInfo, robotTeamID, robotCreatedBy string) bool {
	if info == nil {
		return false
	}
	if !info.TeamOnly && !info.OwnerOnly {
		return true
	}
	if robotCreatedBy != "" && robotCreatedBy == info.UserID {
		if info.TeamOnly {
			if robotTeamID == "" || robotTeamID == info.TeamID {
				return true
			}
			return false
		}
		return true
	}
	return false
}

func checkRobotRead(proc *process.Process, memberID string) error {
	info := authFromProcess(proc)
	if info == nil {
		return fmt.Errorf("unauthorized")
	}
	if GetRobotResponseFn == nil {
		return fmt.Errorf("robot API not initialized")
	}
	resp, err := GetRobotResponseFn(proc.Context, info, memberID)
	if err != nil {
		return err
	}
	if !canRead(info, resp.YaoTeamID, resp.YaoCreatedBy) {
		return fmt.Errorf("permission denied")
	}
	return nil
}

func checkRobotWrite(proc *process.Process, memberID string) error {
	info := authFromProcess(proc)
	if info == nil {
		return fmt.Errorf("unauthorized")
	}
	if GetRobotResponseFn == nil {
		return fmt.Errorf("robot API not initialized")
	}
	resp, err := GetRobotResponseFn(proc.Context, info, memberID)
	if err != nil {
		return err
	}
	if !canWrite(info, resp.YaoTeamID, resp.YaoCreatedBy) {
		return fmt.Errorf("permission denied")
	}
	return nil
}
