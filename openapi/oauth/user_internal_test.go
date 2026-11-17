package oauth

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

func TestBuildUserInfo(t *testing.T) {
	service := newUnitTestService(t)
	client := createUnitTestClient(t, service, "userinfo-client", nil)
	ctx := context.Background()

	subject, err := service.Subject(client.ClientID, "user-info")
	require.NoError(t, err)

	t.Run("claims carry auth_source and identity_source", func(t *testing.T) {
		info := service.BuildUserInfo(ctx, &types.TokenClaims{
			Subject:  subject,
			ClientID: client.ClientID,
			Scope:    "openid profile",
			Extra: map[string]interface{}{
				types.ClaimAuthSource:     types.AuthSourceTao,
				types.ClaimIdentitySource: types.IdentitySourceTao,
			},
		})
		require.NotNil(t, info)
		assert.Equal(t, subject, info.Sub)
		assert.Equal(t, types.AuthSourceTao, info.AuthSource)
		assert.Equal(t, types.IdentitySourceTao, info.IdentitySource)

		raw, err := json.Marshal(info)
		require.NoError(t, err)
		assert.Contains(t, string(raw), `"sub":"`+subject+`"`)
		assert.Contains(t, string(raw), `"auth_source":"tao"`)
		assert.Contains(t, string(raw), `"identity_source":"tao"`)
	})

	t.Run("no Team claims on userinfo", func(t *testing.T) {
		info := service.BuildUserInfo(ctx, &types.TokenClaims{
			Subject:  subject,
			ClientID: client.ClientID,
			Extra: map[string]interface{}{
				types.ClaimAuthSource: types.AuthSourceTao,
			},
		})
		raw, err := json.Marshal(info)
		require.NoError(t, err)
		for _, forbidden := range []string{"team_id", "team_ids", "tenant_id", "Team"} {
			assert.NotContains(t, string(raw), forbidden)
		}
	})

	t.Run("identity_source falls back to local for a password session", func(t *testing.T) {
		info := service.BuildUserInfo(ctx, &types.TokenClaims{
			Subject:  "",
			ClientID: "client-without-subject",
			Extra: map[string]interface{}{
				types.ClaimAuthSource: types.AuthSourcePassword,
			},
		})
		require.NotNil(t, info)
		assert.Equal(t, types.AuthSourcePassword, info.AuthSource)
		assert.Equal(t, types.IdentitySourceLocal, info.IdentitySource)
	})

	t.Run("nil claims return nil", func(t *testing.T) {
		assert.Nil(t, service.BuildUserInfo(ctx, nil))
	})
}
