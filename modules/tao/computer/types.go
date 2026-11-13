// Package computer integrates Yao with the Tao Computers API v1.
package computer

import (
	"context"
	"errors"
	"time"

	computertypes "github.com/yaoapp/yao/modules/computer/types"
)

var ErrTaiStart = errors.New("tai start failed")

// ErrInstanceDeleted is returned when Tao reports the instance as gone.
var ErrInstanceDeleted = errors.New("tao instance deleted")

const (
	PathComputers = "/v1/computers"
	PathCatalog   = "/v1/catalog"

	BillingHour  = "HOUR"
	BillingMonth = "MONTH"
)

// CreateInstanceRequest is sent to Tao POST /v1/computers.
type CreateInstanceRequest struct {
	Resource       string            `json:"resource"`
	Region         string            `json:"region"`
	Name           string            `json:"name,omitempty"`
	BillingUnit    string            `json:"billing_unit,omitempty"`
	PeriodMonths   int               `json:"period_months,omitempty"`
	RequestParams  map[string]any    `json:"request_params,omitempty"`
	SSH            *SSHRequest       `json:"ssh,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	IdempotencyKey string            `json:"-"`
}

// SSHRequest injects an SSH public key or an existing cloud key ID.
type SSHRequest struct {
	PublicKey string `json:"public_key,omitempty"`
	KeyID     string `json:"key_id,omitempty"`
}

// SSHInfo is the SSH endpoint returned by Tao. It never contains a private key.
type SSHInfo struct {
	Host                 string `json:"host,omitempty"`
	Port                 int    `json:"port,omitempty"`
	User                 string `json:"user,omitempty"`
	PublicKeyFingerprint string `json:"public_key_fingerprint,omitempty"`
}

// Instance is one Tao cloud host.
type Instance struct {
	InstanceID        string            `json:"instance_id"`
	Status            string            `json:"status"`
	RawStatus         string            `json:"raw_status,omitempty"`
	Region            string            `json:"region"`
	Resource          string            `json:"resource,omitempty"`
	BillingUnit       string            `json:"billing_unit,omitempty"`
	PeriodMonths      int               `json:"period_months,omitempty"`
	PriceCredits      int               `json:"price_credits,omitempty"`
	PriceTotalCredits *int              `json:"price_total_credits,omitempty"`
	ExpiresAt         *time.Time        `json:"expires_at,omitempty"`
	PublicIP          string            `json:"public_ip,omitempty"`
	PrivateIP         string            `json:"private_ip,omitempty"`
	SSH               SSHInfo           `json:"ssh,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	LastError         string            `json:"last_error,omitempty"`
	CreatedAt         time.Time         `json:"created_at,omitempty"`
	UpdatedAt         time.Time         `json:"updated_at,omitempty"`
}

// ListOptions filters Tao computer lists.
type ListOptions struct {
	Status string
	Region string
	Page   int
	Size   int
}

// DurationTier is a MONTH duration discount tier.
type DurationTier struct {
	Months   string  `json:"months"`
	Discount float64 `json:"discount"`
}

// CatalogResource is one sellable Tao computer resource.
type CatalogResource struct {
	ID             string         `json:"id"`
	Region         string         `json:"region,omitempty"`
	CPU            int            `json:"cpu,omitempty"`
	MemoryMB       int            `json:"memory_mb,omitempty"`
	DiskGB         int            `json:"disk_gb,omitempty"`
	BandwidthMbps  int            `json:"bandwidth_mbps,omitempty"`
	BillingUnits   []string       `json:"billing_units,omitempty"`
	Prices         map[string]int `json:"prices,omitempty"`
	DurationTiers  []DurationTier `json:"duration_tiers,omitempty"`
	Images         []string       `json:"images,omitempty"`
	Enabled        bool           `json:"enabled"`
	AvailableSlots int            `json:"available_slots,omitempty"`
}

// CatalogService is the Tao Catalog service block for computers.
type CatalogService struct {
	ID        string            `json:"id"`
	Kind      string            `json:"kind"`
	Units     []string          `json:"units,omitempty"`
	Resources []CatalogResource `json:"resources"`
}

// Catalog is the Tao /v1/catalog response subset.
type Catalog struct {
	Object   string           `json:"object"`
	Locales  []string         `json:"locales"`
	Services []CatalogService `json:"services"`
}

// ComputerSpec is the Yao Computer part of a purchase.
type ComputerSpec struct {
	Name           string     `json:"name"`
	DataDir        string     `json:"data_dir,omitempty"`
	Visibility     string     `json:"visibility,omitempty"`
	SSHKeyRef      string     `json:"ssh_key_ref,omitempty"`
	SandboxImage   string     `json:"sandbox_image,omitempty"`
	LeaseSource    string     `json:"lease_source,omitempty"`
	LeaseExpiresAt *time.Time `json:"lease_expires_at,omitempty"`
}

// PurchaseRequest is the service-level purchase input.
type PurchaseRequest struct {
	Resource       string
	Region         string
	Name           string
	BillingUnit    string
	PeriodMonths   int
	RequestParams  map[string]any
	SSH            *SSHRequest
	Metadata       map[string]string
	IdempotencyKey string
	Computer       ComputerSpec
	UserID         string
	TeamID         string
}

// PurchaseResult is returned by purchase and query.
type PurchaseResult struct {
	Instance *Instance
	Computer *computertypes.Computer
}

// ComputerService is the subset of Computer service used by orchestration.
type ComputerService interface {
	CreateOrUpdate(context.Context, computertypes.CreateOrUpdateComputerRequest) (*computertypes.UpsertResult, error)
	Get(context.Context, string) (*computertypes.Computer, error)
}

// SSHKeyResolver resolves a Vault SSH key reference to private key material.
type SSHKeyResolver interface {
	ResolveSSHKey(context.Context, string) (string, error)
}

// TaiController is the SSH/Tai wrapper used after Tao reports running.
type TaiController interface {
	Bootstrap(context.Context, computertypes.SSHTarget, computertypes.BootstrapRequest) error
	Start(context.Context, computertypes.SSHTarget, computertypes.StartTaiRequest) error
	Stop(context.Context, computertypes.SSHTarget) error
	Status(context.Context, computertypes.SSHTarget) (*computertypes.TaiStatus, error)
	Logs(context.Context, computertypes.SSHTarget, int) (string, error)
}

// TaoComputersClient is the Tao Computers API contract.
type TaoComputersClient interface {
	Create(context.Context, string, CreateInstanceRequest) (*Instance, error)
	Get(context.Context, string, string) (*Instance, error)
	List(context.Context, string, ListOptions) ([]Instance, error)
	GetCatalog(context.Context, string) (*Catalog, error)
}
