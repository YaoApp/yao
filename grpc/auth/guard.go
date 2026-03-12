package auth

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/acl"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

const (
	healthzMethod     = "/yao.Yao/Healthz"
	apiMethod         = "/yao.Yao/API"
	taiRegisterMethod = "/tai.tunnel.TaiTunnel/Register"
	taiForwardMethod  = "/tai.tunnel.TaiTunnel/Forward"

	metaAuthorization = "authorization"
	metaRefreshToken  = "x-refresh-token"
	metaAccessToken   = "x-access-token"
	metaSandboxID     = "x-sandbox-id"
	metaSessionID     = "x-session-id"
)

type authCtxKey struct{}

// WithAuthorizedInfo stores AuthorizedInfo in context for downstream handlers.
func WithAuthorizedInfo(ctx context.Context, info *types.AuthorizedInfo) context.Context {
	return context.WithValue(ctx, authCtxKey{}, info)
}

// GetAuthorizedInfo retrieves AuthorizedInfo from context (set by the interceptor).
func GetAuthorizedInfo(ctx context.Context) *types.AuthorizedInfo {
	info, _ := ctx.Value(authCtxKey{}).(*types.AuthorizedInfo)
	return info
}

// UnaryInterceptor is the gRPC unary server interceptor for authentication and authorization.
func UnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if info.FullMethod == healthzMethod {
		return handler(ctx, req)
	}

	ctx, err := authenticate(ctx, info.FullMethod, req)
	if err != nil {
		return nil, err
	}
	return handler(ctx, req)
}

// StreamInterceptor is the gRPC stream server interceptor for authentication and authorization.
// For streaming RPCs, the request object is not available at intercept time,
// so ACL scope check uses the method-level virtual path (without request-specific IDs).
func StreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if info.FullMethod == healthzMethod {
		return handler(srv, ss)
	}

	ctx, err := authenticate(ss.Context(), info.FullMethod, nil)
	if err != nil {
		return err
	}

	return handler(srv, &wrappedStream{ServerStream: ss, ctx: ctx})
}

// authenticate calls oauth.Service.AuthenticateToken directly — no gin/HTTP shim.
func authenticate(ctx context.Context, fullMethod string, req interface{}) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx, status.Error(codes.Unauthenticated, "missing metadata")
	}

	svc := oauth.OAuth
	if svc == nil {
		return ctx, status.Error(codes.Internal, "oauth service not initialized")
	}

	bearer := extractBearer(md)
	if bearer == "" {
		return ctx, status.Error(codes.Unauthenticated, "missing authorization token")
	}

	result, err := svc.AuthenticateToken(oauth.AuthInput{
		AccessToken:  bearer,
		RefreshToken: extractMeta(md, metaRefreshToken),
		SessionID:    extractMeta(md, metaSessionID),
	})
	if err != nil {
		return ctx, status.Error(codes.Unauthenticated, err.Error())
	}

	ctx = WithAuthorizedInfo(ctx, result.Info)

	if result.NewAccessToken != "" {
		_ = grpc.SendHeader(ctx, metadata.Pairs(
			metaAccessToken, result.NewAccessToken,
			metaRefreshToken, result.NewRefreshToken,
		))
	}

	// ACL scope check — skip for API proxy and Tai tunnel (infrastructure services).
	if fullMethod != apiMethod && fullMethod != taiRegisterMethod && fullMethod != taiForwardMethod {
		httpMethod, httpPath := VirtualEndpoint(fullMethod, req)
		scopes := strings.Fields(result.Info.Scope)

		enforcer := getACLEnforcer()
		if enforcer != nil && enforcer.Scope != nil {
			decision := enforcer.Scope.Check(&acl.AccessRequest{
				Method: httpMethod,
				Path:   httpPath,
				Scopes: scopes,
			})
			if !decision.Allowed {
				return ctx, status.Errorf(codes.PermissionDenied, "insufficient scope: %s", decision.Reason)
			}
		}
	}

	return ctx, nil
}

// getACLEnforcer returns the ACL enforcer if available and enabled.
func getACLEnforcer() *acl.ACL {
	if acl.Global == nil {
		return nil
	}
	enforcer, ok := acl.Global.(*acl.ACL)
	if !ok || enforcer == nil {
		return nil
	}
	if !enforcer.Config.Enabled {
		return nil
	}
	return enforcer
}

func extractBearer(md metadata.MD) string {
	vals := md.Get(metaAuthorization)
	if len(vals) == 0 {
		return ""
	}
	parts := strings.SplitN(vals[0], " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
		return parts[1]
	}
	return vals[0]
}

func extractMeta(md metadata.MD, key string) string {
	vals := md.Get(key)
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}

// wrappedStream wraps grpc.ServerStream with a custom context.
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}
