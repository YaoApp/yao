package grpc_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	yaogrpc "github.com/yaoapp/yao/tai/grpc"
)

// ── TokenManager unit tests ──────────────────────────────────────────────────

func TestTokenManager_AttachMetadata_WithAllFields(t *testing.T) {
	tm := yaogrpc.NewTokenManager("tok", "ref", "sb-1", "yao:9099")
	ctx := tm.AttachMetadata(context.Background())

	md, ok := metadata.FromOutgoingContext(ctx)
	require.True(t, ok)

	assert.Equal(t, []string{"Bearer tok"}, md.Get("authorization"))
	assert.Equal(t, []string{"ref"}, md.Get("x-refresh-token"))
	assert.Equal(t, []string{"sb-1"}, md.Get("x-sandbox-id"))
	assert.Equal(t, []string{"yao:9099"}, md.Get("x-grpc-upstream"))
}

func TestTokenManager_AttachMetadata_DirectMode(t *testing.T) {
	tm := yaogrpc.NewTokenManager("tok", "ref", "sb-1", "")
	ctx := tm.AttachMetadata(context.Background())

	md, ok := metadata.FromOutgoingContext(ctx)
	require.True(t, ok)

	assert.Equal(t, []string{"Bearer tok"}, md.Get("authorization"))
	assert.Empty(t, md.Get("x-grpc-upstream"), "direct mode should not set x-grpc-upstream")
}

func TestTokenManager_AttachMetadata_EmptyTokens(t *testing.T) {
	tm := yaogrpc.NewTokenManager("", "", "", "")
	ctx := tm.AttachMetadata(context.Background())

	_, ok := metadata.FromOutgoingContext(ctx)
	assert.False(t, ok, "empty tokens should not produce metadata")
}

func TestTokenManager_HandleResponseHeaders(t *testing.T) {
	tm := yaogrpc.NewTokenManager("old-tok", "old-ref", "", "")

	tm.HandleResponseHeaders(metadata.New(map[string]string{
		"x-access-token":  "new-tok",
		"x-refresh-token": "new-ref",
	}))

	assert.Equal(t, "new-tok", tm.AccessToken())

	ctx := tm.AttachMetadata(context.Background())
	md, _ := metadata.FromOutgoingContext(ctx)
	assert.Equal(t, []string{"Bearer new-tok"}, md.Get("authorization"))
	assert.Equal(t, []string{"new-ref"}, md.Get("x-refresh-token"))
}

func TestTokenManager_HandleResponseHeaders_Nil(t *testing.T) {
	tm := yaogrpc.NewTokenManager("tok", "", "", "")
	tm.HandleResponseHeaders(nil)
	assert.Equal(t, "tok", tm.AccessToken())
}

func TestTokenManager_HandleResponseHeaders_EmptyValues(t *testing.T) {
	tm := yaogrpc.NewTokenManager("tok", "ref", "", "")
	tm.HandleResponseHeaders(metadata.New(map[string]string{
		"x-access-token": "",
	}))
	assert.Equal(t, "tok", tm.AccessToken(), "empty header should not overwrite")
}

func TestTokenManager_IsTaiMode(t *testing.T) {
	tmDirect := yaogrpc.NewTokenManager("tok", "", "", "")
	assert.False(t, tmDirect.IsTaiMode())

	tmTai := yaogrpc.NewTokenManager("tok", "", "", "tai:9100")
	assert.True(t, tmTai.IsTaiMode())
}

func TestTokenManager_NewFromEnv_MissingUpstream(t *testing.T) {
	t.Setenv("YAO_GRPC_TAI", "enable")
	t.Setenv("YAO_GRPC_UPSTREAM", "")
	t.Setenv("YAO_TOKEN", "tok")

	_, err := yaogrpc.NewTokenManagerFromEnv()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "YAO_GRPC_UPSTREAM")
}

func TestTokenManager_NewFromEnv_TaiEnabled(t *testing.T) {
	t.Setenv("YAO_GRPC_TAI", "enable")
	t.Setenv("YAO_GRPC_UPSTREAM", "yao:9099")
	t.Setenv("YAO_TOKEN", "my-token")
	t.Setenv("YAO_REFRESH_TOKEN", "my-refresh")
	t.Setenv("YAO_SANDBOX_ID", "sb-42")

	tm, err := yaogrpc.NewTokenManagerFromEnv()
	require.NoError(t, err)
	assert.True(t, tm.IsTaiMode())
	assert.Equal(t, "my-token", tm.AccessToken())
}

func TestTokenManager_NewFromEnv_DirectMode(t *testing.T) {
	t.Setenv("YAO_GRPC_TAI", "")
	t.Setenv("YAO_GRPC_UPSTREAM", "")
	t.Setenv("YAO_TOKEN", "tok")

	tm, err := yaogrpc.NewTokenManagerFromEnv()
	require.NoError(t, err)
	assert.False(t, tm.IsTaiMode())
}

// ── Dial tests ───────────────────────────────────────────────────────────────

func TestNewFromEnv_MissingAddr(t *testing.T) {
	t.Setenv("YAO_GRPC_ADDR", "")
	_, err := yaogrpc.NewFromEnv()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "YAO_GRPC_ADDR")
}

func TestNewFromEnv_Success(t *testing.T) {
	t.Setenv("YAO_GRPC_ADDR", "127.0.0.1:9099")
	t.Setenv("YAO_TOKEN", "test-token")
	t.Setenv("YAO_REFRESH_TOKEN", "test-refresh")
	t.Setenv("YAO_SANDBOX_ID", "sb-1")
	t.Setenv("YAO_GRPC_TAI", "")

	c, err := yaogrpc.NewFromEnv()
	require.NoError(t, err)
	defer c.Close()

	assert.NotNil(t, c.Conn())
	assert.Equal(t, "test-token", c.TokenManager().AccessToken())
}

func TestNewFromEnv_TaiMode_MissingUpstream(t *testing.T) {
	t.Setenv("YAO_GRPC_ADDR", "tai:9100")
	t.Setenv("YAO_GRPC_TAI", "enable")
	t.Setenv("YAO_GRPC_UPSTREAM", "")

	_, err := yaogrpc.NewFromEnv()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "YAO_GRPC_UPSTREAM")
}

func TestDial_WithNilTokenManager(t *testing.T) {
	c, err := yaogrpc.Dial("127.0.0.1:0", nil)
	require.NoError(t, err)
	defer c.Close()

	assert.NotNil(t, c.Conn())
	assert.Nil(t, c.TokenManager())
}

func TestDial_WithTokenManager(t *testing.T) {
	tm := yaogrpc.NewTokenManager("tok", "", "", "")
	c, err := yaogrpc.Dial("127.0.0.1:0", tm)
	require.NoError(t, err)
	defer c.Close()

	assert.NotNil(t, c.TokenManager())
	assert.False(t, c.TokenManager().IsTaiMode())
}

func TestClient_Close_Nil(t *testing.T) {
	c := &yaogrpc.Client{}
	assert.NoError(t, c.Close())
}
