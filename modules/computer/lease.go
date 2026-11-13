package computer

import (
	"context"
	"fmt"
	"time"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/modules/computer/types"
	"github.com/yaoapp/yao/openapi/oauth"
)

// Renew extends a computer lease. Expired computers are moved back to pending.
func (s *Service) Renew(ctx context.Context, computerID string, req types.RenewLeaseRequest) (*types.RenewLeaseResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m, err := s.modelOrErr()
	if err != nil {
		return nil, err
	}
	current, err := s.Get(ctx, computerID)
	if err != nil {
		return nil, err
	}
	expires := req.ExpiresAt
	if expires == nil && req.Duration > 0 {
		value := time.Now().UTC().Add(req.Duration)
		expires = &value
	}
	if expires == nil {
		return nil, fmt.Errorf("%w: expires_at or duration is required", ErrInvalid)
	}
	update := map[string]interface{}{
		"lease_expires_at": *expires,
		"renewed_at":       time.Now().UTC(),
	}
	if req.LeaseSource != "" {
		update["lease_source"] = req.LeaseSource
	}
	wasExpired := current.Status == types.StatusExpired
	if wasExpired {
		update["status"] = types.StatusPending
	}
	if err := m.Update(computerID, update); err != nil {
		return nil, err
	}
	computer, err := s.Get(ctx, computerID)
	if err != nil {
		return nil, err
	}
	result := &types.RenewLeaseResult{Computer: computer}
	if wasExpired && req.IssueNodeToken {
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

// ExpireDue marks computers whose lease has expired and revokes their tokens.
func (s *Service) ExpireDue(ctx context.Context, now time.Time) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	m, err := s.modelOrErr()
	if err != nil {
		return 0, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	rows, err := m.Get(model.QueryParam{Wheres: []model.QueryWhere{{Column: "lease_expires_at", OP: "le", Value: now}}})
	if err != nil {
		return 0, err
	}
	if s.tokens == nil && oauth.OAuth != nil {
		s.tokens = oauth.OAuth
	}
	count := 0
	for _, row := range rows {
		computer := computerFromRow(row)
		if computer.Status == types.StatusExpired || computer.Status == types.StatusReleased {
			continue
		}
		if s.tokens != nil {
			if err := s.tokens.RevokeNodeTokenByComputer(ctx, computer.ID); err != nil {
				return count, err
			}
		}
		if err := m.Update(computer.ID, map[string]interface{}{"status": types.StatusExpired}); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}
