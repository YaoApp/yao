package tai

import (
	"context"

	grpcclient "github.com/yaoapp/yao/grpc/client"
	"github.com/yaoapp/yao/grpc/pb"
)

// YaoClient wraps grpc/client.Client for backward compatibility.
// New code should use grpc/client.Client directly.
type YaoClient = grpcclient.Client

// NewYaoClientFromEnv reads YAO_GRPC_ADDR and token env vars, dials the
// gRPC server, and returns a connected YaoClient.
func NewYaoClientFromEnv() (*YaoClient, error) {
	return grpcclient.NewFromEnv()
}

// DialYao connects to a Yao gRPC server at addr with the given TokenManager.
func DialYao(addr string, tm *TokenManager) (*YaoClient, error) {
	return grpcclient.Dial(addr, tm)
}

// --- Convenience wrappers kept for sandbox/container code ---

// Run executes a Yao process via the given client.
func Run(ctx context.Context, c *YaoClient, process string, args []byte, timeout int32) ([]byte, error) {
	return c.Run(ctx, process, args, timeout)
}

// Shell executes a system command via the given client.
func Shell(ctx context.Context, c *YaoClient, command string, args []string, env map[string]string, timeout int32) (*pb.ShellResponse, error) {
	return c.Shell(ctx, command, args, env, timeout)
}
