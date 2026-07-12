//go:build unit

package webproxy

import (
	"context"
	"net"
	"testing"
	"time"
)

// mockConn implements net.Conn for testing the dialer channel logic.
type mockConn struct {
	closed bool
}

func (c *mockConn) Read([]byte) (int, error)         { return 0, nil }
func (c *mockConn) Write([]byte) (int, error)        { return 0, nil }
func (c *mockConn) Close() error                     { c.closed = true; return nil }
func (c *mockConn) LocalAddr() net.Addr              { return nil }
func (c *mockConn) RemoteAddr() net.Addr             { return nil }
func (c *mockConn) SetDeadline(time.Time) error      { return nil }
func (c *mockConn) SetReadDeadline(time.Time) error  { return nil }
func (c *mockConn) SetWriteDeadline(time.Time) error { return nil }

func TestTunnelDialer_WarmupHit(t *testing.T) {
	d := &tunnelDialer{
		warmCh: make(chan net.Conn, 1),
	}

	// Simulate warmup by pushing a connection
	mc := &mockConn{}
	d.warmCh <- mc

	conn, err := d.DialContext(context.Background(), "tcp", "tunnel:80")
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if conn != mc {
		t.Fatal("expected warm connection to be returned")
	}
}

func TestTunnelDialer_WarmupMiss(t *testing.T) {
	d := &tunnelDialer{
		taiID:       "nonexistent-tai",
		containerID: "nonexistent-container",
		targetPort:  9999,
		warmCh:      make(chan net.Conn, 1),
	}

	// No warm connection available — should fall through to ForwardProxy
	// which will fail because there's no tunnel handler initialized.
	_, err := d.DialContext(context.Background(), "tcp", "tunnel:80")
	if err == nil {
		t.Fatal("expected error from ForwardProxy fallback with no handler")
	}
}

func TestTunnelDialer_Close_DrainsWarmCh(t *testing.T) {
	d := &tunnelDialer{
		warmCh: make(chan net.Conn, 1),
	}

	mc := &mockConn{}
	d.warmCh <- mc

	d.Close()

	if !mc.closed {
		t.Fatal("expected warm connection to be closed during Close()")
	}

	// Channel should be empty
	select {
	case <-d.warmCh:
		t.Fatal("warmCh should be empty after Close()")
	default:
	}
}

func TestTunnelDialer_Close_EmptyChannel(t *testing.T) {
	d := &tunnelDialer{
		warmCh: make(chan net.Conn, 1),
	}

	// Close with empty channel should not panic
	d.Close()
}

func TestTunnelDialer_WarmupRace(t *testing.T) {
	d := &tunnelDialer{
		warmCh: make(chan net.Conn, 1),
	}

	// Fill the channel first
	mc1 := &mockConn{}
	d.warmCh <- mc1

	// Simulate a second warmup attempt that finds the channel full
	mc2 := &mockConn{}
	select {
	case d.warmCh <- mc2:
		t.Fatal("should not be able to push to full warmCh")
	default:
		mc2.Close()
	}

	if !mc2.closed {
		t.Fatal("overflow connection should be closed")
	}
	if mc1.closed {
		t.Fatal("original warm connection should still be open")
	}
}
