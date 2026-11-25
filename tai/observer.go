package tai

import (
	"context"
	"time"

	computertypes "github.com/yaoapp/yao/modules/computer/types"
)

// ComputerStore is the subset of the Computer service used by the observer.
type ComputerStore interface {
	GetByNodeID(context.Context, string) (*computertypes.Computer, error)
	CreateOrUpdate(context.Context, computertypes.CreateOrUpdateComputerRequest) (*computertypes.UpsertResult, error)
}

// ComputerObserver advances Computer status from Tai node lifecycle events.
type ComputerObserver struct {
	Computers ComputerStore
}

// NodeRegistered implements NodeObserver.
func (o *ComputerObserver) NodeRegistered(ctx context.Context, node Node) error {
	if o == nil || o.Computers == nil {
		return nil
	}
	current, err := o.Computers.GetByNodeID(ctx, node.ID)
	if err != nil {
		return err
	}
	_, err = o.Computers.CreateOrUpdate(ctx, computertypes.CreateOrUpdateComputerRequest{
		ID: current.ID, NodeID: node.ID, Status: computertypes.StatusOnline,
		Snapshot: nodeSnapshot(node),
	})
	return err
}

// NodeDisconnected implements NodeObserver.
func (o *ComputerObserver) NodeDisconnected(ctx context.Context, nodeID string) {
	if o == nil || o.Computers == nil {
		return
	}
	current, err := o.Computers.GetByNodeID(ctx, nodeID)
	if err != nil {
		return
	}
	_, _ = o.Computers.CreateOrUpdate(ctx, computertypes.CreateOrUpdateComputerRequest{
		ID: current.ID, NodeID: nodeID, Status: computertypes.StatusOffline,
	})
}

func nodeSnapshot(node Node) *computertypes.ComputerSnapshot {
	snapshot := &computertypes.ComputerSnapshot{
		OS: node.OS, Arch: node.Arch, CPUCores: node.CPUCores,
		MemoryMB: node.MemoryMB, DiskFreeMB: node.DiskFreeMB,
		Capabilities: append([]string(nil), node.Capabilities...),
		TaiVersion:   node.Version, LastSeenAt: time.Now().UTC(),
	}
	for _, runtime := range node.ContainerRuntimes {
		snapshot.ContainerRuntimes = append(snapshot.ContainerRuntimes, computertypes.ContainerRuntime{
			Name: runtime.Name, Version: runtime.Version, Socket: runtime.Socket,
		})
	}
	for _, gpu := range node.GPUs {
		snapshot.GPUs = append(snapshot.GPUs, computertypes.GPU{Name: gpu.Name, VRAMMB: gpu.VRAMMB, Driver: gpu.Driver})
	}
	for _, network := range node.Networks {
		snapshot.Networks = append(snapshot.Networks, computertypes.Network{
			ID: network.ID, Name: network.Name, Kind: network.Kind, Family: network.Family,
			Address: network.Address, PrefixLength: network.PrefixLength, Primary: network.Primary,
		})
		if snapshot.IP == "" && network.Primary {
			snapshot.IP = network.Address
		}
	}
	return snapshot
}
