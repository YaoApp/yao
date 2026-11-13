package computer_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yaoapp/yao/e2e/taocloud"
	modulecomputer "github.com/yaoapp/yao/modules/computer"
	computertypes "github.com/yaoapp/yao/modules/computer/types"
	taocomputer "github.com/yaoapp/yao/modules/tao/computer"
	"golang.org/x/crypto/ssh"
)

func startTaoFakeSSH(t *testing.T) (string, string, *[]string) {
	t.Helper()
	_, hostPriv, _ := ed25519.GenerateKey(rand.Reader)
	hostSigner, _ := ssh.NewSignerFromKey(hostPriv)
	config := &ssh.ServerConfig{PublicKeyCallback: func(ssh.ConnMetadata, ssh.PublicKey) (*ssh.Permissions, error) { return nil, nil }}
	config.AddHostKey(hostSigner)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	commands := &[]string{}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go serveTaoFakeSSH(conn, config, commands)
		}
	}()
	_, clientPriv, _ := ed25519.GenerateKey(rand.Reader)
	block, _ := ssh.MarshalPrivateKey(clientPriv, "")
	return ln.Addr().String(), string(pem.EncodeToMemory(block)), commands
}

func serveTaoFakeSSH(conn net.Conn, config *ssh.ServerConfig, commands *[]string) {
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
				*commands = append(*commands, payload.Command)
				_ = req.Reply(true, nil)
				_, _ = channel.Write([]byte("ok"))
				_, _ = channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{0}))
				return
			}
		}()
	}
}

func TestGetStartsTaiThroughRealSSHController(t *testing.T) {
	srv := taocloud.New()
	t.Cleanup(srv.Close)
	client := taocomputer.NewClient(srv.URL())
	created, err := client.Create(context.Background(), testTAOKey, validCreate("flex-2c2g"))
	if err != nil {
		t.Fatal(err)
	}
	addr, key, commands := startTaoFakeSSH(t)
	host, port, _ := net.SplitHostPort(addr)
	instance := *created
	instance.SSH = taocomputer.SSHInfo{Host: host, Port: mustPort(t, port), User: "test"}
	instance.Status = "running"
	// Replace the client with a local server response carrying the SSH target.
	local := staticInstanceServer(t, instance)
	client = taocomputer.NewClient(local.URL)

	computers := &fakeComputerService{computer: &computertypes.Computer{ID: "cmp_ssh", NodeID: "node_ssh", InstanceID: created.InstanceID, Status: computertypes.StatusProvisioning, Metadata: map[string]interface{}{"ssh_key_ref": "vault://test/key"}}}
	svc := taocomputer.NewService(client, computers).
		WithTai(modulecomputer.NewSSHTaiController()).
		WithSSHKeys(taocomputer.VaultSSHKeyResolver{Get: func(context.Context, string) (string, error) { return key, nil }})
	if _, err := svc.Get(context.Background(), testTAOKey, created.InstanceID); err != nil {
		t.Fatalf("Get real SSH start: %v", err)
	}
	if len(*commands) == 0 || !strings.Contains((*commands)[0], "'tai' 'start'") {
		t.Fatalf("tai start not executed: %v", *commands)
	}
}

func mustPort(t *testing.T, value string) int {
	t.Helper()
	port := 0
	for _, ch := range value {
		port = port*10 + int(ch-'0')
	}
	return port
}

func staticInstanceServer(t *testing.T, instance taocomputer.Instance) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(instance)
	}))
	t.Cleanup(srv.Close)
	return srv
}
