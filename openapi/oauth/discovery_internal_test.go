package oauth

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoveryAdvertisesRPInitiatedLogout(t *testing.T) {
	service := newUnitTestService(t)
	ctx := context.Background()

	endpoints, err := service.Endpoints(ctx)
	require.NoError(t, err)
	assert.Contains(t, endpoints, "end_session_endpoint")
	assert.Equal(t, "https://oauth.unit.test/v1/oauth/logout", endpoints["end_session_endpoint"])
	assert.Equal(t, "https://oauth.unit.test/v1/oauth/authorize", endpoints["authorization_endpoint"])
	assert.Equal(t, "https://oauth.unit.test/v1/oauth/token", endpoints["token_endpoint"])

	metadata, err := service.GetServerMetadata(ctx)
	require.NoError(t, err)
	require.NotNil(t, metadata)
	raw, err := json.Marshal(metadata)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "end_session_endpoint")
	assert.Contains(t, string(raw), "prompt_values_supported")
	t.Run("discovery does not advertise Team claims", func(t *testing.T) {
		metadata, err := service.GetServerMetadata(ctx)
		require.NoError(t, err)
		raw, err := json.Marshal(metadata)
		require.NoError(t, err)
		for _, forbidden := range []string{"team_id", "team_ids", "tenant_id"} {
			assert.NotContains(t, string(raw), forbidden)
		}
	})
}
