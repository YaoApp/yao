package auth_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/yaoapp/yao/grpc/pb"
	"github.com/yaoapp/yao/grpc/tests/testutils"
	"github.com/yaoapp/yao/tai/tunnel/taipb"
)

func TestAuth_NoToken_Rejected(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	_, err := client.Run(context.Background(), &pb.RunRequest{Process: "utils.app.Ping"})
	assert.Error(t, err)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAuth_ValidToken(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:run")
	ctx := testutils.WithToken(context.Background(), token)

	_, err := client.Run(ctx, &pb.RunRequest{Process: "utils.app.Ping"})
	// Run returns Unimplemented (handler stub), not an auth error
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.NotEqual(t, codes.Unauthenticated, st.Code())
	assert.NotEqual(t, codes.PermissionDenied, st.Code())
}

func TestAuth_WrongScope_Denied(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:mcp")
	ctx := testutils.WithToken(context.Background(), token)

	_, err := client.Run(ctx, &pb.RunRequest{Process: "utils.app.Ping"})
	assert.Error(t, err)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, st.Code())
}

func TestAuth_TokenRefresh(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)

	expiredToken := testutils.ObtainExpiredAccessToken(t, "grpc:run")
	refreshToken := testutils.ObtainRefreshToken(t, "grpc:run")
	ctx := testutils.WithRefreshToken(context.Background(), expiredToken, refreshToken)

	// The call should succeed (auth interceptor refreshes the token)
	_, err := client.Run(ctx, &pb.RunRequest{Process: "utils.app.Ping"})
	st, ok := status.FromError(err)
	assert.True(t, ok)
	// Should not be an auth error — either Unimplemented (handler stub) or OK
	assert.NotEqual(t, codes.Unauthenticated, st.Code())
	assert.NotEqual(t, codes.PermissionDenied, st.Code())
}

func TestHealthz_Public(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	resp, err := client.Healthz(context.Background(), &pb.Empty{})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "ok", resp.Status)
}

func TestAuth_InvalidBearerFormat(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "not-a-valid-token-at-all")

	_, err := client.Run(ctx, &pb.RunRequest{Process: "utils.app.Ping"})
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAuth_StreamInterceptor_NoToken(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	stream, err := client.ChatCompletionsStream(context.Background(), &pb.ChatRequest{
		Connector: "openai",
	})
	if err != nil {
		st, _ := status.FromError(err)
		assert.Equal(t, codes.Unauthenticated, st.Code())
		return
	}
	_, err = stream.Recv()
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAuth_StreamInterceptor_ValidToken(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:agent")
	ctx := testutils.WithToken(context.Background(), token)

	stream, err := client.AgentStream(ctx, &pb.AgentRequest{
		AssistantId: "nonexistent",
		Messages:    []byte(`[{"role":"user","content":"hi"}]`),
	})
	if err != nil {
		st, _ := status.FromError(err)
		assert.NotEqual(t, codes.Unauthenticated, st.Code())
		assert.NotEqual(t, codes.PermissionDenied, st.Code())
		return
	}
	_, err = stream.Recv()
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.NotEqual(t, codes.Unauthenticated, st.Code())
	assert.NotEqual(t, codes.PermissionDenied, st.Code())
}

func TestAuth_StreamInterceptor_WrongScope(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:run")
	ctx := testutils.WithToken(context.Background(), token)

	stream, err := client.AgentStream(ctx, &pb.AgentRequest{
		AssistantId: "test",
		Messages:    []byte(`[{"role":"user","content":"hi"}]`),
	})
	if err != nil {
		st, _ := status.FromError(err)
		assert.Equal(t, codes.PermissionDenied, st.Code())
		return
	}
	_, err = stream.Recv()
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.PermissionDenied, st.Code())
}

// ── TaiTunnel auth tests ──────────────────────────────────────────────────

func TestAuth_TaiTunnel_Register_NoToken(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := taipb.NewTaiTunnelClient(conn)
	stream, err := client.Register(context.Background())
	if err != nil {
		st, _ := status.FromError(err)
		assert.Equal(t, codes.Unauthenticated, st.Code())
		return
	}
	_ = stream.Send(&taipb.TunnelControl{Type: "register", NodeId: "n", MachineId: "m"})
	_, err = stream.Recv()
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAuth_TaiTunnel_Forward_NoToken(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := taipb.NewTaiTunnelClient(conn)
	ctx := metadata.AppendToOutgoingContext(context.Background(), "channel_id", "test-ch")
	stream, err := client.Forward(ctx)
	if err != nil {
		st, _ := status.FromError(err)
		assert.Equal(t, codes.Unauthenticated, st.Code())
		return
	}
	_ = stream.Send(&taipb.ForwardData{Data: []byte("x")})
	_, err = stream.Recv()
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAuth_TaiTunnel_Register_ValidToken(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := taipb.NewTaiTunnelClient(conn)
	token := testutils.ObtainAccessToken(t, "tai:connect")
	ctx := testutils.WithToken(context.Background(), token)

	stream, err := client.Register(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = stream.Send(&taipb.TunnelControl{
		Type: "register", NodeId: "auth-test-node", MachineId: "auth-test-machine",
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := stream.Recv()
	if err != nil {
		st, ok := status.FromError(err)
		if ok && (st.Code() == codes.Unauthenticated || st.Code() == codes.PermissionDenied) {
			t.Fatalf("expected auth to pass, got %v: %v", st.Code(), st.Message())
		}
		t.Fatal(err)
	}
	assert.Equal(t, "registered", resp.Type)
	assert.NotEmpty(t, resp.TaiId)
	stream.CloseSend()
}

func TestAuth_TaiTunnel_Register_ExpiredToken(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := taipb.NewTaiTunnelClient(conn)
	token := testutils.ObtainExpiredAccessToken(t, "tai:connect")
	ctx := testutils.WithToken(context.Background(), token)

	stream, err := client.Register(ctx)
	if err != nil {
		st, _ := status.FromError(err)
		assert.Equal(t, codes.Unauthenticated, st.Code())
		return
	}
	_ = stream.Send(&taipb.TunnelControl{Type: "register", NodeId: "n", MachineId: "m"})
	_, err = stream.Recv()
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}
