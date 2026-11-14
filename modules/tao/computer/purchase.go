package computer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	computertypes "github.com/yaoapp/yao/modules/computer/types"
)

// Service orchestrates Tao purchase and Yao Computer lifecycle.
type Service struct {
	client      TaoComputersClient
	computers   ComputerService
	tai         TaiController
	controlKey  *ControlKey
	controlKeys map[string]*ControlKey
	products    ProductWriter
	serverURL   string
	locks       sync.Map // instance_id -> *sync.Mutex
}

// NewService creates a Tao computer purchase service.
func NewService(client TaoComputersClient, computers ComputerService) *Service {
	return &Service{client: client, computers: computers}
}

// WithTai sets the SSH/Tai controller used after Tao reports running.
func (s *Service) WithTai(tai TaiController) *Service {
	s.tai = tai
	return s
}

// WithControlKey sets the active Yao system-level Tai control key.
func (s *Service) WithControlKey(key *ControlKey) *Service {
	s.controlKey = key
	if key != nil {
		if s.controlKeys == nil {
			s.controlKeys = map[string]*ControlKey{}
		}
		if key.KeyID != "" {
			s.controlKeys[key.KeyID] = key
		}
	}
	return s
}

// WithControlKeys sets a control-key ring and its active key ID. Existing
// Computers keep using the key ID recorded in their metadata.
func (s *Service) WithControlKeys(activeID string, keys ...*ControlKey) *Service {
	s.controlKeys = map[string]*ControlKey{}
	for _, key := range keys {
		if key == nil {
			continue
		}
		if key.KeyID == "" {
			continue
		}
		s.controlKeys[key.KeyID] = key
	}
	if key := s.controlKeys[activeID]; key != nil {
		s.controlKey = key
	}
	return s
}

func (s *Service) activeControlKey() *ControlKey {
	if s.controlKey != nil {
		return s.controlKey
	}
	for _, key := range s.controlKeys {
		return key
	}
	return nil
}

func (s *Service) controlKeyFor(computer *computertypes.Computer) *ControlKey {
	if computer != nil && computer.Metadata != nil {
		if id, ok := computer.Metadata["tai_control_key_id"].(string); ok && id != "" {
			if key := s.controlKeys[id]; key != nil {
				return key
			}
		}
	}
	return s.activeControlKey()
}

// WithServerURL sets the Yao server URL passed to tai start.
func (s *Service) WithServerURL(serverURL string) *Service {
	s.serverURL = serverURL
	return s
}

// Purchase creates a Tao host and immediately creates its Yao Computer.
func (s *Service) Purchase(ctx context.Context, taoKey string, req PurchaseRequest) (*PurchaseResult, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if strings.TrimSpace(req.Resource) == "" {
		return nil, fmt.Errorf("resource is required")
	}
	if s.client == nil || s.computers == nil {
		return nil, fmt.Errorf("tao computer service is not configured")
	}
	controlKey := s.activeControlKey()
	if controlKey == nil {
		return nil, ErrTaiControlKey
	}
	metadata := map[string]string{
		"computer_name":      req.Computer.Name,
		"tai_control_key_id": controlKey.KeyID,
	}
	for key, value := range req.Metadata {
		metadata[key] = value
	}
	instance, err := s.client.Create(ctx, taoKey, CreateInstanceRequest{
		Resource: req.Resource, Region: req.Region, Name: req.Name,
		BillingUnit: req.BillingUnit, PeriodMonths: req.PeriodMonths,
		RequestParams: req.RequestParams, SSH: &SSHRequest{PublicKey: controlKey.PublicKey}, Metadata: metadata, IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		return nil, err
	}
	result, err := s.computers.CreateOrUpdate(ctx, computertypes.CreateOrUpdateComputerRequest{
		InstanceID:     instance.InstanceID,
		Name:           computerName(req),
		ProviderType:   "cloud",
		Scope:          scopeFor(req),
		Visibility:     req.Computer.Visibility,
		Status:         computertypes.StatusProvisioning,
		UserID:         req.UserID,
		TeamID:         req.TeamID,
		DataDir:        req.Computer.DataDir,
		LeaseSource:    req.Computer.LeaseSource,
		LeaseExpiresAt: req.Computer.LeaseExpiresAt,
		Metadata: map[string]interface{}{
			"resource":           req.Resource,
			"region":             req.Region,
			"sandbox_image":      req.Computer.SandboxImage,
			"tai_control_key_id": controlKey.KeyID,
		},
	})
	if err != nil {
		return nil, err
	}
	return &PurchaseResult{Instance: instance, Computer: result.Computer}, nil
}

// Orchestrate advances an existing Tao instance's Computer lifecycle.
func (s *Service) Orchestrate(ctx context.Context, taoKey, instanceID string) (*PurchaseResult, error) {
	return s.Get(ctx, taoKey, instanceID)
}

// List returns Tao instances visible to the user key.
func (s *Service) List(ctx context.Context, taoKey string, opts ListOptions) ([]Instance, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if s.client == nil {
		return nil, fmt.Errorf("tao computer service is not configured")
	}
	return s.client.List(ctx, taoKey, opts)
}

// Get refreshes Tao state and advances the Computer lifecycle.
func (s *Service) Get(ctx context.Context, taoKey, instanceID string) (*PurchaseResult, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if s.client == nil || s.computers == nil {
		return nil, fmt.Errorf("tao computer service is not configured")
	}
	instance, err := s.client.Get(ctx, taoKey, instanceID)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			result, updateErr := s.computers.CreateOrUpdate(ctx, computertypes.CreateOrUpdateComputerRequest{InstanceID: instanceID, Status: computertypes.StatusReleased})
			if updateErr != nil {
				return nil, updateErr
			}
			return &PurchaseResult{Computer: result.Computer}, ErrInstanceDeleted
		}
		return nil, err
	}
	unlock := s.lockInstance(instanceID)
	defer unlock()
	status := computertypes.StatusProvisioning
	issueToken := false
	switch instance.Status {
	case "pending":
		status = computertypes.StatusProvisioning
	case "running":
		status = computertypes.StatusPending
		issueToken = true
	case "stopped":
		status = computertypes.StatusSuspended
	case "error":
		status = computertypes.StatusError
	case "deleted":
		status = computertypes.StatusReleased
	}
	result, err := s.computers.CreateOrUpdate(ctx, computertypes.CreateOrUpdateComputerRequest{
		InstanceID:     instanceID,
		Status:         status,
		IssueNodeToken: issueToken,
	})
	if err != nil {
		return nil, err
	}
	if issueToken && s.tai != nil {
		computer := result.Computer
		target := SSHTargetFrom(instance)
		controlKey := s.controlKeyFor(computer)
		if controlKey == nil {
			return nil, fmt.Errorf("%w: %v", ErrTaiStart, ErrTaiControlKey)
		}
		target.PrivateKey = controlKey.PrivateKey
		dataDir := ""
		if computer.Metadata != nil {
			dataDir, _ = computer.Metadata["data_dir"].(string)
		}
		if err := s.tai.Bootstrap(ctx, target, computertypes.BootstrapRequest{
			NodeID: computer.NodeID, NodeToken: result.NodeToken, Server: s.serverURL, DataDir: dataDir,
		}); err != nil {
			_, _ = s.computers.CreateOrUpdate(ctx, computertypes.CreateOrUpdateComputerRequest{InstanceID: instanceID, Status: computertypes.StatusError})
			return nil, fmt.Errorf("%w: %v", ErrTaiStart, err)
		}
		if err := s.tai.Start(ctx, target, StartTaiRequestFor(computer, result.NodeToken, s.serverURL)); err != nil {
			_, _ = s.computers.CreateOrUpdate(ctx, computertypes.CreateOrUpdateComputerRequest{
				InstanceID: instanceID, Status: computertypes.StatusError,
			})
			return nil, fmt.Errorf("%w: %v", ErrTaiStart, err)
		}
	}
	return &PurchaseResult{Instance: instance, Computer: result.Computer}, nil
}

// Action sends start/stop/reboot to Tao and advances Computer state.
func (s *Service) Action(ctx context.Context, taoKey, instanceID, action string) (*PurchaseResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.client == nil || s.computers == nil {
		return nil, fmt.Errorf("tao computer service is not configured")
	}
	if action != "start" && action != "stop" && action != "reboot" {
		return nil, fmt.Errorf("%w: unsupported action", ErrInvalidAction)
	}
	instance, err := s.client.Action(ctx, taoKey, instanceID, action)
	if err != nil {
		return nil, err
	}
	status := computertypes.StatusPending
	if action == "stop" {
		status = computertypes.StatusSuspended
	}
	result, err := s.computers.CreateOrUpdate(ctx, computertypes.CreateOrUpdateComputerRequest{
		InstanceID: instanceID, Status: status,
	})
	if err != nil {
		return nil, err
	}
	return &PurchaseResult{Instance: instance, Computer: result.Computer}, nil
}

// Delete deletes a Tao host and releases the Computer record.
func (s *Service) Delete(ctx context.Context, taoKey, instanceID string) (*PurchaseResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.client == nil || s.computers == nil {
		return nil, fmt.Errorf("tao computer service is not configured")
	}
	if err := s.client.Delete(ctx, taoKey, instanceID); err != nil {
		return nil, err
	}
	result, err := s.computers.CreateOrUpdate(ctx, computertypes.CreateOrUpdateComputerRequest{
		InstanceID: instanceID, Status: computertypes.StatusReleased,
	})
	if err != nil {
		return nil, err
	}
	return &PurchaseResult{Computer: result.Computer}, nil
}

// SSHTargetFrom builds an SSH target from a Tao instance.
func SSHTargetFrom(instance *Instance) computertypes.SSHTarget {
	if instance == nil {
		return computertypes.SSHTarget{}
	}
	return computertypes.SSHTarget{Host: instance.SSH.Host, Port: instance.SSH.Port, User: instance.SSH.User}
}

// StartTaiRequestFor builds the V2 tai start request.
func StartTaiRequestFor(computer *computertypes.Computer, token, serverURL string) computertypes.StartTaiRequest {
	dataDir := ""
	sandboxImage := ""
	if computer != nil && computer.Metadata != nil {
		if value, ok := computer.Metadata["data_dir"].(string); ok {
			dataDir = value
		}
		if value, ok := computer.Metadata["sandbox_image"].(string); ok {
			sandboxImage = value
		}
	}
	nodeID := ""
	if computer != nil {
		nodeID = computer.NodeID
	}
	return computertypes.StartTaiRequest{
		NodeID: nodeID, DataDir: dataDir, ServerURL: serverURL, NodeToken: token,
		SandboxImage: sandboxImage,
		LogLevel:     "info", LogFormat: "text", ShutdownTimeout: 30 * time.Second,
	}
}

func (s *Service) lockInstance(instanceID string) func() {
	value, _ := s.locks.LoadOrStore(instanceID, &sync.Mutex{})
	mu := value.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

func computerName(req PurchaseRequest) string {
	if req.Computer.Name != "" {
		return req.Computer.Name
	}
	if req.Name != "" {
		return req.Name
	}
	return "cloud-computer"
}

func scopeFor(req PurchaseRequest) string {
	if req.TeamID != "" {
		return computertypes.ScopeTeam
	}
	return computertypes.ScopeUser
}
