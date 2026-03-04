package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/yaoapp/yao/grpc/pb"
	"github.com/yaoapp/yao/service"
)

// Handler implements the API gRPC method (internal HTTP proxy).
type Handler struct{}

// API proxies a gRPC request to the internal openapi HTTP router.
func (h *Handler) API(ctx context.Context, req *pb.APIRequest) (*pb.APIResponse, error) {
	router := service.Router
	if router == nil {
		return nil, status.Error(codes.Unavailable, "HTTP router not initialized")
	}

	if req.Method == "" {
		return nil, status.Error(codes.InvalidArgument, "method is required")
	}
	if req.Path == "" {
		return nil, status.Error(codes.InvalidArgument, "path is required")
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.Path, bytes.NewReader(req.Body))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to build HTTP request: %v", err)
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Forward Bearer token from gRPC metadata to HTTP Authorization header
	// when the caller didn't explicitly set it.
	if httpReq.Header.Get("Authorization") == "" {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get("authorization"); len(vals) > 0 {
				httpReq.Header.Set("Authorization", vals[0])
			}
		}
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	result := w.Result()
	defer result.Body.Close()

	respHeaders := make(map[string]string, len(result.Header))
	for k := range result.Header {
		respHeaders[k] = result.Header.Get(k)
	}

	return &pb.APIResponse{
		Status:  int32(result.StatusCode),
		Headers: respHeaders,
		Body:    w.Body.Bytes(),
	}, nil
}
