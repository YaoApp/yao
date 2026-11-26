package computer

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yaoapp/yao/modules/computer/types"
	"golang.org/x/crypto/ssh"
)

func startFakeSSHServer(t *testing.T) (string, string) {
	addr, key, _ := startRecordingSSHServer(t)
	return addr, key
}

func startRecordingSSHServer(t *testing.T) (string, string, *commandRecorder) {
	t.Helper()
	recorder := &commandRecorder{}
	currentRecorder = recorder
	_, hostPriv, _ := ed25519.GenerateKey(rand.Reader)
	hostSigner, _ := ssh.NewSignerFromKey(hostPriv)
	config := &ssh.ServerConfig{PublicKeyCallback: func(ssh.ConnMetadata, ssh.PublicKey) (*ssh.Permissions, error) { return nil, nil }}
	config.AddHostKey(hostSigner)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			recorder := currentRecorder
			go serveFakeSSH(conn, config, recorder)
		}
	}()
	_, clientPriv, _ := ed25519.GenerateKey(rand.Reader)
	block, _ := ssh.MarshalPrivateKey(clientPriv, "")
	return ln.Addr().String(), string(pem.EncodeToMemory(block)), recorder
}

type commandRecorder struct {
	mu       sync.Mutex
	commands []string
}

func (r *commandRecorder) add(command string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commands = append(r.commands, command)
}

func (r *commandRecorder) all() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.commands...)
}

var currentRecorder *commandRecorder

func serveFakeSSH(conn net.Conn, config *ssh.ServerConfig, recorder *commandRecorder) {
	serverConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		return
	}
	defer serverConn.Close()
	go ssh.DiscardRequests(reqs)
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			_ = newChannel.Reject(ssh.UnknownChannelType, "session only")
			continue
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			continue
		}
		go func() {
			defer channel.Close()
			for req := range requests {
				if req.Type != "exec" {
					_ = req.Reply(false, nil)
					continue
				}
				var payload struct{ Command string }
				_ = ssh.Unmarshal(req.Payload, &payload)
				_ = req.Reply(true, nil)
				if recorder != nil {
					recorder.add(payload.Command)
				}
				out := "ok"
				if strings.Contains(payload.Command, "pgrep") {
					out = "123 tai start"
				}
				if strings.Contains(payload.Command, "tail") {
					out = "log line"
				}
				_, _ = channel.Write([]byte(out))
				_, _ = channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{0}))
				return
			}
		}()
	}
}

func TestSSHTaiControllerAgainstFakeServer(t *testing.T) {
	addr, key := startFakeSSHServer(t)
	host, port, _ := net.SplitHostPort(addr)
	target := types.SSHTarget{Host: host, Port: mustPort(t, port), User: "tester", PrivateKey: key}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c := NewSSHTaiController()
	if err := c.Bootstrap(ctx, target, types.BootstrapRequest{DataDir: "/tmp/data"}); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if err := c.Start(ctx, target, types.StartTaiRequest{NodeID: "node-x", DataDir: "/tmp/data", ServerURL: "https://yao", NodeToken: "tok", SandboxImage: "img", LogLevel: "info", LogFormat: "json", LogFile: "/tmp/tai.log"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	status, err := c.Status(ctx, target)
	if err != nil || !status.Running {
		t.Fatalf("status=%+v err=%v", status, err)
	}
	logs, err := c.Logs(ctx, target, 10)
	if err != nil || !strings.Contains(logs, "log line") {
		t.Fatalf("logs=%q err=%v", logs, err)
	}
	if err := c.Stop(ctx, target); err != nil {
		t.Fatalf("stop: %v", err)
	}
}

func mustPort(t *testing.T, value string) int {
	t.Helper()
	var port int
	for _, ch := range value {
		port = port*10 + int(ch-'0')
	}
	return port
}

var _ = sync.Mutex{}

func TestSSHTaiControllerUsesAbsoluteWritablePaths(t *testing.T) {
	addr, key, recorder := startRecordingSSHServer(t)
	host, port, _ := net.SplitHostPort(addr)
	target := types.SSHTarget{Host: host, Port: mustPort(t, port), User: "max", PrivateKey: key}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c := NewSSHTaiController()
	if err := c.Bootstrap(ctx, target, types.BootstrapRequest{NodeID: "node-x", NodeToken: "tok"}); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if err := c.Start(ctx, target, types.StartTaiRequest{NodeID: "node-x", NodeToken: "tok"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	commands := recorder.all()
	if len(commands) < 2 {
		t.Fatalf("commands=%v", commands)
	}
	dataDir := "/home/max/.tai"
	logFile := "/home/max/.tai/logs/tai.log"
	if !strings.Contains(commands[0], dataDir) {
		t.Fatalf("bootstrap does not use home data dir: %s", commands[0])
	}
	start := commands[1]
	if !strings.Contains(start, "'--data' '"+dataDir+"'") || !strings.Contains(start, "'--log-file' '"+logFile+"'") {
		t.Fatalf("start does not use absolute paths: %s", start)
	}
	if strings.Contains(start, "--log-file 'logs/tai.log'") || strings.Contains(start, "> 'logs/tai.log'") {
		t.Fatalf("start still passes relative log path: %s", start)
	}
}

func TestNormalizeTaiPaths(t *testing.T) {
	target := types.SSHTarget{User: "max"}
	cases := map[string]string{
		"":          "/home/max/.tai",
		"~":         "/home/max",
		"~/data":    "/home/max/data",
		"data":      "/home/max/data",
		"/data/tai": "/data/tai",
	}
	for input, want := range cases {
		if got := normalizeDataDir(target, input); got != want {
			t.Fatalf("dataDir(%q)=%q want %q", input, got, want)
		}
	}
	if got := normalizeLogFile(target, "/home/max/.tai", ""); got != "/home/max/.tai/logs/tai.log" {
		t.Fatalf("default log=%q", got)
	}
	if got := normalizeLogFile(target, "/home/max/.tai", "logs/custom.log"); got != "/home/max/.tai/logs/custom.log" {
		t.Fatalf("relative log=%q", got)
	}
	if got := normalizeDataDir(types.SSHTarget{User: "root"}, ""); got != "/root/.tai" {
		t.Fatalf("root data dir=%q", got)
	}
}
