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

// mockErrorProvider always returns a non-ErrorResponse error for GetClientByID.
type mockErrorProvider struct {
	err error
}

func (m *mockErrorProvider) GetClientByID(_ context.Context, _ string) (*types.ClientInfo, error) {
	return nil, m.err
}
func (m *mockErrorProvider) GetClientByCredentials(_ context.Context, _, _ string) (*types.ClientInfo, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockErrorProvider) CreateClient(_ context.Context, _ *types.ClientInfo) (*types.ClientInfo, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockErrorProvider) UpdateClient(_ context.Context, _ string, _ *types.ClientInfo) (*types.ClientInfo, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockErrorProvider) DeleteClient(_ context.Context, _ string) error {
	return fmt.Errorf("not implemented")
}
func (m *mockErrorProvider) ValidateClient(_ context.Context, _ *types.ClientInfo) (*types.ValidationResult, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockErrorProvider) ListClients(_ context.Context, _ map[string]interface{}, _, _ int) ([]*types.ClientInfo, int, error) {
	return nil, 0, fmt.Errorf("not implemented")
}
func (m *mockErrorProvider) ValidateRedirectURI(_ context.Context, _, _ string) (*types.ValidationResult, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockErrorProvider) ValidateScope(_ context.Context, _ string, _ []string) (*types.ValidationResult, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockErrorProvider) IsClientActive(_ context.Context, _ string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

func (m *mockClientProvider) GetClientByID(_ context.Context, clientID string) (*types.ClientInfo, error) {
	c, ok := m.clients[clientID]
	if !ok {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidClient,
			ErrorDescription: "Client not found",
		}
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
	r, err := mgr.GetClientRole(context.Background(), "forged-client-id")
	require.NoError(t, err)
	assert.Equal(t, "client:free", r)
}

func TestGetClientRole_ProviderInternalError(t *testing.T) {
	provider := &mockErrorProvider{err: fmt.Errorf("database connection lost")}
	mgr := role.NewManager(nil, nil, provider)
	_, err := mgr.GetClientRole(context.Background(), "any-client")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection lost")
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
