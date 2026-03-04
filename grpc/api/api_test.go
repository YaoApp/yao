package api_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/yaoapp/yao/grpc/pb"
	"github.com/yaoapp/yao/grpc/tests/testutils"
)

func TestAPI_Proxy(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)

	// The API method's ACL check uses the actual openapi path, so we grant all gRPC scopes.
	// The openapi guard inside the HTTP router handles further auth via the forwarded Authorization header.
	token := testutils.ObtainAccessToken(t,
		"grpc:run", "grpc:stream", "grpc:shell", "grpc:mcp", "grpc:llm", "grpc:agent",
	)
	ctx := testutils.WithToken(context.Background(), token)

	resp, err := client.API(ctx, &pb.APIRequest{
		Method: "GET",
		Path:   "/api/__yao/app/setting",
	})

	// The proxy itself should succeed (no gRPC error), even if the HTTP response
	// is a non-200 status (e.g. 401 from openapi's own guard).
	assert.NoError(t, err)
	if assert.NotNil(t, resp) {
		assert.Greater(t, resp.Status, int32(0))
		assert.NotNil(t, resp.Body)
	}
}

func TestAPI_NotFoundEndpoint(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t,
		"grpc:run", "grpc:stream", "grpc:shell", "grpc:mcp", "grpc:llm", "grpc:agent",
	)
	ctx := testutils.WithToken(context.Background(), token)

	resp, err := client.API(ctx, &pb.APIRequest{
		Method: "GET",
		Path:   "/api/this/does/not/exist",
	})
	assert.NoError(t, err)
	if assert.NotNil(t, resp) {
		assert.Equal(t, int32(404), resp.Status)
	}
}

func TestAPI_MissingMethod(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:run")
	ctx := testutils.WithToken(context.Background(), token)

	_, err := client.API(ctx, &pb.APIRequest{
		Method: "",
		Path:   "/api/test",
	})
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestAPI_MissingPath(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:run")
	ctx := testutils.WithToken(context.Background(), token)

	_, err := client.API(ctx, &pb.APIRequest{
		Method: "GET",
		Path:   "",
	})
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestAPI_WithHeaders(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t,
		"grpc:run", "grpc:stream", "grpc:shell", "grpc:mcp", "grpc:llm", "grpc:agent",
	)
	ctx := testutils.WithToken(context.Background(), token)

	resp, err := client.API(ctx, &pb.APIRequest{
		Method:  "GET",
		Path:    "/api/__yao/app/setting",
		Headers: map[string]string{"X-Custom-Header": "test-value"},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Greater(t, resp.Status, int32(0))
}

func TestAPI_PostWithBody(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t,
		"grpc:run", "grpc:stream", "grpc:shell", "grpc:mcp", "grpc:llm", "grpc:agent",
	)
	ctx := testutils.WithToken(context.Background(), token)

	resp, err := client.API(ctx, &pb.APIRequest{
		Method: "POST",
		Path:   "/api/this/does/not/exist",
		Body:   []byte(`{"key":"value"}`),
	})
	assert.NoError(t, err)
	if assert.NotNil(t, resp) {
		assert.Equal(t, int32(404), resp.Status)
	}
}
