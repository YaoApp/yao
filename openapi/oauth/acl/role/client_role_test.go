package role_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/openapi/oauth/acl/role"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// mockClientProvider implements types.ClientProvider for testing.
type mockClientProvider struct {
	clients map[string]*types.ClientInfo
}

func (m *mockClientProvider) GetClientByID(_ context.Context, clientID string) (*types.ClientInfo, error) {
	c, ok := m.clients[clientID]
	if !ok {
		return nil, fmt.Errorf("client %s not found", clientID)
	}
	return c, nil
}

func (m *mockClientProvider) GetClientByCredentials(_ context.Context, id, secret string) (*types.ClientInfo, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockClientProvider) CreateClient(_ context.Context, ci *types.ClientInfo) (*types.ClientInfo, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockClientProvider) UpdateClient(_ context.Context, id string, ci *types.ClientInfo) (*types.ClientInfo, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockClientProvider) DeleteClient(_ context.Context, id string) error {
	return fmt.Errorf("not implemented")
}
func (m *mockClientProvider) ValidateClient(_ context.Context, ci *types.ClientInfo) (*types.ValidationResult, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockClientProvider) ListClients(_ context.Context, f map[string]interface{}, l, o int) ([]*types.ClientInfo, int, error) {
	return nil, 0, fmt.Errorf("not implemented")
}
func (m *mockClientProvider) ValidateRedirectURI(_ context.Context, id, uri string) (*types.ValidationResult, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockClientProvider) ValidateScope(_ context.Context, id string, scopes []string) (*types.ValidationResult, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockClientProvider) IsClientActive(_ context.Context, id string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

func TestGetClientRole_NilProvider(t *testing.T) {
	mgr := role.NewManager(nil, nil, nil)
	r, err := mgr.GetClientRole(context.Background(), "anything")
	require.NoError(t, err)
	assert.Equal(t, "client:free", r)
}

func TestGetClientRole_ClientNotFound(t *testing.T) {
	provider := &mockClientProvider{clients: map[string]*types.ClientInfo{}}
	mgr := role.NewManager(nil, nil, provider)
	_, err := mgr.GetClientRole(context.Background(), "forged-client-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve client role for forged-client-id")
}

func TestGetClientRole_EmptyRole_ReturnsFree(t *testing.T) {
	provider := &mockClientProvider{
		clients: map[string]*types.ClientInfo{
			"existing-client": {ClientID: "existing-client", Role: ""},
		},
	}
	mgr := role.NewManager(nil, nil, provider)
	r, err := mgr.GetClientRole(context.Background(), "existing-client")
	require.NoError(t, err)
	assert.Equal(t, "client:free", r)
}

func TestGetClientRole_ExplicitRole(t *testing.T) {
	provider := &mockClientProvider{
		clients: map[string]*types.ClientInfo{
			"admin-client": {ClientID: "admin-client", Role: "system:root"},
		},
	}
	mgr := role.NewManager(nil, nil, provider)
	r, err := mgr.GetClientRole(context.Background(), "admin-client")
	require.NoError(t, err)
	assert.Equal(t, "system:root", r)
}
