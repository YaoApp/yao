package shell

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/yaoapp/yao/grpc/pb"
)

const (
	defaultTimeout = 30 * time.Second
	maxTimeout     = 300 * time.Second
)

// Handler implements the Shell gRPC method.
type Handler struct{}

// Shell executes a system command in the host process and returns stdout/stderr/exit code.
func (h *Handler) Shell(ctx context.Context, req *pb.ShellRequest) (*pb.ShellResponse, error) {
	if os.Getuid() == 0 {
		return nil, status.Error(codes.PermissionDenied, "shell execution refused when running as root")
	}

	if req.Command == "" {
		return nil, status.Error(codes.InvalidArgument, "command is required")
	}

	timeout := defaultTimeout
	if req.Timeout > 0 {
		timeout = time.Duration(req.Timeout) * time.Second
		if timeout > maxTimeout {
			timeout = maxTimeout
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, req.Command, req.Args...)

	if len(req.Env) > 0 {
		env := os.Environ()
		for k, v := range req.Env {
			env = append(env, k+"="+v)
		}
		cmd.Env = env
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	resp := &pb.ShellResponse{
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
		ExitCode: 0,
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, status.Error(codes.DeadlineExceeded, "command timed out")
		}

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				resp.ExitCode = int32(ws.ExitStatus())
			} else {
				resp.ExitCode = int32(exitErr.ExitCode())
			}
			return resp, nil
		}

		if errors.Is(err, exec.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "command not found: %s", req.Command)
		}
		return nil, status.Errorf(codes.Internal, "command execution failed: %v", err)
	}

	return resp, nil
}
