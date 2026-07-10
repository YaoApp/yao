package webproxy

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

// Binding represents a single active port proxy binding.
type Binding struct {
	HostPort    int
	TargetID    string
	ContainerID string
	TargetPort  int
	Label       string
	TaiID       string
	Mode        ConnMode
	TargetAddr  string // resolved target address (e.g. "127.0.0.1:3000")
	Server      *http.Server
	Cancel      context.CancelFunc
	CreatedAt   time.Time

	activeAt    atomic.Int64
	idleTimeout time.Duration
}

// ConnMode represents the connection strategy.
type ConnMode int

const (
	ModeDirect ConnMode = iota
	ModeRelay
)

// HostID is the special target_id for host mode.
const HostID = "__host__"

// Info returns a JSON-serializable snapshot of this binding.
func (b *Binding) Info() *BindingInfo {
	return &BindingInfo{
		HostPort:   b.HostPort,
		TargetID:   b.TargetID,
		TargetPort: b.TargetPort,
		Label:      b.Label,
		Status:     "ready",
		CreatedAt:  b.CreatedAt,
	}
}

// LastActive returns the time of the last request handled.
func (b *Binding) LastActive() time.Time {
	ts := b.activeAt.Load()
	if ts == 0 {
		return b.CreatedAt
	}
	return time.Unix(0, ts)
}

func (b *Binding) touch() {
	b.activeAt.Store(time.Now().UnixNano())
}

// Start launches the HTTP listener on an already-open net.Listener.
// Passing the listener directly eliminates the TOCTOU race between
// port-availability check and actual listen.
func (b *Binding) Start(ctx context.Context, ln net.Listener) error {
	inner := b.buildHandler()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.auth" {
			b.handleAuth(w, r)
			return
		}
		authMiddleware(inner).ServeHTTP(w, r)
	})
	b.Server = &http.Server{Handler: handler}

	go func() {
		if err := b.Server.Serve(ln); err != nil && err != http.ErrServerClosed {
			// Log error but don't crash
		}
	}()

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		b.Server.Shutdown(shutCtx)
	}()

	return nil
}

// Stop gracefully shuts down the binding.
func (b *Binding) Stop() {
	b.Cancel()
}

// handleAuth sets the proxy auth cookie and redirects to the target service.
func (b *Binding) handleAuth(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     proxyAuthCookie,
		Value:    url.QueryEscape("Bearer " + token),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})
	redirect := r.URL.Query().Get("redirect")
	if redirect == "" {
		redirect = "/"
	}
	http.Redirect(w, r, redirect, http.StatusFound)
}

func (b *Binding) buildHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b.touch()

		if isWebSocket(r) {
			b.handleWebSocket(w, r)
			return
		}

		target, _ := url.Parse("http://" + b.TargetAddr)
		proxy := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = target.Scheme
				req.URL.Host = target.Host
				req.Host = r.Host
				stripProxyCookie(req)
			},
			FlushInterval: -1,
		}

		if b.Mode == ModeRelay {
			proxy.Transport = &relayTransport{
				containerID: b.ContainerID,
				targetPort:  b.TargetPort,
				relayAddr:   b.TargetAddr,
			}
		}

		proxy.ServeHTTP(w, r)
	})
}

func (b *Binding) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	var targetConn net.Conn
	var err error

	if b.Mode == ModeRelay {
		targetConn, err = dialRelay(b.TargetAddr, b.TargetPort)
	} else {
		targetConn, err = net.DialTimeout("tcp", b.TargetAddr, 5*time.Second)
	}
	if err != nil {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer targetConn.Close()

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "WebSocket hijack not supported", http.StatusInternalServerError)
		return
	}

	clientConn, clientBuf, err := hj.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	stripProxyCookie(r)
	if err := r.Write(targetConn); err != nil {
		return
	}

	if clientBuf.Reader.Buffered() > 0 {
		buffered := make([]byte, clientBuf.Reader.Buffered())
		clientBuf.Read(buffered)
		targetConn.Write(buffered)
	}

	done := make(chan struct{}, 2)
	go func() {
		io.Copy(targetConn, clientConn)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(clientConn, targetConn)
		done <- struct{}{}
	}()
	<-done
}

// isWebSocket checks both Upgrade and Connection headers per RFC 6455.
func isWebSocket(r *http.Request) bool {
	upgrade := false
	for _, v := range r.Header["Upgrade"] {
		if strings.EqualFold(v, "websocket") {
			upgrade = true
			break
		}
	}
	if !upgrade {
		return false
	}
	for _, v := range r.Header["Connection"] {
		for _, part := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(part), "upgrade") {
				return true
			}
		}
	}
	return false
}
