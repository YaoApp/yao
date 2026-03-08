package client

import (
	"context"
	"os"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// TokenManager attaches auth credentials as gRPC metadata on every call
// and handles automatic token refresh from response headers.
type TokenManager struct {
	mu           sync.RWMutex
	accessToken  string
	refreshToken string
	sandboxID    string
}

// NewTokenManagerFromEnv creates a TokenManager from environment variables.
func NewTokenManagerFromEnv() (*TokenManager, error) {
	return &TokenManager{
		accessToken:  os.Getenv("YAO_TOKEN"),
		refreshToken: os.Getenv("YAO_REFRESH_TOKEN"),
		sandboxID:    os.Getenv("YAO_SANDBOX_ID"),
	}, nil
}

// NewTokenManager creates a TokenManager with explicit values.
func NewTokenManager(accessToken, refreshToken, sandboxID string) *TokenManager {
	return &TokenManager{
		accessToken:  accessToken,
		refreshToken: refreshToken,
		sandboxID:    sandboxID,
	}
}

// AttachMetadata returns a context with auth credentials in gRPC metadata.
func (tm *TokenManager) AttachMetadata(ctx context.Context) context.Context {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var pairs []string
	if tm.accessToken != "" {
		pairs = append(pairs, "authorization", "Bearer "+tm.accessToken)
	}
	if tm.refreshToken != "" {
		pairs = append(pairs, "x-refresh-token", tm.refreshToken)
	}
	if tm.sandboxID != "" {
		pairs = append(pairs, "x-sandbox-id", tm.sandboxID)
	}

	if len(pairs) == 0 {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, pairs...)
}

// HandleResponseHeaders reads new tokens from response headers and updates
// the in-memory credentials.
func (tm *TokenManager) HandleResponseHeaders(header metadata.MD) {
	if header == nil {
		return
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	if vals := header.Get("x-access-token"); len(vals) > 0 && vals[0] != "" {
		tm.accessToken = vals[0]
	}
	if vals := header.Get("x-refresh-token"); len(vals) > 0 && vals[0] != "" {
		tm.refreshToken = vals[0]
	}
}

// UnaryInterceptor returns a gRPC unary client interceptor that attaches
// auth metadata and handles token refresh from response headers.
func (tm *TokenManager) UnaryInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any,
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		ctx = tm.AttachMetadata(ctx)
		var header metadata.MD
		opts = append(opts, grpc.Header(&header))
		err := invoker(ctx, method, req, reply, cc, opts...)
		tm.HandleResponseHeaders(header)
		return err
	}
}

// StreamInterceptor returns a gRPC stream client interceptor that attaches
// auth metadata.
func (tm *TokenManager) StreamInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn,
		method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {

		ctx = tm.AttachMetadata(ctx)
		stream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			return nil, err
		}
		if header, hErr := stream.Header(); hErr == nil {
			tm.HandleResponseHeaders(header)
		}
		return stream, nil
	}
}

// AccessToken returns the current access token.
func (tm *TokenManager) AccessToken() string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.accessToken
}
