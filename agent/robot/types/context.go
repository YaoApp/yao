package types

import (
	"context"

	"github.com/yaoapp/yao/openapi/oauth/types"
)

// Context - robot execution context (lightweight)
type Context struct {
	context.Context                       // embed standard context
	Auth            *types.AuthorizedInfo `json:"auth,omitempty"`       // reuse oauth AuthorizedInfo
	MemberID        string                `json:"member_id,omitempty"`  // current robot member ID
	RequestID       string                `json:"request_id,omitempty"` // request trace ID
	Locale          string                `json:"locale,omitempty"`     // locale (e.g., "en-US")
}

// NewContext creates a new robot context
func NewContext(parent context.Context, auth *types.AuthorizedInfo) *Context {
	if parent == nil {
		parent = context.Background()
	}
	return &Context{
		Context: parent,
		Auth:    auth,
	}
}

// UserID returns user ID from auth
func (c *Context) UserID() string {
	if c.Auth == nil {
		return ""
	}
	return c.Auth.UserID
}

// TeamID returns team ID from auth
func (c *Context) TeamID() string {
	if c.Auth == nil {
		return ""
	}
	return c.Auth.TeamID
}
