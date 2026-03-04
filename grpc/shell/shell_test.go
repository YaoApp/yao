package shell_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/yaoapp/yao/grpc/pb"
	"github.com/yaoapp/yao/grpc/tests/testutils"
)

func TestShell_Echo(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:shell")
	ctx := testutils.WithToken(context.Background(), token)

	resp, err := client.Shell(ctx, &pb.ShellRequest{
		Command: "echo",
		Args:    []string{"hello"},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Contains(t, string(resp.Stdout), "hello")
	assert.Equal(t, int32(0), resp.ExitCode)
}

func TestShell_CommandNotFound(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:shell")
	ctx := testutils.WithToken(context.Background(), token)

	_, err := client.Shell(ctx, &pb.ShellRequest{
		Command: "this_command_does_not_exist_xyz",
	})
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestShell_Timeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sleep command not available on Windows")
	}

	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:shell")
	ctx := testutils.WithToken(context.Background(), token)

	_, err := client.Shell(ctx, &pb.ShellRequest{
		Command: "sleep",
		Args:    []string{"10"},
		Timeout: 1,
	})
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.DeadlineExceeded, st.Code())
}

func TestShell_EmptyCommand(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:shell")
	ctx := testutils.WithToken(context.Background(), token)

	_, err := client.Shell(ctx, &pb.ShellRequest{Command: ""})
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestShell_NonZeroExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("false command not available on Windows")
	}

	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:shell")
	ctx := testutils.WithToken(context.Background(), token)

	resp, err := client.Shell(ctx, &pb.ShellRequest{
		Command: "false",
	})
	assert.NoError(t, err)
	assert.NotEqual(t, int32(0), resp.ExitCode)
}

func TestShell_WithEnv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("printenv not available on Windows")
	}

	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:shell")
	ctx := testutils.WithToken(context.Background(), token)

	resp, err := client.Shell(ctx, &pb.ShellRequest{
		Command: "printenv",
		Args:    []string{"TEST_GRPC_VAR"},
		Env:     map[string]string{"TEST_GRPC_VAR": "grpc_value"},
	})
	assert.NoError(t, err)
	assert.Contains(t, string(resp.Stdout), "grpc_value")
}

func TestShell_MaxTimeoutCapped(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("echo not available on Windows")
	}

	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:shell")
	ctx := testutils.WithToken(context.Background(), token)

	resp, err := client.Shell(ctx, &pb.ShellRequest{
		Command: "echo",
		Args:    []string{"ok"},
		Timeout: 9999,
	})
	assert.NoError(t, err)
	assert.Contains(t, string(resp.Stdout), "ok")
}

func TestShell_Stderr(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash not available on Windows")
	}

	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:shell")
	ctx := testutils.WithToken(context.Background(), token)

	resp, err := client.Shell(ctx, &pb.ShellRequest{
		Command: "bash",
		Args:    []string{"-c", "echo error_msg >&2"},
	})
	assert.NoError(t, err)
	assert.Contains(t, string(resp.Stderr), "error_msg")
	assert.Equal(t, int32(0), resp.ExitCode)
}
