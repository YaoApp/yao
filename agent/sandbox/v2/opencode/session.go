package opencode

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	infra "github.com/yaoapp/yao/sandbox/v2"
)

// session encapsulates a single OpenCode CLI execution lifecycle:
// process start, stderr collection, kill on cancel, Wait with timeout.
type session struct {
	ctx      context.Context
	computer infra.Computer
	plat     platform
	exec     *infra.ExecStream
	stderr   strings.Builder
	stderrMu sync.Mutex
	logger   *agentContext.RequestLogger
	chatID   string
}

func startSession(ctx context.Context, computer infra.Computer, p platform, cmd command, chatID string, logger *agentContext.RequestLogger) (*session, error) {
	opts := []infra.ExecOption{infra.WithWorkDir(cmd.workDir), infra.WithEnv(cmd.env)}

	logger.Info("opencode session starting: cmd=%v workDir=%s platform=%s chatID=%s",
		cmd.shell, cmd.workDir, p.OS(), chatID)

	execStream, err := computer.Stream(ctx, cmd.shell, opts...)
	if err != nil {
		return nil, fmt.Errorf("computer.Stream: %w", err)
	}

	// Write user message to stdin, then close. OpenCode reads the prompt
	// from stdin when no positional message is given (same as Claude runner).
	// Closing after write signals EOF so OpenCode begins processing.
	if execStream.Stdin != nil {
		if cmd.stdin != "" {
			if _, err := io.WriteString(execStream.Stdin, cmd.stdin); err != nil {
				logger.Warn("failed to write stdin: %v", err)
			}
		}
		execStream.Stdin.Close()
	}

	return &session{
		ctx:      ctx,
		computer: computer,
		plat:     p,
		exec:     execStream,
		logger:   logger,
		chatID:   chatID,
	}, nil
}

// runStream executes the main stream processing loop.
// Returns (completed, error) where completed=true means OpenCode CLI sent
// a step_finish with reason=stop and the stream finished normally.
func (s *session) runStream(handler message.StreamFunc) (completed bool, err error) {
	s.collectStderr()

	cleanup := s.watchCancel()
	defer cleanup()

	// Tee stdout to a debug log so we can inspect raw JSONL timing.
	stdout := s.teeStdout()

	parser := newStreamParser(handler)
	parseErr := parser.parse(s.ctx, stdout)

	s.logger.Debug("runStream: parse returned completed=%v parseErr=%v", parser.completed, parseErr)

	if parser.completed {
		s.logger.Info("opencode stream completed normally")
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
		s.logger.Warn("opencode exited with code 0 but stream incomplete and stderr present: %s", stderrStr)
		errMsg := fmt.Errorf("opencode CLI setup failed: %s", stderrStr)
		if handler != nil {
			handler(message.ChunkError, []byte(errMsg.Error()))
		}
		return false, errMsg
	}

	return false, nil
}

// teeStdout wraps exec.Stdout with a TeeReader that writes a copy to a
// timestamped log file. Returns the original Stdout if tee setup fails.
func (s *session) teeStdout() io.ReadCloser {
	logDir := os.Getenv("YAO_LOG_PATH")
	if logDir == "" {
		logDir = "/tmp"
	}
	logFile := filepath.Join(logDir, fmt.Sprintf("opencode-stream-%s-%d.jsonl", s.chatID, time.Now().Unix()))
	f, err := os.Create(logFile)
	if err != nil {
		s.logger.Debug("teeStdout: cannot create %s: %v", logFile, err)
		return s.exec.Stdout
	}
	s.logger.Info("teeStdout: raw JSONL -> %s", logFile)

	tee := io.TeeReader(s.exec.Stdout, f)
	return &teeReadCloser{Reader: tee, closers: []io.Closer{s.exec.Stdout, f}}
}

type teeReadCloser struct {
	io.Reader
	closers []io.Closer
}

func (t *teeReadCloser) Close() error {
	var firstErr error
	for _, c := range t.closers {
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

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
				s.logger.Debug("opencode stderr: %s", chunk)
			}
			if err != nil {
				return
			}
		}
	}()
}

// killProcess terminates the OpenCode CLI process (Node.js).
func (s *session) killProcess(ctx context.Context) {
	if s.chatID != "" {
		name := sanitizeSessionName(s.chatID)
		result, err := s.computer.Exec(ctx, s.plat.KillSessionCmd(name))
		s.logger.Debug("killProcess: KillSessionCmd(%s) exitCode=%d err=%v", name, result.ExitCode, err)
		return
	}
	// OpenCode is a Node.js process; match both "opencode" and "node.*opencode"
	result, err := s.computer.Exec(ctx, s.plat.KillCmd("opencode"))
	s.logger.Debug("killProcess: KillCmd(opencode) exitCode=%d err=%v", result.ExitCode, err)
}

func (s *session) watchCancel() func() {
	done := make(chan struct{})
	go func() {
		select {
		case <-s.ctx.Done():
			s.logger.Info("context cancelled, killing opencode: %v", s.ctx.Err())
			killCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			s.killProcess(killCtx)
			s.exec.Cancel()
		case <-done:
		}
	}()
	return func() { close(done) }
}

// shutdown cleans up after normal stream completion.
//
// OpenCode exits cleanly after step_finish(stop), but we still need to
// release the Docker exec connection. Like Claude runner, we first kill
// only the opencode process with SIGKILL (which cannot be caught, so
// OpenCode has no chance to propagate signals to child processes), then
// close the exec connection. Children (browsers, servers, etc.) that were
// launched via nohup/setsid survive because they are in separate sessions.
func (s *session) shutdown() {
	s.logger.Info("shutting down completed opencode exec session: chatID=%s", s.chatID)
	killCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.killProcess(killCtx)
	s.exec.Cancel()
}

func (s *session) waitForExit(parseErr error) error {
	s.logger.Info("opencode stream did not complete normally, waiting for exit")

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
			s.logger.Error("opencode did not exit after kill, timeout")
			return fmt.Errorf("opencode did not exit after kill (timeout)")
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
		s.logger.Warn("opencode exited with non-zero code: exitCode=%d stderr=%s", exitCode, stderrStr)
		if stderrStr != "" {
			return fmt.Errorf("opencode CLI exited with code %d: %s", exitCode, stderrStr)
		}
		return fmt.Errorf("opencode CLI exited with code %d", exitCode)
	}
	return nil
}
