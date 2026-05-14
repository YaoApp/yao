package run

import (
	"encoding/json"
	"io"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/yaoapp/yao/agent/test"
	"github.com/yaoapp/yao/grpc/pb"
)

// Stream dispatches a server-streaming RPC to the appropriate handler
// based on req.Process. Unlike Run (which uses gou process.Of), Stream
// calls domain packages directly so it can inject an io.Writer for
// real-time progress output.
func (h *Handler) Stream(req *pb.RunRequest, stream grpc.ServerStreamingServer[pb.Chunk]) error {
	if req.Process == "" {
		return status.Error(codes.InvalidArgument, "process name is required")
	}

	switch req.Process {
	case "agent.test.Run":
		return h.streamAgentTest(req, stream)
	default:
		return status.Errorf(codes.Unimplemented, "stream not supported for: %s", req.Process)
	}
}

// streamAgentTest runs the full test suite on the server side and streams
// progress output back to the caller. The final chunk (Done=true) carries
// the JSON-encoded *test.Report.
func (h *Handler) streamAgentTest(req *pb.RunRequest, stream grpc.ServerStreamingServer[pb.Chunk]) error {
	var opts test.Options
	if len(req.Args) > 0 {
		if err := json.Unmarshal(req.Args, &opts); err != nil {
			return status.Errorf(codes.InvalidArgument, "invalid options JSON: %v", err)
		}
	}

	if opts.JSONOutput {
		opts.EventWriter = &grpcEventWriter{stream: stream}
		opts.Writer = io.Discard
	} else {
		opts.Writer = &grpcChunkWriter{stream: stream}
	}

	runner := test.NewRunner(&opts)
	report, err := runner.Run()

	if err != nil {
		errReport := &test.Report{
			Summary: &test.Summary{Total: 1, Errors: 1, AgentID: opts.AgentID},
			Error:   err.Error(),
		}
		data, _ := json.Marshal(errReport)
		_ = stream.Send(&pb.Chunk{Data: data, Done: true})
		return nil
	}

	data, _ := json.Marshal(report)
	_ = stream.Send(&pb.Chunk{Data: data, Done: true})
	return nil
}

// grpcChunkWriter implements io.Writer by sending each Write as a gRPC Chunk.
// Used in text mode to stream colored terminal output.
type grpcChunkWriter struct {
	stream grpc.ServerStreamingServer[pb.Chunk]
}

func (w *grpcChunkWriter) Write(p []byte) (int, error) {
	cp := make([]byte, len(p))
	copy(cp, p)
	if err := w.stream.Send(&pb.Chunk{Data: cp}); err != nil {
		return 0, err
	}
	return len(p), nil
}

// grpcEventWriter implements test.EventWriter by sending NDJSON lines
// via gRPC Chunk messages. Used in JSON mode (--json).
type grpcEventWriter struct {
	stream grpc.ServerStreamingServer[pb.Chunk]
}

func (w *grpcEventWriter) WriteEvent(data []byte) error {
	line := make([]byte, len(data)+1)
	copy(line, data)
	line[len(data)] = '\n'
	return w.stream.Send(&pb.Chunk{Data: line})
}
