package computer

import (
	"context"

	"github.com/yaoapp/yao/modules/computer/types"
)

// ListWorkspaces returns workspaces registered for the Tai node.
func (s *Service) ListWorkspaces(ctx context.Context, computerID string) ([]types.WorkspaceRef, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	computer, err := s.Get(ctx, computerID)
	if err != nil {
		return nil, err
	}
	if s.taiNodes == nil {
		return nil, ErrNotSupported
	}
	items, err := s.taiNodes.ListWorkspaces(ctx, computer.NodeID)
	if err != nil {
		return nil, err
	}
	out := make([]types.WorkspaceRef, 0, len(items))
	for _, item := range items {
		out = append(out, types.WorkspaceRef{ID: item.ID, Name: item.Name})
	}
	return out, nil
}
