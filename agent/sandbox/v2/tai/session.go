package tai

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

// session encapsulates a single "tai agent run" subprocess lifecycle:
// start, stderr collection, cancel on context, wait with timeout.
type session struct {
	ctx      context.Context
	computer infra.Computer
	exec     *infra.ExecStream
	stderr   strings.Builder
	stderrMu sync.Mutex
	logger   *agentContext.RequestLogger
	chatID   string
}

func startSession(ctx context.Context, computer infra.Computer, cmd command, chatID string, logger *agentContext.RequestLogger) (*session, error) {
	opts := []infra.ExecOption{
		infra.WithWorkDir(cmd.workDir),
		infra.WithEnv(cmd.env),
	}

	logger.Info("tai session starting: cmd=%v workDir=%s chatID=%s", cmd.shell, cmd.workDir, chatID)

	execStream, err := computer.Stream(ctx, cmd.shell, opts...)
	if err != nil {
		return nil, fmt.Errorf("computer.Stream: %w", err)
	}

	// If the command carries image payload, write it to stdin before closing.
	if cmd.stdinData != nil && execStream.Stdin != nil {
		if _, err := execStream.Stdin.Write(cmd.stdinData); err != nil {
			logger.Warn("failed to write images to tai stdin: %v", err)
		}
	}
	if execStream.Stdin != nil {
		execStream.Stdin.Close()
	}

	return &session{
		ctx:      ctx,
		computer: computer,
		exec:     execStream,
		logger:   logger,
		chatID:   chatID,
	}, nil
}

// runStream runs the NDJSON parse loop over the subprocess stdout.
// Returns (completed, error): completed=true means agent_end was received.
func (s *session) runStream(handler message.StreamFunc, locale string, prepareLoadingMsgID string) (completed bool, err error) {
	s.collectStderr()

	cleanup := s.watchCancel()
	defer cleanup()

	parser := newStreamParser(handler, locale, prepareLoadingMsgID)
	parseErr := parser.parse(s.ctx, s.exec.Stdout)

	s.logger.Debug("runStream: parse returned completed=%v parseErr=%v", parser.completed, parseErr)

	if !parser.completed && parseErr != nil {
		s.exec.Cancel()
	}

	if parser.completed {
		s.logger.Info("tai stream completed normally")
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
		s.logger.Warn("tai exited with code 0 but stream incomplete and stderr present: %s", stderrStr)
		errMsg := fmt.Errorf("tai agent run failed: %s", stderrStr)
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
				s.logger.Debug("tai stderr: %s", chunk)
				fmt.Printf("[YAO-DEBUG] tai stderr: %s\n", chunk)
			}
			if err != nil {
				return
			}
		}
	}()
}

func (s *session) watchCancel() func() {
	done := make(chan struct{})
	go func() {
		select {
		case <-s.ctx.Done():
			s.logger.Info("context cancelled, killing tai: %v", s.ctx.Err())
			s.exec.Cancel()
		case <-done:
		}
	}()
	return func() { close(done) }
}

// shutdown releases subprocess resources after normal completion.
func (s *session) shutdown() {
	s.logger.Info("shutting down completed tai exec session: chatID=%s", s.chatID)
	s.exec.Cancel()
}

func (s *session) waitForExit(parseErr error) error {
	s.logger.Info("tai stream did not complete normally, waiting for exit")

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
			s.logger.Error("tai did not exit after kill, timeout")
			return fmt.Errorf("tai did not exit after kill (timeout)")
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
		s.logger.Warn("tai exited with non-zero code: exitCode=%d stderr=%s", exitCode, stderrStr)
		if stderrStr != "" {
			return fmt.Errorf("tai agent run exited with code %d: %s", exitCode, stderrStr)
		}
		return fmt.Errorf("tai agent run exited with code %d", exitCode)
	}
	return nil
}
