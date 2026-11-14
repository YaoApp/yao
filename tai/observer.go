package tai

import (
	"context"

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
