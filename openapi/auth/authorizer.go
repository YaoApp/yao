package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/openapi/oauth/acl/role"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

var (
	// ErrComputerNotFound is returned when the target computer does not exist.
	ErrComputerNotFound = errors.New("computer not found")
	// ErrComputerForbidden is returned when the caller cannot manage the computer.
	ErrComputerForbidden = errors.New("computer forbidden")
)

// ComputerAuthorizer checks whether the authenticated identity can manage a computer.
type ComputerAuthorizer interface {
	CanManage(ctx context.Context, info *types.AuthorizedInfo, computerID string) error
}

// ModelComputerAuthorizer resolves computer ownership from __yao.computers.
type ModelComputerAuthorizer struct{}

// CanManage checks the owner fields written by the model permission layer.
func (ModelComputerAuthorizer) CanManage(ctx context.Context, info *types.AuthorizedInfo, computerID string) error {
	if info == nil || info.UserID == "" {
		return ErrComputerForbidden
	}
	row, err := model.Select("__yao.computers").Find(computerID, model.QueryParam{})
	if err != nil {
		return ErrComputerNotFound
	}
	for _, key := range []string{"__yao_user_id", "__yao_created_by"} {
		if fmt.Sprint(row[key]) == info.UserID {
			return nil
		}
	}
	if role.RoleManager != nil {
		if roleID, err := role.RoleManager.GetUserRole(ctx, info.UserID); err == nil {
			if roleID == "system:root" || roleID == "system:admin" {
				return nil
			}
		}
	}
	return ErrComputerForbidden
}
