package tunnel

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"github.com/yaoapp/yao/grpc/auth"
	tai "github.com/yaoapp/yao/tai"
	"github.com/yaoapp/yao/tai/registry"
	"github.com/yaoapp/yao/tai/taiid"
	"github.com/yaoapp/yao/tai/tunnel/taipb"
	"github.com/yaoapp/yao/tai/types"
)

var globalHandler *TunnelHandler

// GlobalHandler returns the global TunnelHandler instance set by NewTunnelHandler.
func GlobalHandler() *TunnelHandler { return globalHandler }

// TunnelHandler implements the TaiTunnel gRPC service.
type TunnelHandler struct {
	taipb.UnimplementedTaiTunnelServer
	reg     *registry.Registry
	pending sync.Map // channel_id → chan taipb.TaiTunnel_ForwardServer
	logger  *slog.Logger

	sendMu sync.Map // taiID → *sync.Mutex – serializes Send on each Register stream
}

// NewTunnelHandler creates a TunnelHandler backed by the given registry.
// It also registers a bridge function so that OpenLocalListener uses
// gRPC Forward streams instead of WS data channels.
func NewTunnelHandler(reg *registry.Registry) *TunnelHandler {
	h := &TunnelHandler{
		reg:    reg,
		logger: slog.Default(),
	}
	reg.SetBridgeFunc(h.bridgeConn)
	globalHandler = h
	return h
}

// Register implements the control-plane stream (Tai → Yao).
func (h *TunnelHandler) Register(stream taipb.TaiTunnel_RegisterServer) error {
	msg, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("recv register: %w", err)
	}
	if msg.Type != "register" {
		return fmt.Errorf("expected register, got %q", msg.Type)
	}
	if msg.NodeId == "" || msg.MachineId == "" {
		return fmt.Errorf("register: node_id and machine_id required")
	}

	resolvedTaiID, err := taiid.Generate(msg.MachineId, msg.NodeId)
	if err != nil {
		return fmt.Errorf("taiid: %w", err)
	}

	authInfo := authInfoFromStream(stream)
	remoteIP := ""
	if p, ok := peer.FromContext(stream.Context()); ok {
		if host, _, err := net.SplitHostPort(p.Addr.String()); err == nil {
			remoteIP = host
		}
	}

	node := &registry.TaiNode{
		TaiID:        resolvedTaiID,
		MachineID:    msg.MachineId,
		Version:      msg.Version,
		DisplayName:  msg.DisplayName,
		Auth:         authInfo,
		System:       systemFromProto(msg.System),
		Mode:         "tunnel",
		Addr:         "tunnel://" + remoteIP,
		Ports:        portsFromProto(msg.Ports),
		Capabilities: capsFromProto(msg.Caps),
	}

	var mu sync.Mutex
	h.sendMu.Store(resolvedTaiID, &mu)

	h.reg.Register(node)
	h.reg.SetRegisterStream(resolvedTaiID, stream)
	defer func() {
		h.sendMu.Delete(resolvedTaiID)
		h.reg.Unregister(resolvedTaiID)
		h.logger.Info("tai gRPC tunnel disconnected", "tai_id", resolvedTaiID)
	}()

	mu.Lock()
	err = stream.Send(&taipb.TunnelControl{
		Type:  "registered",
		TaiId: resolvedTaiID,
	})
	mu.Unlock()
	if err != nil {
		return fmt.Errorf("send registered: %w", err)
	}

	h.logger.Info("tai gRPC tunnel connected", "tai_id", resolvedTaiID, "version", msg.Version)

	go h.connectTunnelNode(resolvedTaiID)

	const pingTimeout = 90 * time.Second
	recvCh := make(chan *taipb.TunnelControl)
	errCh := make(chan error, 1)
	go func() {
		for {
			ctrl, err := stream.Recv()
			if err != nil {
				errCh <- err
				return
			}
			recvCh <- ctrl
		}
	}()

	timer := time.NewTimer(pingTimeout)
	defer timer.Stop()

	for {
		select {
		case ctrl := <-recvCh:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(pingTimeout)

			switch ctrl.Type {
			case "ping":
				h.reg.UpdatePing(resolvedTaiID)
				mu.Lock()
				sendErr := stream.Send(&taipb.TunnelControl{Type: "pong"})
				mu.Unlock()
				if sendErr != nil {
					return sendErr
				}
			}

		case err := <-errCh:
			if err == io.EOF {
				return nil
			}
			return err

		case <-timer.C:
			h.logger.Warn("tai ping timeout, closing tunnel", "tai_id", resolvedTaiID, "timeout", pingTimeout)
			return fmt.Errorf("tai %s: ping timeout (%s)", resolvedTaiID, pingTimeout)
		}
	}
}

// Forward implements the data-plane stream (Tai → Yao).
func (h *TunnelHandler) Forward(stream taipb.TaiTunnel_ForwardServer) error {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return fmt.Errorf("missing metadata")
	}
	vals := md.Get("channel_id")
	if len(vals) == 0 || vals[0] == "" {
		return fmt.Errorf("missing channel_id in metadata")
	}
	channelID := vals[0]

	short := registry.ShortChannelID(channelID)
	h.logger.Debug("[forward] Forward stream arrived", "channel_id", short)

	if ch, ok := h.pending.LoadAndDelete(channelID); ok {
		ch.(chan taipb.TaiTunnel_ForwardServer) <- stream
	} else {
		h.logger.Warn("[forward] no pending channel (expired?)", "channel_id", short)
		return fmt.Errorf("no pending channel for %s", channelID)
	}

	<-stream.Context().Done()
	return nil
}

// RequestForward sends an "open" command to Tai via the Register stream and
// waits for Tai to call back with a Forward stream. Returns the Forward stream.
//
// route may be nil for raw TCP tunnels (gRPC, Docker API, K8s API).
func (h *TunnelHandler) RequestForward(taiID string, route *forwardRoute) (taipb.TaiTunnel_ForwardServer, error) {
	stream := h.reg.GetRegisterStream(taiID)
	if stream == nil {
		return nil, fmt.Errorf("tai %s: no active register stream", taiID)
	}

	muVal, ok := h.sendMu.Load(taiID)
	if !ok {
		return nil, fmt.Errorf("tai %s: no send mutex (stream closing?)", taiID)
	}
	mu := muVal.(*sync.Mutex)

	channelID, err := registry.GenerateChannelID()
	if err != nil {
		return nil, fmt.Errorf("generate channel_id: %w", err)
	}

	waitCh := make(chan taipb.TaiTunnel_ForwardServer, 1)
	h.pending.Store(channelID, waitCh)
	defer h.pending.Delete(channelID)

	regStream, ok := stream.(taipb.TaiTunnel_RegisterServer)
	if !ok {
		return nil, fmt.Errorf("tai %s: register stream type mismatch", taiID)
	}

	ctrl := &taipb.TunnelControl{
		Type:      "open",
		ChannelId: channelID,
	}
	if route != nil {
		ctrl.ChannelType = route.channelType
		ctrl.ContainerId = route.containerID
		ctrl.ContainerPort = int32(route.containerPort)
	}

	short := registry.ShortChannelID(channelID)
	h.logger.Debug("[forward] sending open command",
		"tai_id", taiID, "channel_type", ctrl.ChannelType,
		"container", ctrl.ContainerId, "channel_id", short)

	mu.Lock()
	sendErr := regStream.Send(ctrl)
	mu.Unlock()
	if sendErr != nil {
		return nil, fmt.Errorf("send open: %w", sendErr)
	}

	h.logger.Debug("[forward] open sent, waiting for callback",
		"tai_id", taiID, "channel_id", short)

	select {
	case fwd := <-waitCh:
		h.logger.Debug("[forward] callback received",
			"tai_id", taiID, "channel_id", short)
		return fwd, nil
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("tai %s: forward timeout (10s) channel=%s", taiID, short)
	case <-regStream.Context().Done():
		return nil, fmt.Errorf("tai %s: register stream closed while waiting for forward", taiID)
	}
}

// connectTunnelNode establishes gRPC resources to the Tai node through the tunnel.
func (h *TunnelHandler) connectTunnelNode(taiID string) {
	res, err := tai.DialTunnel(taiID, h.reg)
	if err != nil {
		h.logger.Warn("failed to connect tunnel node",
			"tai_id", taiID, "err", err)
		return
	}
	h.reg.SetResources(taiID, res)
	h.logger.Info("tunnel node resources connected", "tai_id", taiID)
}

// bridgeConn bridges a local TCP connection to a Tai port via gRPC Forward stream.
// Called by registry.OpenLocalListener for each accepted TCP connection.
// Uses raw TCP forwarding (TargetPort only, no container routing).
func (h *TunnelHandler) bridgeConn(taiID string, targetPort int, localConn net.Conn) {
	fwd, err := h.requestForwardRaw(taiID, targetPort)
	if err != nil {
		localConn.Close()
		h.logger.Error("request forward failed",
			"tai_id", taiID, "port", targetPort, "err", err)
		return
	}

	streamConn := newForwardConn(fwd)
	bridgeTCP(localConn, streamConn)
}

// requestForwardRaw sends an "open" command with only TargetPort (no container
// routing). Used by bridgeConn for raw TCP tunnels (gRPC, Docker API, K8s API).
func (h *TunnelHandler) requestForwardRaw(taiID string, targetPort int) (taipb.TaiTunnel_ForwardServer, error) {
	stream := h.reg.GetRegisterStream(taiID)
	if stream == nil {
		return nil, fmt.Errorf("tai %s: no active register stream", taiID)
	}

	muVal, ok := h.sendMu.Load(taiID)
	if !ok {
		return nil, fmt.Errorf("tai %s: no send mutex (stream closing?)", taiID)
	}
	mu := muVal.(*sync.Mutex)

	channelID, err := registry.GenerateChannelID()
	if err != nil {
		return nil, fmt.Errorf("generate channel_id: %w", err)
	}

	waitCh := make(chan taipb.TaiTunnel_ForwardServer, 1)
	h.pending.Store(channelID, waitCh)
	defer h.pending.Delete(channelID)

	regStream, ok := stream.(taipb.TaiTunnel_RegisterServer)
	if !ok {
		return nil, fmt.Errorf("tai %s: register stream type mismatch", taiID)
	}

	short := registry.ShortChannelID(channelID)
	h.logger.Debug("[forward] sending open command (raw)",
		"tai_id", taiID, "port", targetPort, "channel_id", short)

	mu.Lock()
	sendErr := regStream.Send(&taipb.TunnelControl{
		Type:       "open",
		ChannelId:  channelID,
		TargetPort: int32(targetPort),
	})
	mu.Unlock()
	if sendErr != nil {
		return nil, fmt.Errorf("send open: %w", sendErr)
	}

	select {
	case fwd := <-waitCh:
		return fwd, nil
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("tai %s: forward timeout (10s) channel=%s", taiID, short)
	case <-regStream.Context().Done():
		return nil, fmt.Errorf("tai %s: register stream closed while waiting for forward", taiID)
	}
}

// forwardConn wraps a Forward stream as a net.Conn-like reader/writer.
type forwardConn struct {
	stream taipb.TaiTunnel_ForwardServer
	buf    []byte
}

func newForwardConn(stream taipb.TaiTunnel_ForwardServer) *forwardConn {
	return &forwardConn{stream: stream}
}

func (c *forwardConn) Read(p []byte) (int, error) {
	if len(c.buf) > 0 {
		n := copy(p, c.buf)
		c.buf = c.buf[n:]
		return n, nil
	}
	msg, err := c.stream.Recv()
	if err != nil {
		return 0, err
	}
	n := copy(p, msg.Data)
	if n < len(msg.Data) {
		c.buf = msg.Data[n:]
	}
	return n, nil
}

func (c *forwardConn) Write(p []byte) (int, error) {
	if err := c.stream.Send(&taipb.ForwardData{Data: p}); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (c *forwardConn) Close() error {
	return nil
}

// bridgeTCP copies bytes bidirectionally, closing both sides when done.
func bridgeTCP(a, b io.ReadWriteCloser) {
	var wg sync.WaitGroup
	wg.Add(2)
	cp := func(dst io.WriteCloser, src io.ReadCloser) {
		defer wg.Done()
		io.Copy(dst, src)
		dst.Close()
	}
	go cp(a, b)
	go cp(b, a)
	wg.Wait()
}

// ── helpers ──────────────────────────────────────────────────────────────────

func authInfoFromStream(stream taipb.TaiTunnel_RegisterServer) types.AuthInfo {
	info := auth.GetAuthorizedInfo(stream.Context())
	if info == nil {
		return types.AuthInfo{}
	}
	return types.AuthInfo{
		Subject:  info.Subject,
		UserID:   info.UserID,
		ClientID: info.ClientID,
		Scope:    info.Scope,
		TeamID:   info.TeamID,
		TenantID: info.TenantID,
	}
}

func portsFromProto(p *taipb.Ports) types.Ports {
	if p == nil {
		return types.Ports{}
	}
	return types.Ports{
		GRPC:   int(p.Grpc),
		HTTP:   int(p.Http),
		VNC:    int(p.Vnc),
		Docker: int(p.Docker),
		K8s:    int(p.K8S),
	}
}

func capsFromProto(c *taipb.Capabilities) types.Capabilities {
	if c == nil {
		return types.Capabilities{}
	}
	return types.Capabilities{
		Docker:   c.Docker,
		K8s:      c.K8S,
		HostExec: c.HostExec,
		VNC:      c.Vnc,
	}
}

func systemFromProto(s *taipb.SystemInfo) types.SystemInfo {
	if s == nil {
		return types.SystemInfo{}
	}
	return types.SystemInfo{
		OS:       s.Os,
		Arch:     s.Arch,
		Hostname: s.Hostname,
		Shell:    s.Shell,
	}
}
