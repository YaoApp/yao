package testutils

import (
	"context"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	gouapi "github.com/yaoapp/gou/api"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/query"
	"github.com/yaoapp/gou/query/gou"
	"github.com/yaoapp/xun/capsule"
	yaoagent "github.com/yaoapp/yao/agent"
	"github.com/yaoapp/yao/agent/caller"
	agentllm "github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/config"
	yaogrpc "github.com/yaoapp/yao/grpc"
	_ "github.com/yaoapp/yao/grpc/auth"
	"github.com/yaoapp/yao/grpc/pb"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/service"
	"github.com/yaoapp/yao/tai/registry"
	"github.com/yaoapp/yao/test"

	_ "github.com/yaoapp/gou/encoding"
	_ "github.com/yaoapp/gou/text"
	_ "github.com/yaoapp/yao/agent/assistant"
)

// Prepare initializes the Yao runtime (DB, V8, models, stores, scripts),
// loads the OpenAPI server (which bootstraps oauth.OAuth and acl.Global),
// sets up the HTTP router for API proxy tests,
// then starts a real gRPC server on a random port.
// Returns a connected grpc.ClientConn ready to create service clients.
func Prepare(t *testing.T) *grpc.ClientConn {
	t.Helper()

	cfg := config.Conf
	cfg.GRPC.Port = 0
	cfg.GRPC.Host = "0.0.0.0"
	cfg.GRPC.Enabled = ""

	test.Prepare(t, config.Conf)

	if openapi.Server == nil {
		if _, err := openapi.Load(config.Conf); err != nil {
			t.Fatalf("failed to load OpenAPI server: %v", err)
		}
	}

	// Load KB (required for agent KB features).
	if _, err := kb.Load(config.Conf); err != nil {
		t.Logf("warning: failed to load KB: %v", err)
	}

	// Load agent DSL (required for AgentStream handler).
	if yaoagent.GetAgent() == nil {
		if err := yaoagent.Load(config.Conf); err != nil {
			t.Logf("warning: failed to load agent DSL: %v", err)
		}
	}

	// Register JSAPI factories (idempotent, needed because Go init order is not guaranteed).
	caller.SetJSAPIFactory()
	agentllm.SetJSAPIFactory()

	// Register default query engine (required for DB search).
	if _, has := query.Engines["default"]; !has && capsule.Global != nil {
		query.Register("default", &gou.Query{
			Query: capsule.Query(),
			GetTableName: func(s string) string {
				if mod, has := model.Models[s]; has {
					return mod.MetaData.Table.Name
				}
				return s
			},
			AESKey: config.Conf.DB.AESKey,
		})
	}

	// Set up the HTTP router so grpc/api can proxy requests internally.
	if service.Router == nil {
		router := gin.New()
		if openapi.Server != nil {
			gouapi.SetRoutes(router, openapi.Server.Config.BaseURL)
			gouapi.BuildRouteTable()
			openapi.Server.Attach(router)
		}
		service.Router = router
	}

	if registry.Global() == nil {
		registry.SetGlobalForTest(registry.NewForTest())
	}

	if err := yaogrpc.StartServer(cfg); err != nil {
		t.Fatalf("failed to start gRPC server: %v", err)
	}

	addrs := yaogrpc.Addr()
	if len(addrs) == 0 {
		t.Fatal("gRPC server has no listen address")
	}

	conn, err := grpc.NewClient(addrs[0], grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to dial gRPC server: %v", err)
	}

	return conn
}

// Clean stops the gRPC server and tears down the Yao runtime.
func Clean() {
	yaogrpc.Stop()
	service.Router = nil
	openapi.Server = nil
	test.Clean()
}

// Addr returns the gRPC server listen address.
func Addr() string {
	addrs := yaogrpc.Addr()
	if len(addrs) == 0 {
		return ""
	}
	return addrs[0]
}

// RelayAddr returns the gRPC address reachable from a Docker container.
// When TAI_TEST_HOST_IP is set (e.g. to the docker bridge gateway),
// it replaces the host portion so that the Tai container can reach the
// Yao gRPC server running on the CI host.
func RelayAddr() string {
	addr := Addr()
	if addr == "" {
		return ""
	}
	hostIP := os.Getenv("TAI_TEST_HOST_IP")
	if hostIP == "" {
		return addr
	}
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return hostIP + ":" + port
}

// ObtainAccessToken mints a token with the given scopes via oauth.MakeAccessToken.
func ObtainAccessToken(t *testing.T, scopes ...string) string {
	t.Helper()
	svc := oauth.OAuth
	if svc == nil {
		t.Fatal("oauth service not initialized")
	}

	scope := strings.Join(scopes, " ")
	token, err := svc.MakeAccessToken("grpc-test", scope, "test-user", 3600)
	if err != nil {
		t.Fatalf("failed to make access token: %v", err)
	}
	return token
}

// ObtainAccessTokenForUser mints a token for a specific user ID.
func ObtainAccessTokenForUser(t *testing.T, userID string, scopes ...string) string {
	t.Helper()
	svc := oauth.OAuth
	if svc == nil {
		t.Fatal("oauth service not initialized")
	}

	scope := strings.Join(scopes, " ")
	token, err := svc.MakeAccessToken("grpc-test", scope, userID, 3600)
	if err != nil {
		t.Fatalf("failed to make access token: %v", err)
	}
	return token
}

// ObtainExpiredAccessToken mints an already-expired token (TTL=1s already elapsed).
func ObtainExpiredAccessToken(t *testing.T, scopes ...string) string {
	t.Helper()
	svc := oauth.OAuth
	if svc == nil {
		t.Fatal("oauth service not initialized")
	}

	scope := strings.Join(scopes, " ")
	token, err := svc.MakeAccessToken("grpc-test", scope, "test-user", -1)
	if err != nil {
		t.Fatalf("failed to make expired access token: %v", err)
	}
	return token
}

// ObtainRefreshToken mints a refresh token.
func ObtainRefreshToken(t *testing.T, scopes ...string) string {
	t.Helper()
	svc := oauth.OAuth
	if svc == nil {
		t.Fatal("oauth service not initialized")
	}

	scope := strings.Join(scopes, " ")
	token, err := svc.MakeRefreshToken("grpc-test", scope, "test-user", 0)
	if err != nil {
		t.Fatalf("failed to make refresh token: %v", err)
	}
	return token
}

// WithToken attaches a Bearer token to the context via gRPC metadata.
func WithToken(ctx context.Context, token string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
}

// WithRefreshToken attaches both Bearer and x-refresh-token to the context.
func WithRefreshToken(ctx context.Context, token, refreshToken string) context.Context {
	return metadata.AppendToOutgoingContext(ctx,
		"authorization", "Bearer "+token,
		"x-refresh-token", refreshToken,
	)
}

// WithSandboxMetadata attaches x-sandbox-id and x-grpc-upstream metadata.
func WithSandboxMetadata(ctx context.Context, sandboxID, upstream string) context.Context {
	return metadata.AppendToOutgoingContext(ctx,
		"x-sandbox-id", sandboxID,
		"x-grpc-upstream", upstream,
	)
}

// NewClient creates a pb.YaoClient from a connection.
func NewClient(conn *grpc.ClientConn) pb.YaoClient {
	return pb.NewYaoClient(conn)
}
