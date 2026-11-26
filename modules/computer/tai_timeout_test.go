package computer

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/yaoapp/yao/modules/computer/types"
	"golang.org/x/crypto/ssh"
)

func startHangingSSHServer(t *testing.T) (string, string) {
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
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				serverConn, chans, reqs, err := ssh.NewServerConn(conn, config)
				if err != nil {
					return
				}
				defer serverConn.Close()
				go ssh.DiscardRequests(reqs)
				for newChannel := range chans {
					channel, requests, err := newChannel.Accept()
					if err != nil {
						continue
					}
					go func() {
						defer channel.Close()
						for req := range requests {
							if req.Type == "exec" {
								_ = req.Reply(true, nil)
								// Deliberately never send exit-status: simulates a
								// remote shell waiting on a background process.
							}
						}
					}()
				}
			}(conn)
		}
	}()
	_, clientPriv, _ := ed25519.GenerateKey(rand.Reader)
	block, _ := ssh.MarshalPrivateKey(clientPriv, "")
	return ln.Addr().String(), string(pem.EncodeToMemory(block))
}

func TestSSHTaiControllerCommandTimeout(t *testing.T) {
	addr, key := startHangingSSHServer(t)
	host, port, _ := net.SplitHostPort(addr)
	target := types.SSHTarget{Host: host, Port: mustPort(t, port), User: "max", PrivateKey: key}
	c := &SSHTaiController{Timeout: 100 * time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	started := time.Now()
	err := c.Start(ctx, target, types.StartTaiRequest{NodeID: "node-x", NodeToken: "tok"})
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("err=%v", err)
	}
	if time.Since(started) > time.Second {
		t.Fatalf("timeout took too long: %s", time.Since(started))
	}
}
