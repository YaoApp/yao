package oauth

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// captureLogOutput redirects the global logger to an in-memory buffer and
// restores the previous configuration when the test finishes.
func captureLogOutput(t *testing.T) *bytes.Buffer {
	t.Helper()

	std := log.StandardLogger()
	previousOut := std.Out
	previousFormatter := std.Formatter
	previousLevel := log.GetLevel()

	buffer := &bytes.Buffer{}
	log.SetFormatter(log.JSON)
	log.SetOutput(buffer)
	log.SetLevel(log.InfoLevel)

	t.Cleanup(func() {
		log.SetOutput(previousOut)
		std.SetFormatter(previousFormatter)
		log.SetLevel(previousLevel)
	})

	return buffer
}

// =============================================================================
// Dynamic Client Registration: audit
// =============================================================================

func TestRecordDynamicClientRegistrationAudit(t *testing.T) {
	buffer := captureLogOutput(t)

	request := &types.DynamicClientRegistrationRequest{
		ClientName:      "Audited Client",
		RedirectURIs:    []string{"https://localhost/callback"},
		GrantTypes:      []string{types.GrantTypeAuthorizationCode},
		ResponseTypes:   []string{types.ResponseTypeCode},
		Scope:           "openid profile",
		ApplicationType: types.ApplicationTypeNative,
	}

	t.Run("successful registration is audited", func(t *testing.T) {
		buffer.Reset()
		RecordDynamicClientRegistrationAudit(request, &types.ClientInfo{
			ClientID:   "client-audit",
			ClientName: "Audited Client",
		}, "", "203.0.113.9", "unit-test-agent")

		output := buffer.String()
		for _, expected := range []string{
			`"event":"oauth.dynamic_client_registration"`,
			`"result":"success"`,
			`"client_id":"client-audit"`,
			`"client_name":"Audited Client"`,
			`"redirect_uris":["https://localhost/callback"]`,
			`"grant_types":["authorization_code"]`,
			`"response_types":["code"]`,
			`"scope":"openid profile"`,
			`"application_type":"native"`,
			`"remote_ip":"203.0.113.9"`,
			`"user_agent":"unit-test-agent"`,
		} {
			assert.Contains(t, output, expected)
		}
		assert.NotContains(t, output, `"error"`, "successful audit must not carry an error field")
	})

	t.Run("failed registration is audited", func(t *testing.T) {
		buffer.Reset()
		RecordDynamicClientRegistrationAudit(nil, nil, "invalid_client_metadata", "198.51.100.7", "bad-agent")

		output := buffer.String()
		assert.Contains(t, output, `"result":"failure"`)
		assert.Contains(t, output, `"error":"invalid_client_metadata"`)
		assert.Contains(t, output, `"remote_ip":"198.51.100.7"`)
		assert.Contains(t, output, `"user_agent":"bad-agent"`)
	})
}

func TestBootstrapClients(t *testing.T) {
	service := newUnitTestService(t)
	ctx := context.Background()
	clients := []types.ClientInfo{
		{
			ClientID: "bootstrap-web-0123456789abcdef", ClientName: "Web", ClientType: types.ClientTypeConfidential,
			RedirectURIs:  []string{"https://app.example.com/callback"},
			GrantTypes:    []string{types.GrantTypeAuthorizationCode, types.GrantTypeRefreshToken},
			ResponseTypes: []string{types.ResponseTypeCode}, Scope: "openid profile",
		},
		{
			ClientID: "bootstrap-desktop-0123456789ab", ClientName: "Desktop", ClientType: types.ClientTypePublic,
			RedirectURIs:  []string{"http://127.0.0.1:5173/callback"},
			GrantTypes:    []string{types.GrantTypeAuthorizationCode, types.GrantTypeRefreshToken},
			ResponseTypes: []string{types.ResponseTypeCode}, Scope: "openid profile",
		},
	}
	require.NoError(t, service.BootstrapClients(ctx, clients))
	web, err := service.clientProvider.GetClientByID(ctx, clients[0].ClientID)
	require.NoError(t, err)
	require.Equal(t, types.ClientTypeConfidential, web.ClientType)
	require.True(t, web.RequirePKCE)
	require.NotEmpty(t, web.ClientSecret)
	desktop, err := service.clientProvider.GetClientByID(ctx, clients[1].ClientID)
	require.NoError(t, err)
	require.Equal(t, types.ClientTypePublic, desktop.ClientType)
	require.Empty(t, desktop.ClientSecret)
	require.True(t, desktop.RequirePKCE)
	// Idempotent: second bootstrap must not overwrite the existing client.
	require.NoError(t, service.BootstrapClients(ctx, clients))
	_, err = service.clientProvider.GetClientByID(ctx, clients[0].ClientID)
	require.NoError(t, err)
}
