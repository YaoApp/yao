package authorized

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/process"
)

func TestProcessAuthInfo(t *testing.T) {
	t.Run("WithNilProcess", func(t *testing.T) {
		result := ProcessAuthInfo(nil)
		assert.Nil(t, result)
	})

	t.Run("WithProcessNoAuth", func(t *testing.T) {
		p := &process.Process{}
		result := ProcessAuthInfo(p)
		// GetAuthorized returns empty struct instead of nil, so ProcessAuthInfo will return an empty AuthorizedInfo
		require.NotNil(t, result)
		assert.Empty(t, result.UserID)
		assert.Empty(t, result.TeamID)
		assert.Empty(t, result.Subject)
	})

	t.Run("WithProcessWithAuth", func(t *testing.T) {
		p := &process.Process{
			Authorized: &process.AuthorizedInfo{
				Subject:    "user123",
				ClientID:   "client456",
				UserID:     "u789",
				Scope:      "read write",
				TeamID:     "t123",
				TenantID:   "tenant456",
				SessionID:  "session789",
				RememberMe: true,
				Constraints: process.DataConstraints{
					OwnerOnly:   true,
					CreatorOnly: false,
					EditorOnly:  false,
					TeamOnly:    true,
					Extra: map[string]interface{}{
						"department": "engineering",
					},
				},
			},
		}

		result := ProcessAuthInfo(p)
		require.NotNil(t, result)

		assert.Equal(t, "user123", result.Subject)
		assert.Equal(t, "client456", result.ClientID)
		assert.Equal(t, "u789", result.UserID)
		assert.Equal(t, "read write", result.Scope)
		assert.Equal(t, "t123", result.TeamID)
		assert.Equal(t, "tenant456", result.TenantID)
		assert.Equal(t, "session789", result.SessionID)
		assert.True(t, result.RememberMe)

		assert.True(t, result.Constraints.OwnerOnly)
		assert.False(t, result.Constraints.CreatorOnly)
		assert.False(t, result.Constraints.EditorOnly)
		assert.True(t, result.Constraints.TeamOnly)
		assert.Equal(t, "engineering", result.Constraints.Extra["department"])
	})

	t.Run("WithPartialData", func(t *testing.T) {
		p := &process.Process{
			Authorized: &process.AuthorizedInfo{
				UserID: "u123",
				TeamID: "t456",
				Constraints: process.DataConstraints{
					TeamOnly: true,
				},
			},
		}

		result := ProcessAuthInfo(p)
		require.NotNil(t, result)

		assert.Equal(t, "u123", result.UserID)
		assert.Equal(t, "t456", result.TeamID)
		assert.True(t, result.Constraints.TeamOnly)
		assert.False(t, result.Constraints.OwnerOnly)
	})
}
