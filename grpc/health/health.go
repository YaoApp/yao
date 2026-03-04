package health

import (
	"context"

	"github.com/yaoapp/yao/grpc/pb"
)

// Handler implements the Healthz RPC.
type Handler struct{}

// Healthz returns server health status. This method is public (no auth required).
func (h *Handler) Healthz(ctx context.Context, req *pb.Empty) (*pb.HealthzResponse, error) {
	return &pb.HealthzResponse{Status: "ok"}, nil
}
