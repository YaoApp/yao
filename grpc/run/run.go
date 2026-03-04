package run

import (
	"context"
	"encoding/json"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/grpc/auth"
	"github.com/yaoapp/yao/grpc/pb"
)

// Handler implements the Run gRPC method.
type Handler struct{}

// Run executes a Yao process by name and returns the JSON-encoded result.
func (h *Handler) Run(ctx context.Context, req *pb.RunRequest) (*pb.RunResponse, error) {
	if req.Process == "" {
		return nil, status.Error(codes.InvalidArgument, "process name is required")
	}

	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(req.Timeout)*time.Second)
		defer cancel()
	}

	var args []interface{}
	if len(req.Args) > 0 {
		if err := json.Unmarshal(req.Args, &args); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid args JSON: %v", err)
		}
	}

	p, err := process.Of(req.Process, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "process error: %v", err)
	}

	p.WithContext(ctx)
	injectAuth(p, ctx)

	if err := p.Execute(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, status.Error(codes.DeadlineExceeded, "process execution timed out")
		}
		return nil, status.Errorf(codes.Internal, "process execution failed: %v", err)
	}
	defer p.Release()

	val := p.Value()
	data, err := json.Marshal(val)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal result: %v", err)
	}

	return &pb.RunResponse{Data: data}, nil
}

// injectAuth propagates AuthorizedInfo from the gRPC context into the Process.
func injectAuth(p *process.Process, ctx context.Context) {
	authInfo := auth.GetAuthorizedInfo(ctx)
	if authInfo == nil {
		return
	}
	p.WithSID(authInfo.SessionID)
	p.WithAuthorized(&process.AuthorizedInfo{
		Subject:   authInfo.Subject,
		ClientID:  authInfo.ClientID,
		Scope:     authInfo.Scope,
		SessionID: authInfo.SessionID,
		UserID:    authInfo.UserID,
		TeamID:    authInfo.TeamID,
		TenantID:  authInfo.TenantID,
	})
}
