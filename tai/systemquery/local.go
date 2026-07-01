package systemquery

import (
	"context"

	"google.golang.org/grpc"

	"github.com/yaoapp/yao/tai/systemquery/pb"
)

// LocalClient implements pb.SystemQueryClient using an in-process LocalCollector.
// Used in Host mode where no remote gRPC connection exists.
type LocalClient struct {
	collector *LocalCollector
}

var _ pb.SystemQueryClient = (*LocalClient)(nil)

// NewLocalClient creates a LocalClient backed by a LocalCollector.
func NewLocalClient() *LocalClient {
	return &LocalClient{collector: NewLocalCollector()}
}

// ListPorts implements pb.SystemQueryClient.
func (c *LocalClient) ListPorts(ctx context.Context, _ *pb.ListPortsRequest, _ ...grpc.CallOption) (*pb.ListPortsResponse, error) {
	ports, err := c.collector.ListPorts(ctx)
	if err != nil {
		return nil, err
	}
	return &pb.ListPortsResponse{Ports: ports}, nil
}

// ListProcesses implements pb.SystemQueryClient.
func (c *LocalClient) ListProcesses(ctx context.Context, req *pb.ListProcessesRequest, _ ...grpc.CallOption) (*pb.ListProcessesResponse, error) {
	procs, load, err := c.collector.ListProcesses(ctx, req.GetSkipCpuSample())
	if err != nil {
		return nil, err
	}
	return &pb.ListProcessesResponse{Processes: procs, Load: load}, nil
}
