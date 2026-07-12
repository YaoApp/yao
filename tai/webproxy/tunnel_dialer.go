package webproxy

import (
	"context"
	"net"

	"github.com/yaoapp/yao/tai/tunnel"
)

// tunnelDialer implements a custom DialContext for http.Transport that uses
// gRPC Forward streams as the underlying connection. It supports pre-warming
// one connection via Warmup() so the first HTTP request experiences zero
// tunnel-establishment latency.
type tunnelDialer struct {
	taiID       string
	containerID string
	targetPort  int
	warmCh      chan net.Conn
}

func newTunnelDialer(taiID, containerID string, targetPort int) *tunnelDialer {
	return &tunnelDialer{
		taiID:       taiID,
		containerID: containerID,
		targetPort:  targetPort,
		warmCh:      make(chan net.Conn, 1),
	}
}

// DialContext is called by http.Transport for each new connection.
// It first checks for a pre-warmed connection before falling back to
// establishing a new Forward stream.
func (d *tunnelDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	select {
	case conn := <-d.warmCh:
		return conn, nil
	default:
	}
	return tunnel.ForwardProxy(d.taiID, d.containerID, d.targetPort)
}

// Warmup pre-establishes a single Forward stream connection and stores it
// for immediate use by the first DialContext call.
func (d *tunnelDialer) Warmup() {
	conn, err := tunnel.ForwardProxy(d.taiID, d.containerID, d.targetPort)
	if err != nil {
		return
	}
	select {
	case d.warmCh <- conn:
	default:
		conn.Close()
	}
}

// Close drains the warm channel and closes any pre-warmed connection
// that was never consumed.
func (d *tunnelDialer) Close() {
	select {
	case conn := <-d.warmCh:
		conn.Close()
	default:
	}
}
