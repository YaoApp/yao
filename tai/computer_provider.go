package tai

import (
	"context"

	"github.com/yaoapp/tai-golang/node"
	computertypes "github.com/yaoapp/yao/modules/computer/types"
)

// ComputerProvider adapts the Tai registry to computertypes.TaiNodeProvider.
type ComputerProvider struct{}

// Node implements computertypes.TaiNodeProvider.
func (ComputerProvider) Node(nodeID string) (computertypes.TaiNode, bool) {
	item, ok := (Provider{}).Node(nodeID)
	if !ok {
		return computertypes.TaiNode{}, false
	}
	out := computertypes.TaiNode{
		ID: item.ID, Version: item.Version, OS: item.OS, Arch: item.Arch,
		Capabilities: append([]string(nil), item.Capabilities...),
	}
	for _, runtime := range item.ContainerRuntimes {
		out.ContainerRuntimes = append(out.ContainerRuntimes, computertypes.Runtime{
			Name: runtime.Name, Version: runtime.Version, Ready: true,
		})
	}
	return out, true
}

// GoAway implements computertypes.TaiNodeController.
func (ComputerProvider) GoAway(_ context.Context, nodeID, reason string) error {
	if node.Default() == nil {
		return nil
	}
	rn := node.Get(nodeID)
	if rn == nil {
		return nil
	}
	return rn.GoAway(reason, 0)
}

// ListWorkspaces implements computertypes.TaiNodeProvider.
func (ComputerProvider) ListWorkspaces(ctx context.Context, nodeID string) ([]computertypes.TaiWorkspace, error) {
	entries, err := (Provider{}).ListWorkspaces(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	out := make([]computertypes.TaiWorkspace, 0, len(entries))
	for _, entry := range entries {
		out = append(out, computertypes.TaiWorkspace{ID: entry.ID, Name: entry.Name})
	}
	return out, nil
}
