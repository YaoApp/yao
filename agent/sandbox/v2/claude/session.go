package claude

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	infra "github.com/yaoapp/yao/sandbox/v2"
)

// session encapsulates a single Claude CLI execution lifecycle:
// process start, stderr collection, kill on cancel, Wait with timeout.
type session struct {
	ctx      context.Context
	computer infra.Computer
	plat     platform
	exec     *infra.ExecStream
	stderr   strings.Builder
	stderrMu sync.Mutex
	logger   *agentContext.RequestLogger
}

func startSession(ctx context.Context, computer infra.Computer, p platform, cmd command, logger *agentContext.RequestLogger) (*session, error) {
	opts := []infra.ExecOption{infra.WithWorkDir(cmd.workDir), infra.WithEnv(cmd.env)}
	if len(cmd.stdin) > 0 {
		opts = append(opts, infra.WithStdin(cmd.stdin))
	}

	logger.Info("claude session starting: cmd=%s workDir=%s platform=%s stdinLen=%d",
		cmd.shell, cmd.workDir, p.OS(), len(cmd.stdin))

	execStream, err := computer.Stream(ctx, cmd.shell, opts...)
	if err != nil {
		return nil, fmt.Errorf("computer.Stream: %w", err)
	}

	return &session{
		ctx:      ctx,
		computer: computer,
		plat:     p,
		exec:     execStream,
		logger:   logger,
	}, nil
}

// runStream executes the main stream processing loop.
// Returns (completed, error) where completed=true means Claude CLI sent
// a "result" message and the stream finished normally.
func (s *session) runStream(handler message.StreamFunc) (completed bool, err error) {
	s.collectStderr()

	cleanup := s.watchCancel()
	defer cleanup()

	parser := newStreamParser(handler)
	parseErr := parser.parse(s.ctx, s.exec.Stdout)

	s.logger.Debug("runStream: parse returned completed=%v parseErr=%v", parser.completed, parseErr)

	if parser.completed {
		s.logger.Info("claude stream completed normally")
		return true, nil
	}

	exitErr := s.waitForExit(parseErr)
	if exitErr != nil {
		if handler != nil {
			handler(message.ChunkError, []byte(exitErr.Error()))
		}
		return false, exitErr
	}

	s.stderrMu.Lock()
	stderrStr := strings.TrimSpace(s.stderr.String())
	s.stderrMu.Unlock()
	if stderrStr != "" {
		s.logger.Warn("claude exited with code 0 but stream incomplete and stderr present: %s", stderrStr)
		errMsg := fmt.Errorf("claude CLI setup failed: %s", stderrStr)
		if handler != nil {
			handler(message.ChunkError, []byte(errMsg.Error()))
		}
		return false, errMsg
	}

	return false, nil
}

// collectStderr reads stderr in a background goroutine.
// Unlike the old code, this NEVER triggers stream cancellation.
// stderr is purely informational — logged and collected for error reporting.
func (s *session) collectStderr() {
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := s.exec.Stderr.Read(buf)
			if n > 0 {
				chunk := string(buf[:n])
				s.stderrMu.Lock()
				s.stderr.WriteString(chunk)
				s.stderrMu.Unlock()
				s.logger.Debug("claude stderr: %s", chunk)
			}
			if err != nil {
				return
			}
		}
	}()
}

// watchCancel monitors context cancellation and kills the Claude process.
// Returns a cleanup function that must be deferred.
func (s *session) watchCancel() func() {
	done := make(chan struct{})
	go func() {
		select {
		case <-s.ctx.Done():
			s.logger.Info("context cancelled, killing claude: %v", s.ctx.Err())
			killCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			s.computer.Exec(killCtx, s.plat.KillCmd("claude"))
			s.exec.Cancel()
		case <-done:
		}
	}()
	return func() { close(done) }
}

// shutdown terminates the claude process after a normal stream completion.
//
// Claude CLI's stream-json mode has a known bug where the process hangs
// indefinitely after emitting the "result" event (anthropics/claude-code#25629).
// There is no graceful exit mechanism, so we must kill the process externally.
//
// We send SIGKILL (-9) to processes named exactly "claude". SIGKILL cannot be
// caught, so Claude CLI has no opportunity to run its SIGTERM handler which
// would actively terminate child processes (web servers, etc.). Those children
// survive because they run in separate process groups/sessions.
func (s *session) shutdown() {
	s.logger.Info("shutting down completed claude exec session")
	killCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := s.computer.Exec(killCtx, []string{"sh", "-c", "pkill -9 -x claude || true"})
	s.logger.Debug("shutdown: pkill -9 -x claude exitCode=%d err=%v", result.ExitCode, err)
	s.exec.Cancel()
}

// waitForExit waits for the Claude process to exit with timeout protection.
// This fixes the old code's issue where Wait() could block forever.
func (s *session) waitForExit(parseErr error) error {
	s.logger.Info("claude stream did not complete normally, waiting for exit")

	type waitResult struct {
		exitCode int
		err      error
	}
	ch := make(chan waitResult, 1)
	go func() {
		code, err := s.exec.Wait()
		ch <- waitResult{code, err}
	}()

	var exitCode int
	var waitErr error

	select {
	case wr := <-ch:
		exitCode, waitErr = wr.exitCode, wr.err
	case <-s.ctx.Done():
		select {
		case wr := <-ch:
			exitCode, waitErr = wr.exitCode, wr.err
		case <-time.After(10 * time.Second):
			s.exec.Cancel()
			s.logger.Error("claude did not exit after kill, timeout")
			return fmt.Errorf("claude did not exit after kill (timeout)")
		}
	}

	s.stderrMu.Lock()
	stderrStr := strings.TrimSpace(s.stderr.String())
	s.stderrMu.Unlock()

	if parseErr != nil {
		if stderrStr != "" {
			return fmt.Errorf("%w (stderr: %s)", parseErr, stderrStr)
		}
		return parseErr
	}
	if waitErr != nil {
		if stderrStr != "" {
			return fmt.Errorf("%w (stderr: %s)", waitErr, stderrStr)
		}
		return waitErr
	}
	if exitCode != 0 {
		s.logger.Warn("claude exited with non-zero code: exitCode=%d stderr=%s", exitCode, stderrStr)
		if stderrStr != "" {
			return fmt.Errorf("claude CLI exited with code %d: %s", exitCode, stderrStr)
		}
		return fmt.Errorf("claude CLI exited with code %d", exitCode)
	}
	return nil
}
