// Package auth adapts Yao node tokens for Tai's authentication boundary.
package auth

import (
	"context"
	"errors"

	"github.com/yaoapp/yao/openapi/oauth"
)

// Provider validates Yang node tokens for Tai connections.
type Provider struct {
	Service interface {
		ValidateNodeToken(context.Context, string) (*oauth.NodeInfo, error)
	}
}

// New creates a Tai node-token provider.
func New(service interface {
	ValidateNodeToken(context.Context, string) (*oauth.NodeInfo, error)
}) *Provider {
	return &Provider{Service: service}
}

// Validate verifies a node token and returns the node identity.
func (p *Provider) Validate(ctx context.Context, raw string) (*oauth.NodeInfo, error) {
	if p == nil || p.Service == nil {
		return nil, errors.New("node token provider is not configured")
	}
	return p.Service.ValidateNodeToken(ctx, raw)
}
