package sandbox

import (
	"context"
	"sync"
	"time"

	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/grpc/pb"
)

// HeartbeatData holds the latest heartbeat from a sandbox container.
type HeartbeatData struct {
	SandboxID    string
	CPUPercent   int32
	MemBytes     int64
	RunningProcs int32
	LastSeen     time.Time
}

// Handler implements sandbox-related gRPC methods.
type Handler struct {
	mu         sync.RWMutex
	heartbeats map[string]*HeartbeatData
	onBeat     func(data *HeartbeatData) string // optional callback; returns action
}

// NewHandler creates a Handler. onBeat is called on each heartbeat and
// may return "ok" or "shutdown" to signal the container.
func NewHandler(onBeat func(data *HeartbeatData) string) *Handler {
	return &Handler{
		heartbeats: make(map[string]*HeartbeatData),
		onBeat:     onBeat,
	}
}

// Heartbeat handles the Heartbeat RPC from sandbox containers.
func (h *Handler) Heartbeat(_ context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	data := &HeartbeatData{
		SandboxID:    req.SandboxId,
		CPUPercent:   req.CpuPercent,
		MemBytes:     req.MemBytes,
		RunningProcs: req.RunningProcs,
		LastSeen:     time.Now(),
	}

	h.mu.Lock()
	h.heartbeats[req.SandboxId] = data
	h.mu.Unlock()

	action := "ok"
	if h.onBeat != nil {
		if a := h.onBeat(data); a != "" {
			action = a
		}
	}

	log.Trace("sandbox heartbeat: id=%s cpu=%d%% mem=%d procs=%d → %s",
		req.SandboxId, req.CpuPercent, req.MemBytes, req.RunningProcs, action)

	return &pb.HeartbeatResponse{Action: action}, nil
}

// LastHeartbeat returns the most recent heartbeat for a sandbox, or nil.
func (h *Handler) LastHeartbeat(sandboxID string) *HeartbeatData {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.heartbeats[sandboxID]
}

// RemoveHeartbeat cleans up heartbeat data for a removed sandbox.
func (h *Handler) RemoveHeartbeat(sandboxID string) {
	h.mu.Lock()
	delete(h.heartbeats, sandboxID)
	h.mu.Unlock()
}
