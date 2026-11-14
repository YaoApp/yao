package computer

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/yaoapp/yao/modules/computer/types"
	"golang.org/x/crypto/ssh"
)

// SSHTaiController controls a Tai process on a remote host over SSH.
type SSHTaiController struct {
	Timeout time.Duration
}

// NewSSHTaiController creates the default SSH controller.
func NewSSHTaiController() *SSHTaiController {
	return &SSHTaiController{Timeout: 20 * time.Second}
}

// Bootstrap creates the data directory and verifies that Tai is installed.
func (c *SSHTaiController) Bootstrap(ctx context.Context, target types.SSHTarget, req types.BootstrapRequest) error {
	dir := req.DataDir
	if dir == "" {
		dir = "/data/volumes"
	}
	cmd := "mkdir -p " + shellQuote(dir) + " && command -v tai >/dev/null"
	_, err := c.run(ctx, target, cmd)
	return err
}

// Start launches Tai v2 in the background.
func (c *SSHTaiController) Start(ctx context.Context, target types.SSHTarget, req types.StartTaiRequest) error {
	args := []string{"nohup", "tai", "start", "--node-id", req.NodeID}
	if req.DataDir != "" {
		args = append(args, "--data", req.DataDir)
	}
	if req.ServerURL != "" {
		args = append(args, "--server", req.ServerURL)
	}
	if req.NodeToken != "" {
		args = append(args, "--node-token", req.NodeToken)
	}
	if req.SandboxImage != "" {
		args = append(args, "--sandbox-image", req.SandboxImage)
	}
	if req.LogLevel != "" {
		args = append(args, "--log-level", req.LogLevel)
	}
	if req.LogFormat != "" {
		args = append(args, "--log-format", req.LogFormat)
	}
	if req.LogFile != "" {
		args = append(args, "--log-file", req.LogFile)
	}
	if req.ShutdownTimeout > 0 {
		args = append(args, "--shutdown-timeout", req.ShutdownTimeout.String())
	}
	logFile := req.LogFile
	if logFile == "" {
		logFile = "logs/tai.log"
	}
	cmd := "mkdir -p " + shellQuote(req.DataDir) + " " + shellQuote(filepath.Dir(logFile)) + " && " + strings.Join(quoteArgs(args), " ") + " > " + shellQuote(logFile) + " 2>&1 &"
	_, err := c.run(ctx, target, cmd)
	return err
}

// Stop terminates the Tai process.
func (c *SSHTaiController) Stop(ctx context.Context, target types.SSHTarget) error {
	_, err := c.run(ctx, target, "pkill -f 'tai start' || true")
	return err
}

// Status reports whether a Tai process is running.
func (c *SSHTaiController) Status(ctx context.Context, target types.SSHTarget) (*types.TaiStatus, error) {
	out, err := c.run(ctx, target, "pgrep -af 'tai start' || true")
	if err != nil {
		return nil, err
	}
	return &types.TaiStatus{Running: strings.TrimSpace(out) != ""}, nil
}

// Logs returns the recent Tai log lines.
func (c *SSHTaiController) Logs(ctx context.Context, target types.SSHTarget, lines int) (string, error) {
	if lines <= 0 {
		lines = 200
	}
	return c.run(ctx, target, "tail -n "+strconv.Itoa(lines)+" logs/tai.log 2>/dev/null || true")
}

func (c *SSHTaiController) run(ctx context.Context, target types.SSHTarget, command string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if target.Host == "" || target.PrivateKey == "" {
		return "", fmt.Errorf("ssh target requires host and private key")
	}
	port := target.Port
	if port == 0 {
		port = 22
	}
	config := &ssh.ClientConfig{
		User:            target.User,
		Auth:            []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         c.timeout(),
	}
	signer, err := ssh.ParsePrivateKey([]byte(target.PrivateKey))
	if err != nil {
		return "", fmt.Errorf("parse ssh key: %w", err)
	}
	config.Auth = append(config.Auth, ssh.PublicKeys(signer))
	client, err := ssh.Dial("tcp", net.JoinHostPort(target.Host, strconv.Itoa(port)), config)
	if err != nil {
		return "", err
	}
	defer client.Close()
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	done := make(chan error, 1)
	go func() { done <- session.Run(command) }()
	select {
	case <-ctx.Done():
		_ = client.Close()
		return stdout.String(), ctx.Err()
	case err := <-done:
		if err != nil {
			return stdout.String(), fmt.Errorf("ssh command: %w: %s", err, strings.TrimSpace(stderr.String()))
		}
	}
	return stdout.String(), nil
}

func (c *SSHTaiController) timeout() time.Duration {
	if c != nil && c.Timeout > 0 {
		return c.Timeout
	}
	return 20 * time.Second
}

func quoteArgs(args []string) []string {
	out := make([]string, len(args))
	for i, arg := range args {
		out[i] = shellQuote(arg)
	}
	return out
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
