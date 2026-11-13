// Package computer implements the local Computer module.
package computer

import (
	"context"
	"errors"
	"fmt"
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
	store  computerStore
	tokens NodeTokenService
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

func computerFromRow(row map[string]interface{}) types.Computer {
	c := types.Computer{
		ID:           stringValue(row, "id"),
		NodeID:       stringValue(row, "node_id"),
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
	if value := stringValue(row, "created_at"); value != "" {
		c.CreatedAt, _ = time.Parse(time.RFC3339, value)
	}
	return c
}

func stringValue(row map[string]interface{}, key string) string {
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
