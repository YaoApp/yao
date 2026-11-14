package computer

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/modules/computer/types"
	"github.com/yaoapp/yao/openapi/oauth"
)

// NodeTokenService is the node-token subset used by Connect/Delete.
type NodeTokenService interface {
	IssueNodeToken(context.Context, oauth.IssueRequest) (*oauth.NodeToken, error)
	RevokeNodeTokenByComputer(context.Context, string) error
}

// WithTokens sets the node token service (useful for tests).
func (s *Service) WithTokens(tokens NodeTokenService) *Service {
	s.tokens = tokens
	return s
}

// Connect creates a local computer and issues its first node token.
func (s *Service) Connect(ctx context.Context, req types.ConnectRequest) (*types.ConnectResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req.Name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalid)
	}
	if req.Visibility == "" {
		req.Visibility = types.VisibilityPrivate
	}
	if !validVisibility(req.Visibility) {
		return nil, fmt.Errorf("%w: invalid visibility", ErrInvalid)
	}
	if req.Scope == "" {
		req.Scope = types.ScopeUser
	}
	if req.Scope == types.ScopeUser && req.UserID == "" {
		return nil, fmt.Errorf("%w: user_id is required", ErrInvalid)
	}
	if req.Scope == types.ScopeTeam && req.TeamID == "" {
		return nil, fmt.Errorf("%w: team_id is required", ErrInvalid)
	}
	m, err := s.modelOrErr()
	if err != nil {
		return nil, err
	}
	var nodeID string
	for i := 0; i < 10; i++ {
		candidate := NewNodeID()
		rows, err := m.Get(modelQueryBy("node_id", candidate))
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			nodeID = candidate
			break
		}
	}
	if nodeID == "" {
		return nil, fmt.Errorf("%w: cannot allocate node_id", ErrConflict)
	}
	computer, err := s.Create(ctx, types.CreateComputerRequest{
		NodeID:         nodeID,
		Name:           req.Name,
		Scope:          req.Scope,
		Visibility:     req.Visibility,
		Status:         types.StatusPending,
		UserID:         req.UserID,
		TeamID:         req.TeamID,
		DataDir:        req.DataDir,
		LeaseSource:    req.LeaseSource,
		LeaseExpiresAt: req.LeaseExpiresAt,
	})
	if err != nil {
		return nil, err
	}
	if s.tokens == nil && oauth.OAuth != nil {
		s.tokens = oauth.OAuth
	}
	if s.tokens == nil {
		_ = s.Delete(ctx, computer.ID)
		return nil, fmt.Errorf("%w: node token service is not configured", ErrInvalid)
	}
	token, err := s.tokens.IssueNodeToken(ctx, oauth.IssueRequest{
		ComputerID: computer.ID,
		NodeID:     computer.NodeID,
		Scope:      computer.Scope,
		Visibility: computer.Visibility,
		UserID:     computer.UserID,
		TeamID:     computer.TeamID,
	})
	if err != nil {
		_ = s.Delete(ctx, computer.ID)
		return nil, fmt.Errorf("issue node token: %w", err)
	}
	result := &types.ConnectResult{
		Computer:      computer,
		ComputerID:    computer.ID,
		NodeID:        computer.NodeID,
		NodeToken:     token.Token,
		ExpiresAt:     token.ExpiresAt,
		TransportPort: transportPort(),
		Protocols:     []string{"HTTP/2", "HTTP/3"},
	}
	host := config.Conf.Host
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	result.Server = fmt.Sprintf("http://%s:%d", host, config.Conf.Port)
	return result, nil
}

// Disconnect marks an active computer disconnected while retaining its token.
// For a registered Tai node it also sends GoAway before updating local state.
func (s *Service) Disconnect(ctx context.Context, computerID string) error {
	if controller, ok := s.taiNodes.(types.TaiNodeController); ok {
		computer, err := s.Get(ctx, computerID)
		if err != nil {
			return err
		}
		if err := controller.GoAway(ctx, computer.NodeID, "disconnect requested"); err != nil {
			return err
		}
	}
	return s.setStatus(ctx, computerID, types.StatusDisconnected)
}

// Reconnect moves a disconnected/offline computer back to pending.
func (s *Service) Reconnect(ctx context.Context, computerID string) error {
	return s.setStatus(ctx, computerID, types.StatusPending)
}

func (s *Service) setStatus(ctx context.Context, computerID, status string) error {
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
	if err := m.Update(computerID, map[string]interface{}{"status": status}); err != nil {
		return err
	}
	return nil
}

func modelQueryBy(column string, value interface{}) model.QueryParam {
	return model.QueryParam{Wheres: []model.QueryWhere{{Column: column, Value: value}}}
}

func transportPort() int {
	if config.Conf.Tai.Server.TransportPort > 0 {
		return config.Conf.Tai.Server.TransportPort
	}
	return 6099
}
