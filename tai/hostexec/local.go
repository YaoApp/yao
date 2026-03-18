package hostexec

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	pb "github.com/yaoapp/yao/tai/hostexec/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const defaultMaxOutputBytes = 10 * 1024 * 1024 // 10 MB

// Policy controls which commands and directories are allowed.
type Policy struct {
	FullAccess      bool     // bypass command and path checks
	AllowedCommands []string // empty = all denied (unless FullAccess)
	AllowedDirs     []string // working_dir must be under one of these
	DeniedDirs      []string // higher priority than AllowedDirs
}

// ---------------------------------------------------------------------------
// LocalClient — in-process HostExecClient (no gRPC network hop)
// ---------------------------------------------------------------------------

// LocalClient implements pb.HostExecClient by executing commands directly on
// the current host via os/exec.
type LocalClient struct {
	defaultDir string
	policy     Policy
}

// Compile-time interface check.
var _ pb.HostExecClient = (*LocalClient)(nil)

// NewLocalClient creates a LocalClient.
func NewLocalClient(defaultDir string, policy Policy) *LocalClient {
	return &LocalClient{defaultDir: defaultDir, policy: policy}
}

// Exec runs a command synchronously and returns the result.
func (c *LocalClient) Exec(ctx context.Context, req *pb.ExecRequest, _ ...grpc.CallOption) (*pb.ExecResponse, error) {
	if err := c.checkCommand(req.Command); err != nil {
		return &pb.ExecResponse{Error: err.Error()}, nil
	}
	if err := c.checkWorkingDir(req.WorkingDir); err != nil {
		return &pb.ExecResponse{Error: err.Error()}, nil
	}

	timeout := time.Duration(req.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, req.Command, req.Args...)
	cmd.Dir = c.resolveDir(req.WorkingDir)
	cmd.Env = c.buildEnv(req.Env)
	if len(req.Stdin) > 0 {
		cmd.Stdin = bytes.NewReader(req.Stdin)
	}

	maxBytes := req.MaxOutputBytes
	if maxBytes <= 0 {
		maxBytes = defaultMaxOutputBytes
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &limitWriter{buf: &stdout, max: maxBytes}
	cmd.Stderr = &limitWriter{buf: &stderr, max: maxBytes}

	start := time.Now()
	err := cmd.Run()

	resp := &pb.ExecResponse{
		Stdout:     stdout.Bytes(),
		Stderr:     stderr.Bytes(),
		DurationMs: time.Since(start).Milliseconds(),
	}
	if int64(len(resp.Stdout)+len(resp.Stderr)) >= maxBytes {
		resp.Truncated = true
	}

	if err != nil {
		if ctx.Err() != nil {
			resp.Error = "command timed out"
			resp.ExitCode = -1
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			resp.ExitCode = int32(exitErr.ExitCode())
		} else {
			resp.Error = err.Error()
			resp.ExitCode = -1
		}
	}
	return resp, nil
}

// ExecStream runs a command and streams stdout/stderr via a channel-based
// adapter that satisfies grpc.ServerStreamingClient[pb.ExecOutput].
func (c *LocalClient) ExecStream(ctx context.Context, req *pb.ExecRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[pb.ExecOutput], error) {
	if err := c.checkCommand(req.Command); err != nil {
		return newErrorStream(ctx, err.Error()), nil
	}
	if err := c.checkWorkingDir(req.WorkingDir); err != nil {
		return newErrorStream(ctx, err.Error()), nil
	}

	timeout := time.Duration(req.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)

	cmd := exec.CommandContext(ctx, req.Command, req.Args...)
	cmd.Dir = c.resolveDir(req.WorkingDir)
	cmd.Env = c.buildEnv(req.Env)
	if len(req.Stdin) > 0 {
		cmd.Stdin = bytes.NewReader(req.Stdin)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return newErrorStream(ctx, err.Error()), nil
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return newErrorStream(ctx, err.Error()), nil
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return newErrorStream(ctx, err.Error()), nil
	}

	ch := make(chan *pb.ExecOutput, 64)
	go func() {
		defer cancel()
		defer close(ch)

		done := make(chan struct{})
		go func() {
			defer close(done)
			streamPipe(ch, stdoutPipe, pb.ExecOutput_STDOUT)
		}()
		streamPipe(ch, stderrPipe, pb.ExecOutput_STDERR)
		<-done

		waitErr := cmd.Wait()
		final := &pb.ExecOutput{Done: true}
		if waitErr != nil {
			if exitErr, ok := waitErr.(*exec.ExitError); ok {
				final.ExitCode = int32(exitErr.ExitCode())
			} else {
				final.Error = waitErr.Error()
				final.ExitCode = -1
			}
		}
		ch <- final
	}()

	return &localStream{ctx: ctx, ch: ch}, nil
}

// ---------------------------------------------------------------------------
// Policy checks (identical to Tai hostexec/server.go)
// ---------------------------------------------------------------------------

func (c *LocalClient) checkCommand(command string) error {
	if c.policy.FullAccess {
		return nil
	}
	if len(c.policy.AllowedCommands) == 0 {
		return fmt.Errorf("hostexec: no commands are allowed (allowed_commands is empty)")
	}
	base := filepath.Base(command)
	for _, allowed := range c.policy.AllowedCommands {
		if command == allowed || base == allowed {
			return nil
		}
	}
	return fmt.Errorf("hostexec: command %q is not in the allowed list", command)
}

func (c *LocalClient) checkWorkingDir(dir string) error {
	if dir == "" || c.policy.FullAccess {
		return nil
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("hostexec: invalid working_dir %q: %w", dir, err)
	}
	resolved, err := filepath.EvalSymlinks(absDir)
	if err != nil {
		resolved = absDir
	}
	for _, denied := range c.policy.DeniedDirs {
		if matchDir(resolved, denied) {
			return fmt.Errorf("hostexec: working_dir %q is in a denied directory", dir)
		}
	}
	if len(c.policy.AllowedDirs) == 0 {
		return nil
	}
	for _, allowed := range c.policy.AllowedDirs {
		if matchDir(resolved, allowed) {
			return nil
		}
	}
	return fmt.Errorf("hostexec: working_dir %q is not in any allowed directory", dir)
}

func matchDir(resolved, dir string) bool {
	absDir, _ := filepath.Abs(dir)
	resolvedDir, err := filepath.EvalSymlinks(absDir)
	if err != nil {
		resolvedDir = absDir
	}
	if resolved == resolvedDir {
		return true
	}
	return strings.HasPrefix(resolved, resolvedDir+string(filepath.Separator))
}

func (c *LocalClient) resolveDir(dir string) string {
	if dir != "" {
		return dir
	}
	if c.defaultDir != "" {
		return c.defaultDir
	}
	return ""
}

func (c *LocalClient) buildEnv(userEnv map[string]string) []string {
	env := os.Environ()
	for k, v := range userEnv {
		env = append(env, k+"="+v)
	}
	return env
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func streamPipe(ch chan<- *pb.ExecOutput, pipe io.ReadCloser, st pb.ExecOutput_Stream) {
	buf := make([]byte, 32*1024)
	for {
		n, err := pipe.Read(buf)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])
			ch <- &pb.ExecOutput{Stream: st, Data: data}
		}
		if err != nil {
			return
		}
	}
}

type limitWriter struct {
	buf *bytes.Buffer
	max int64
}

func (w *limitWriter) Write(p []byte) (int, error) {
	remaining := w.max - int64(w.buf.Len())
	if remaining <= 0 {
		return len(p), nil
	}
	if int64(len(p)) > remaining {
		p = p[:remaining]
	}
	return w.buf.Write(p)
}

// ---------------------------------------------------------------------------
// localStream — channel-based grpc.ServerStreamingClient adapter
// ---------------------------------------------------------------------------

type localStream struct {
	ctx context.Context
	ch  <-chan *pb.ExecOutput
}

var _ grpc.ServerStreamingClient[pb.ExecOutput] = (*localStream)(nil)

func (s *localStream) Recv() (*pb.ExecOutput, error) {
	select {
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	case msg, ok := <-s.ch:
		if !ok {
			return nil, io.EOF
		}
		return msg, nil
	}
}

func (s *localStream) Header() (metadata.MD, error) { return nil, nil }
func (s *localStream) Trailer() metadata.MD         { return nil }
func (s *localStream) CloseSend() error             { return nil }
func (s *localStream) Context() context.Context     { return s.ctx }
func (s *localStream) SendMsg(any) error            { return nil }
func (s *localStream) RecvMsg(any) error            { return nil }

// newErrorStream returns a stream that yields a single Done message with the
// given error, then EOF. Used for early policy-check failures.
func newErrorStream(ctx context.Context, errMsg string) grpc.ServerStreamingClient[pb.ExecOutput] {
	ch := make(chan *pb.ExecOutput, 1)
	ch <- &pb.ExecOutput{Done: true, Error: errMsg, ExitCode: -1}
	close(ch)
	return &localStream{ctx: ctx, ch: ch}
}
