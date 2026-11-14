// Package computer implements the local Computer module.
package computer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/modules/computer/types"
	"github.com/yaoapp/yao/openapi/oauth"
)

// ErrNotFound is returned when a computer does not exist.
var ErrNotFound = errors.New("computer not found")

// ErrConflict is returned when a node already exists.
var ErrConflict = errors.New("computer conflict")

// ErrInvalid is returned for invalid input.
var ErrInvalid = errors.New("invalid computer")

const modelName = "__yao.computers"

// computerStore is the persistence subset used by the Computer service.
type computerStore interface {
	Find(id interface{}, param model.QueryParam) (maps.MapStr, error)
	Get(param model.QueryParam) ([]maps.MapStr, error)
	Insert(columns []string, rows [][]interface{}) error
	Update(id interface{}, row maps.MapStrAny) error
	Delete(id interface{}) error
}

type modelStore struct {
	model *model.Model
}

func (s *modelStore) Find(id interface{}, param model.QueryParam) (maps.MapStr, error) {
	return s.model.Find(id, param)
}

func (s *modelStore) Get(param model.QueryParam) ([]maps.MapStr, error) {
	return s.model.Get(param)
}

func (s *modelStore) Insert(columns []string, rows [][]interface{}) error {
	return s.model.Insert(columns, rows)
}

func (s *modelStore) Update(id interface{}, row maps.MapStrAny) error {
	return s.model.Update(id, row)
}

func (s *modelStore) Delete(id interface{}) error {
	return s.model.Delete(id)
}

// Service owns computer persistence and local connection lifecycle.
type Service struct {
	store    computerStore
	tokens   NodeTokenService
	taiNodes types.TaiNodeProvider
}

// WithTaiNodeProvider attaches the Tai node registry provider.
func (s *Service) WithTaiNodeProvider(provider types.TaiNodeProvider) *Service {
	s.taiNodes = provider
	return s
}

// New creates a computer service using the loaded application model.
func New() *Service {
	service := &Service{}
	if model.Exists(modelName) {
		service.store = &modelStore{model: model.Select(modelName)}
	}
	return service
}

func (s *Service) modelOrErr() (computerStore, error) {
	if s.store == nil && model.Exists(modelName) {
		s.store = &modelStore{model: model.Select(modelName)}
	}
	if s.store == nil {
		return nil, fmt.Errorf("%w: computer model is not loaded", ErrInvalid)
	}
	return s.store, nil
}

// Create inserts a computer row. It is used by Connect and cloud provisioning.
func (s *Service) Create(ctx context.Context, req types.CreateComputerRequest) (*types.Computer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req.NodeID == "" || req.Name == "" {
		return nil, fmt.Errorf("%w: node_id and name are required", ErrInvalid)
	}
	if req.Scope == "" {
		req.Scope = types.ScopeUser
	}
	if req.Visibility == "" {
		req.Visibility = types.VisibilityPrivate
	}
	if req.Status == "" {
		req.Status = types.StatusPending
	}
	if req.ProviderType == "" {
		req.ProviderType = "local"
	}
	if req.ID == "" {
		req.ID = NewComputerID()
	}
	m, err := s.modelOrErr()
	if err != nil {
		return nil, err
	}
	metadata := map[string]interface{}{}
	for k, v := range req.Metadata {
		metadata[k] = v
	}
	if req.DataDir != "" {
		metadata["data_dir"] = req.DataDir
	}
	row := map[string]interface{}{
		"id":               req.ID,
		"node_id":          req.NodeID,
		"instance_id":      req.InstanceID,
		"name":             req.Name,
		"provider_type":    req.ProviderType,
		"scope":            req.Scope,
		"visibility":       req.Visibility,
		"status":           req.Status,
		"__yao_created_by": req.UserID,
		"__yao_team_id":    req.TeamID,
		"metadata":         metadata,
	}
	columns := []string{"id", "node_id", "name", "provider_type", "scope", "visibility", "status", "__yao_created_by", "__yao_team_id", "metadata"}
	values := [][]interface{}{{row["id"], row["node_id"], row["name"], row["provider_type"], row["scope"], row["visibility"], row["status"], row["__yao_created_by"], row["__yao_team_id"], row["metadata"]}}
	if req.InstanceID != "" {
		columns = append(columns, "instance_id")
		values[0] = append(values[0], req.InstanceID)
	}
	if req.LeaseSource != "" {
		columns = append(columns, "lease_source")
		values[0] = append(values[0], req.LeaseSource)
	}
	if req.LeaseExpiresAt != nil {
		columns = append(columns, "lease_expires_at")
		values[0] = append(values[0], *req.LeaseExpiresAt)
	}
	if err := m.Insert(columns, values); err != nil {
		return nil, fmt.Errorf("insert computer: %w", err)
	}
	return s.Get(ctx, req.ID)
}

// List returns computers matching the filter.
func (s *Service) List(ctx context.Context, filter types.ListFilter) ([]types.Computer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m, err := s.modelOrErr()
	if err != nil {
		return nil, err
	}
	param := model.QueryParam{}
	if filter.Scope != "" {
		param.Wheres = append(param.Wheres, model.QueryWhere{Column: "scope", Value: filter.Scope})
	}
	if filter.Visibility != "" {
		param.Wheres = append(param.Wheres, model.QueryWhere{Column: "visibility", Value: filter.Visibility})
	}
	if filter.Status != "" {
		param.Wheres = append(param.Wheres, model.QueryWhere{Column: "status", Value: filter.Status})
	}
	if filter.UserID != "" {
		param.Wheres = append(param.Wheres, model.QueryWhere{Column: "__yao_created_by", Value: filter.UserID})
	}
	if filter.TeamID != "" {
		param.Wheres = append(param.Wheres, model.QueryWhere{Column: "__yao_team_id", Value: filter.TeamID})
	}
	param.Orders = []model.QueryOrder{{Column: "position", Option: "asc"}, {Column: "created_at", Option: "desc"}}
	if filter.Page > 0 {
		param.Page = filter.Page
		if filter.PageSize > 0 {
			param.PageSize = filter.PageSize
		}
	}
	rows, err := m.Get(param)
	if err != nil {
		return nil, err
	}
	out := make([]types.Computer, 0, len(rows))
	for _, row := range rows {
		out = append(out, computerFromRow(row))
	}
	return out, nil
}

// Get returns one computer.
// GetByNodeID returns the Computer for a node ID.
func (s *Service) GetByNodeID(ctx context.Context, nodeID string) (*types.Computer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(nodeID) == "" {
		return nil, fmt.Errorf("%w: node_id is required", ErrInvalid)
	}
	m, err := s.modelOrErr()
	if err != nil {
		return nil, err
	}
	rows, err := m.Get(modelQueryBy("node_id", nodeID))
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	computer := computerFromRow(rows[0])
	return &computer, nil
}

// Get returns one Computer by ID.
func (s *Service) Get(ctx context.Context, computerID string) (*types.Computer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if computerID == "" {
		return nil, fmt.Errorf("%w: computer_id is required", ErrInvalid)
	}
	m, err := s.modelOrErr()
	if err != nil {
		return nil, err
	}
	row, err := m.Find(computerID, model.QueryParam{})
	if err != nil {
		return nil, ErrNotFound
	}
	c := computerFromRow(row)
	return &c, nil
}

// Update updates mutable computer fields.
func (s *Service) Update(ctx context.Context, computerID string, req types.UpdateRequest) (*types.Computer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req.Visibility != nil && !validVisibility(*req.Visibility) {
		return nil, fmt.Errorf("%w: invalid visibility", ErrInvalid)
	}
	m, err := s.modelOrErr()
	if err != nil {
		return nil, err
	}
	row := map[string]interface{}{}
	if req.Name != nil {
		row["name"] = *req.Name
	}
	if req.Visibility != nil {
		row["visibility"] = *req.Visibility
	}
	if len(row) == 0 {
		return s.Get(ctx, computerID)
	}
	if err := m.Update(computerID, row); err != nil {
		return nil, err
	}
	return s.Get(ctx, computerID)
}

// Delete removes a computer and revokes its active node token.
func (s *Service) Delete(ctx context.Context, computerID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m, err := s.modelOrErr()
	if err != nil {
		return err
	}
	if _, err := s.Get(ctx, computerID); err != nil {
		return err
	}
	if s.tokens == nil && oauth.OAuth != nil {
		s.tokens = oauth.OAuth
	}
	if s.tokens != nil {
		if err := s.tokens.RevokeNodeTokenByComputer(ctx, computerID); err != nil {
			return err
		}
	}
	return m.Delete(computerID)
}

// CreateOrUpdate upserts a Computer by computer_id or instance_id.
func (s *Service) CreateOrUpdate(ctx context.Context, req types.CreateOrUpdateComputerRequest) (*types.UpsertResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req.InstanceID == "" && req.ID == "" {
		return nil, fmt.Errorf("%w: computer_id or instance_id is required", ErrInvalid)
	}
	if req.ProviderType == "" {
		req.ProviderType = "cloud"
	}
	if req.Status == "" {
		req.Status = types.StatusProvisioning
	}
	m, err := s.modelOrErr()
	if err != nil {
		return nil, err
	}

	var rows []maps.MapStr
	if req.ID != "" {
		if row, findErr := m.Find(req.ID, model.QueryParam{}); findErr == nil {
			rows = []maps.MapStr{row}
		}
	}
	if len(rows) == 0 && req.InstanceID != "" {
		rows, err = m.Get(modelQueryBy("instance_id", req.InstanceID))
		if err != nil {
			return nil, err
		}
	}
	var computer *types.Computer
	if len(rows) == 0 {
		nodeID := req.NodeID
		if nodeID == "" {
			nodeID = NewNodeID()
		}
		name := req.Name
		if name == "" {
			if req.InstanceID != "" {
				name = "cloud-" + req.InstanceID
			} else {
				name = "cloud-" + req.ID
			}
		}
		computer, err = s.Create(ctx, types.CreateComputerRequest{
			ID: req.ID, NodeID: nodeID, InstanceID: req.InstanceID,
			Name: name, ProviderType: req.ProviderType, Scope: req.Scope,
			Visibility: req.Visibility, Status: req.Status, UserID: req.UserID,
			TeamID: req.TeamID, DataDir: req.DataDir, Metadata: req.Metadata,
			LeaseSource: req.LeaseSource, LeaseExpiresAt: req.LeaseExpiresAt,
		})
		if err != nil {
			return nil, err
		}
	} else {
		existing := computerFromRow(rows[0])
		computer = &existing
		update := map[string]interface{}{}
		if req.InstanceID != "" && req.InstanceID != computer.InstanceID {
			update["instance_id"] = req.InstanceID
		}
		if req.NodeID != "" && req.NodeID != computer.NodeID {
			update["node_id"] = req.NodeID
		}
		if req.Name != "" && req.Name != computer.Name {
			update["name"] = req.Name
		}
		if req.Status != "" && req.Status != computer.Status {
			update["status"] = req.Status
		}
		if req.LeaseSource != "" {
			update["lease_source"] = req.LeaseSource
		}
		if req.LeaseExpiresAt != nil {
			update["lease_expires_at"] = *req.LeaseExpiresAt
		}
		if len(update) > 0 {
			if err := m.Update(computer.ID, update); err != nil {
				return nil, err
			}
			computer, err = s.Get(ctx, computer.ID)
			if err != nil {
				return nil, err
			}
		}
	}
	result := &types.UpsertResult{Computer: computer}
	if req.IssueNodeToken {
		if s.tokens == nil && oauth.OAuth != nil {
			s.tokens = oauth.OAuth
		}
		if s.tokens == nil {
			return nil, fmt.Errorf("%w: node token service is not configured", ErrInvalid)
		}
		token, err := s.tokens.IssueNodeToken(ctx, oauth.IssueRequest{
			ComputerID: computer.ID, NodeID: computer.NodeID, Scope: computer.Scope,
			Visibility: computer.Visibility, UserID: computer.UserID, TeamID: computer.TeamID,
		})
		if err != nil {
			return nil, fmt.Errorf("issue node token: %w", err)
		}
		result.NodeToken = token.Token
		result.ExpiresAt = token.ExpiresAt
	}
	return result, nil
}

func timeValue(value interface{}) *time.Time {
	switch v := value.(type) {
	case time.Time:
		return &v
	case *time.Time:
		return v
	case string:
		if v == "" {
			return nil
		}
		for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05"} {
			if t, err := time.Parse(layout, v); err == nil {
				return &t
			}
		}
	}
	return nil
}

func computerFromRow(row map[string]interface{}) types.Computer {
	c := types.Computer{
		ID:           stringValue(row, "id"),
		NodeID:       stringValue(row, "node_id"),
		InstanceID:   stringValue(row, "instance_id"),
		Name:         stringValue(row, "name"),
		ProviderType: stringValue(row, "provider_type"),
		Scope:        stringValue(row, "scope"),
		Visibility:   stringValue(row, "visibility"),
		Status:       stringValue(row, "status"),
		UserID:       firstString(row, "__yao_created_by", "__yao_user_id"),
		TeamID:       stringValue(row, "__yao_team_id"),
	}
	if metadata, ok := row["metadata"].(map[string]interface{}); ok {
		c.Metadata = metadata
		if value, ok := metadata["data_dir"].(string); ok {
			c.DataDir = value
		}
	}
	c.LeaseSource = stringValue(row, "lease_source")
	c.LeaseExpiresAt = timeValue(row["lease_expires_at"])
	c.RenewedAt = timeValue(row["renewed_at"])
	if value := stringValue(row, "created_at"); value != "" {
		c.CreatedAt, _ = time.Parse(time.RFC3339, value)
	}
	return c
}

func stringValue(row map[string]interface{}, key string) string {
	if row[key] == nil {
		return ""
	}
	if value, ok := row[key].(string); ok {
		return value
	}
	return fmt.Sprint(row[key])
}

func firstString(row map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value := stringValue(row, key); value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

func validVisibility(value string) bool {
	switch value {
	case types.VisibilityPrivate, types.VisibilityMembers, types.VisibilityTeam, types.VisibilityGlobal:
		return true
	default:
		return false
	}
}

func nowUnixNano() int64 { return time.Now().UnixNano() }
