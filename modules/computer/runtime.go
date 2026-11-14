package computer

import (
	"context"
	"errors"

	"github.com/yaoapp/yao/modules/computer/types"
)

// ErrNotSupported is returned when the Tai node does not expose an API yet.
var ErrNotSupported = errors.New("computer capability is not supported")

// ListRuntimes returns container runtimes advertised by the registered Tai node.
func (s *Service) ListRuntimes(ctx context.Context, computerID string) ([]types.Runtime, error) {
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
	node, ok := s.taiNodes.Node(computer.NodeID)
	if !ok {
		return nil, ErrNotSupported
	}
	out := make([]types.Runtime, 0, len(node.ContainerRuntimes))
	for _, runtime := range node.ContainerRuntimes {
		out = append(out, types.Runtime{Name: runtime.Name, Version: runtime.Version, Ready: true})
	}
	return out, nil
}

// ListContainers is not available through the Tai SDK yet.
func (s *Service) ListContainers(ctx context.Context, computerID string) ([]types.Container, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, err := s.Get(ctx, computerID); err != nil {
		return nil, err
	}
	return nil, ErrNotSupported
}
