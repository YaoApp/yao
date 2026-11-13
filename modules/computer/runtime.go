package computer

import (
	"context"

	"github.com/yaoapp/yao/modules/computer/types"
)

// ListRuntimes returns runtimes advertised by the node. Tai capability wiring
// is intentionally not faked before the runtime endpoint exists.
func (s *Service) ListRuntimes(ctx context.Context, computerID string) ([]types.Runtime, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, err := s.Get(ctx, computerID); err != nil {
		return nil, err
	}
	return []types.Runtime{}, nil
}

// ListContainers returns containers advertised by the node. An empty result is
// used until Tai exposes container inspection.
func (s *Service) ListContainers(ctx context.Context, computerID string) ([]types.Container, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, err := s.Get(ctx, computerID); err != nil {
		return nil, err
	}
	return []types.Container{}, nil
}
