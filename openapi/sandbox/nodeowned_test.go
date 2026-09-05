package sandbox_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi/oauth/types"
	sandbox "github.com/yaoapp/yao/openapi/sandbox"
	taitypes "github.com/yaoapp/yao/tai/types"
)

func TestNodeOwnedBy_NilAuthInfo(t *testing.T) {
	snap := &taitypes.NodeMeta{}
	assert.False(t, sandbox.ExportNodeOwnedBy(snap, nil),
		"nil authInfo must deny access")
}

func TestNodeOwnedBy_TeamMatch(t *testing.T) {
	snap := &taitypes.NodeMeta{
		Auth: taitypes.AuthInfo{TeamID: "team-abc"},
	}
	authInfo := &types.AuthorizedInfo{TeamID: "team-abc"}
	assert.True(t, sandbox.ExportNodeOwnedBy(snap, authInfo))
}

func TestNodeOwnedBy_TeamMismatch(t *testing.T) {
	snap := &taitypes.NodeMeta{
		Auth: taitypes.AuthInfo{TeamID: "team-abc"},
	}
	authInfo := &types.AuthorizedInfo{TeamID: "team-other"}
	assert.False(t, sandbox.ExportNodeOwnedBy(snap, authInfo))
}

func TestNodeOwnedBy_UserMatch(t *testing.T) {
	snap := &taitypes.NodeMeta{
		Auth: taitypes.AuthInfo{UserID: "user1"},
	}
	authInfo := &types.AuthorizedInfo{UserID: "user1"}
	assert.True(t, sandbox.ExportNodeOwnedBy(snap, authInfo))
}

func TestNodeOwnedBy_UserMismatch(t *testing.T) {
	snap := &taitypes.NodeMeta{
		Auth: taitypes.AuthInfo{UserID: "user1"},
	}
	authInfo := &types.AuthorizedInfo{UserID: "user-other"}
	assert.False(t, sandbox.ExportNodeOwnedBy(snap, authInfo))
}

func TestNodeOwnedBy_NoTeamNoUser(t *testing.T) {
	snap := &taitypes.NodeMeta{}
	authInfo := &types.AuthorizedInfo{}
	assert.True(t, sandbox.ExportNodeOwnedBy(snap, authInfo),
		"empty authInfo (no team/user) falls through to default allow")
}
