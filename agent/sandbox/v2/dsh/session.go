package dsh

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	infra "github.com/yaoapp/yao/sandbox/v2"
)

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
	opts := []infra.ExecOption{
		infra.WithWorkDir(cmd.workDir),
		infra.WithEnv(cmd.env),
		infra.WithKeepStdinOpen(),
	}
	if len(cmd.stdin) > 0 {
		opts = append(opts, infra.WithStdin(cmd.stdin))
	}

	logger.Info("dsh session starting: workDir=%s platform=%s stdinLen=%d chatID=%s",
		cmd.workDir, p.OS(), len(cmd.stdin), chatID)

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
		chatID:   chatID,
	}, nil
}

func (s *session) runStream(handler message.StreamFunc) (completed bool, err error) {
	s.collectStderr()

	cleanup := s.watchCancel()
	defer cleanup()

	parser := newStreamParser(handler, s.chatID)
	parseErr := parser.parse(s.ctx, s.exec.Stdout)

	s.logger.Debug("runStream: parse returned completed=%v parseErr=%v", parser.completed, parseErr)

	if parser.completed {
		s.logger.Info("dsh stream completed normally")
		return true, nil
	}

	if parseErr != nil && !isContextErr(parseErr) {
		s.exec.Cancel()
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
		s.logger.Warn("dsh exited with code 0 but stream incomplete and stderr present: %s", stderrStr)
		errMsg := fmt.Errorf("dsh setup failed: %s", stderrStr)
		if handler != nil {
			handler(message.ChunkError, []byte(errMsg.Error()))
		}
		return false, errMsg
	}

	return false, nil
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
				s.logger.Debug("dsh stderr: %s", chunk)
			}
			if err != nil {
				return
			}
		}
	}()
}

func (s *session) killProcess(ctx context.Context) {
	if s.chatID != "" {
		s.computer.Exec(ctx, s.plat.KillSessionCmd("dsh-"+s.chatID))
		return
	}
	s.computer.Exec(ctx, s.plat.KillCmd("dsh-jsonrpc-agent"))
}

func (s *session) watchCancel() func() {
	done := make(chan struct{})
	go func() {
		select {
		case <-s.ctx.Done():
			s.logger.Info("context cancelled, initiating graceful stop")
			stdinOK := true

			// Layer 1: session/cancel RPC (idempotent x3, 1s each)
			if stdinOK {
				for i := 0; i < 3; i++ {
					if err := s.writeStdin(s.buildCancelRPC()); err != nil {
						s.logger.Warn("stdin broken, skipping to signal layer: %v", err)
						stdinOK = false
						break
					}
					if s.waitDone(done, 1*time.Second) {
						return
					}
				}
			}

			// Layer 2: shutdown RPC (1s)
			if stdinOK {
				s.logger.Warn("cancel timeout, sending shutdown RPC")
				if err := s.writeStdin(s.buildShutdownRPC()); err != nil {
					stdinOK = false
				} else if s.waitDone(done, 1*time.Second) {
					return
				}
			}

			// Layer 3: SIGTERM (POSIX only, 2s)
			if s.plat.OS() != "windows" {
				s.logger.Warn("stdin layers exhausted, sending SIGTERM")
				s.gracefulKill()
				if s.waitDone(done, 2*time.Second) {
					return
				}
			}

			// Layer 4: SIGKILL / taskkill /F
			s.logger.Error("dsh hung, force killing")
			killCtx, killCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer killCancel()
			s.killProcess(killCtx)
			s.exec.Cancel()

		case <-done:
		}
	}()
	return func() { close(done) }
}

func (s *session) waitDone(done <-chan struct{}, timeout time.Duration) bool {
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

func (s *session) writeStdin(data []byte) error {
	if s.exec.Stdin == nil {
		return fmt.Errorf("stdin is nil")
	}
	_, err := s.exec.Stdin.Write(data)
	return err
}

func (s *session) buildCancelRPC() []byte {
	id, _ := json.Marshal(s.chatID)
	return []byte(fmt.Sprintf(
		"{\"jsonrpc\":\"2.0\",\"method\":\"session/cancel\",\"params\":{\"sessionId\":%s},\"id\":9998}\n",
		id))
}

func (s *session) buildShutdownRPC() []byte {
	return []byte("{\"jsonrpc\":\"2.0\",\"method\":\"shutdown\",\"id\":9999}\n")
}

func (s *session) gracefulKill() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	s.computer.Exec(ctx, s.plat.GracefulKillSessionCmd("dsh-"+s.chatID))
}

func isContextErr(err error) bool {
	return err != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
}

func (s *session) shutdown() {
	s.logger.Info("shutting down completed dsh session: chatID=%s", s.chatID)

	// Close stdin → DSH detects EOF → triggers natural exit via onIdle handler.
	if s.exec.Stdin != nil {
		s.exec.Stdin.Close()
	}

	// Grace period: let DSH flush write-behind buffer (~200ms), dispose
	// plugins, and kill child processes on its own.
	time.Sleep(2 * time.Second)

	// Safety net: force kill anything still lingering.
	killCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.killProcess(killCtx)
	s.exec.Cancel()
}

func (s *session) waitForExit(parseErr error) error {
	s.logger.Info("dsh stream did not complete normally, waiting for exit")

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
			s.logger.Error("dsh did not exit after kill, timeout")
			return fmt.Errorf("dsh did not exit after kill (timeout)")
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
		if stderrStr != "" {
			return fmt.Errorf("dsh exited with code %d: %s", exitCode, stderrStr)
		}
		return fmt.Errorf("dsh exited with code %d", exitCode)
	}
	return nil
}
