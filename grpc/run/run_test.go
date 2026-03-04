package run_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/yaoapp/yao/grpc/pb"
	"github.com/yaoapp/yao/grpc/tests/testutils"
)

func TestRun_ProcessExec(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:run")
	ctx := testutils.WithToken(context.Background(), token)

	resp, err := client.Run(ctx, &pb.RunRequest{
		Process: "utils.app.Ping",
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Data)
}

func TestRun_WithArgs(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:run")
	ctx := testutils.WithToken(context.Background(), token)

	args, _ := json.Marshal([]interface{}{"hello", " world"})
	resp, err := client.Run(ctx, &pb.RunRequest{
		Process: "utils.str.Concat",
		Args:    args,
	})
	assert.NoError(t, err)
	if assert.NotNil(t, resp) {
		assert.NotEmpty(t, resp.Data)

		var result string
		err = json.Unmarshal(resp.Data, &result)
		assert.NoError(t, err)
		assert.Equal(t, "hello world", result)
	}
}

func TestRun_InvalidProcess(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:run")
	ctx := testutils.WithToken(context.Background(), token)

	_, err := client.Run(ctx, &pb.RunRequest{Process: "nonexistent.process.here"})
	assert.Error(t, err)
}

func TestRun_EmptyProcessName(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:run")
	ctx := testutils.WithToken(context.Background(), token)

	_, err := client.Run(ctx, &pb.RunRequest{Process: ""})
	assert.Error(t, err)
}

func TestRun_BadArgsJSON(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:run")
	ctx := testutils.WithToken(context.Background(), token)

	_, err := client.Run(ctx, &pb.RunRequest{
		Process: "utils.app.Ping",
		Args:    []byte("{not-json"),
	})
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestRun_WithTimeout(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:run")
	ctx := testutils.WithToken(context.Background(), token)

	resp, err := client.Run(ctx, &pb.RunRequest{
		Process: "utils.app.Ping",
		Timeout: 30,
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestRun_EmptyProcessName_StatusCode(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:run")
	ctx := testutils.WithToken(context.Background(), token)

	_, err := client.Run(ctx, &pb.RunRequest{Process: ""})
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestRun_InvalidProcess_StatusCode(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:run")
	ctx := testutils.WithToken(context.Background(), token)

	_, err := client.Run(ctx, &pb.RunRequest{Process: "nonexistent.process.here"})
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestRun_NilArgs(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:run")
	ctx := testutils.WithToken(context.Background(), token)

	resp, err := client.Run(ctx, &pb.RunRequest{
		Process: "utils.app.Ping",
		Args:    nil,
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}
