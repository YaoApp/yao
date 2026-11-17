package types_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// =============================================================================
// Phase 2: Yao OAuth Server client authorization types
// =============================================================================

func TestAuthorizationRequestServerResolvedIdentityNotSerialized(t *testing.T) {
	request := &types.AuthorizationRequest{
		ClientID:       "client-1",
		ResponseType:   "code",
		RedirectURI:    "https://client.example.com/cb",
		State:          "state-1",
		Subject:        "opaque-subject",
		AuthSource:     "tao",
		IdentitySource: "tao",
	}

	raw, err := json.Marshal(request)
	require.NoError(t, err)

	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, "client-1", decoded["client_id"])
	assert.Equal(t, "code", decoded["response_type"])
	for _, key := range []string{"Subject", "subject", "AuthSource", "auth_source", "identity_source"} {
		assert.NotContains(t, decoded, key, "server-resolved identity must never be serialized")
	}

	// Identity values supplied by a client must be ignored.
	var parsed types.AuthorizationRequest
	require.NoError(t, json.Unmarshal([]byte(`{"client_id":"client-1","subject":"injected","auth_source":"tao"}`), &parsed))
	assert.Empty(t, parsed.Subject)
	assert.Empty(t, parsed.AuthSource)
	assert.Empty(t, parsed.IdentitySource)
}

func TestOIDCUserInfoIdentityClaimsJSON(t *testing.T) {
	info := &types.OIDCUserInfo{
		Sub:            "opaque-subject",
		AuthSource:     types.AuthSourceTao,
		IdentitySource: types.IdentitySourceTao,
	}

	raw, err := json.Marshal(info)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"auth_source":"tao"`)
	assert.Contains(t, string(raw), `"identity_source":"tao"`)
	for _, forbidden := range []string{"team_id", "team_ids", "tenant_id"} {
		assert.NotContains(t, string(raw), forbidden)
	}

	var decoded types.OIDCUserInfo
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, types.AuthSourceTao, decoded.AuthSource)
	assert.Equal(t, types.IdentitySourceTao, decoded.IdentitySource)
}

func TestIdentityClaimConstants(t *testing.T) {
	assert.Equal(t, "auth_source", types.ClaimAuthSource)
	assert.Equal(t, "identity_source", types.ClaimIdentitySource)

	assert.Equal(t, "tao", types.AuthSourceTao)
	assert.Equal(t, "password", types.AuthSourcePassword)
	assert.Equal(t, "tao", types.IdentitySourceTao)
	assert.Equal(t, "local", types.IdentitySourceLocal)

	assert.Equal(t, "login_required", types.ErrorLoginRequired)
	require.NotNil(t, types.ErrLoginRequired)
	assert.Equal(t, types.ErrorLoginRequired, types.ErrLoginRequired.Code)
}
