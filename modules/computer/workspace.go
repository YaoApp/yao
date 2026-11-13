package computer

import (
	"context"

	"github.com/yaoapp/yao/modules/computer/types"
)

// ListWorkspaces returns workspaces associated with a computer. Workspace
// storage integration is a later phase; an empty list is returned for now.
func (s *Service) ListWorkspaces(ctx context.Context, computerID string) ([]types.WorkspaceRef, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, err := s.Get(ctx, computerID); err != nil {
		return nil, err
	}
	return []types.WorkspaceRef{}, nil
}
