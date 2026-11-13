package types

import "context"

// InstanceStatus is the normalized cloud provider instance state.
type InstanceStatus string

const (
	InstancePending InstanceStatus = "pending"
	InstanceRunning InstanceStatus = "running"
	InstanceStopped InstanceStatus = "stopped"
	InstanceError   InstanceStatus = "error"
	InstanceDeleted InstanceStatus = "deleted"
)

// CreateRequest is passed to a cloud provider.
type CreateRequest struct {
	OrderID        string
	ProviderPlanID string
	Region         string
	Name           string
	IdempotencyKey string
	RequestParams  map[string]interface{}
	Metadata       map[string]interface{}
}

// UpdateRequest updates a cloud provider instance.
type ProviderUpdateRequest struct {
	Name           string
	ProviderPlanID string
	Metadata       map[string]interface{}
}

// ListOptions filters cloud provider instances.
type ListOptions struct {
	Region string
	Status string
	Page   int
	Size   int
}

// Instance is a cloud provider instance.
type Instance struct {
	ProviderInstanceID string
	Status             InstanceStatus
	RawStatus          string
	IP                 string
	Region             string
}

// Provider is the cloud provider driver contract.
type Provider interface {
	Name() string
	Create(ctx context.Context, req CreateRequest) (*Instance, error)
	Get(ctx context.Context, instanceID string) (*Instance, error)
	List(ctx context.Context, opts ListOptions) ([]Instance, error)
	Update(ctx context.Context, instanceID string, req ProviderUpdateRequest) (*Instance, error)
	Start(ctx context.Context, instanceID string) error
	Stop(ctx context.Context, instanceID string) error
	Reboot(ctx context.Context, instanceID string) error
	Delete(ctx context.Context, instanceID string) error
}
