package run_test

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	agenttest "github.com/yaoapp/yao/agent/test"
	"github.com/yaoapp/yao/grpc/pb"
	"github.com/yaoapp/yao/grpc/tests/testutils"
)

// ── Layer 1: Basic RPC validation (no LLM, no real agent) ─────────────────

func TestStream_EmptyProcess(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:stream")
	ctx := testutils.WithToken(context.Background(), token)

	stream, err := client.Stream(ctx, &pb.RunRequest{Process: ""})
	if err != nil {
		st, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		return
	}
	_, err = stream.Recv()
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestStream_UnknownProcess(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:stream")
	ctx := testutils.WithToken(context.Background(), token)

	stream, err := client.Stream(ctx, &pb.RunRequest{Process: "nonexistent.stream"})
	if err != nil {
		st, _ := status.FromError(err)
		assert.Equal(t, codes.Unimplemented, st.Code())
		return
	}
	_, err = stream.Recv()
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unimplemented, st.Code())
}

func TestStream_NoToken(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)

	stream, err := client.Stream(context.Background(), &pb.RunRequest{
		Process: "agent.test.Run",
	})
	if err != nil {
		st, _ := status.FromError(err)
		assert.Equal(t, codes.Unauthenticated, st.Code())
		return
	}
	_, err = stream.Recv()
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestStream_WrongScope(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:run")
	ctx := testutils.WithToken(context.Background(), token)

	stream, err := client.Stream(ctx, &pb.RunRequest{
		Process: "agent.test.Run",
	})
	if err != nil {
		st, _ := status.FromError(err)
		assert.Equal(t, codes.PermissionDenied, st.Code())
		return
	}
	_, err = stream.Recv()
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.PermissionDenied, st.Code())
}

func TestStream_InvalidArgsJSON(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:stream")
	ctx := testutils.WithToken(context.Background(), token)

	stream, err := client.Stream(ctx, &pb.RunRequest{
		Process: "agent.test.Run",
		Args:    []byte("{bad-json"),
	})
	if err != nil {
		st, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		return
	}
	_, err = stream.Recv()
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestStream_RunnerError(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	opts := map[string]interface{}{
		"agent_id":   "tests.simple-greeting",
		"input":      "/nonexistent/path/cases.jsonl",
		"input_mode": "file",
	}
	optsJSON, _ := json.Marshal(opts)

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:stream")
	ctx := testutils.WithToken(context.Background(), token)

	stream, err := client.Stream(ctx, &pb.RunRequest{
		Process: "agent.test.Run",
		Args:    optsJSON,
	})
	assert.NoError(t, err)

	// Runner error is now embedded in the Report via the Done chunk,
	// not as a gRPC status error. The stream closes cleanly.
	var report agenttest.Report
	for {
		chunk, err := stream.Recv()
		if err != nil {
			assert.ErrorIs(t, err, io.EOF)
			break
		}
		if chunk.Done {
			json.Unmarshal(chunk.Data, &report)
			break
		}
	}
	assert.NotEmpty(t, report.Error, "report.Error should contain runner error message")
	assert.True(t, report.HasFailures(), "report should indicate failure")
}

// ── Layer 2: DryRun mode (needs agent resolver, no LLM) ──────────────────

func TestStream_DryRun_TextMode(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	inputFile := writeTempJSONL(t)

	opts := map[string]interface{}{
		"agent_id":   "tests.simple-greeting",
		"input":      inputFile,
		"input_mode": "file",
		"dry_run":    true,
	}
	optsJSON, err := json.Marshal(opts)
	if !assert.NoError(t, err) {
		return
	}

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:stream")
	ctx := testutils.WithToken(context.Background(), token)

	stream, err := client.Stream(ctx, &pb.RunRequest{
		Process: "agent.test.Run",
		Args:    optsJSON,
	})
	if !assert.NoError(t, err) {
		return
	}

	var chunks int
	var lastChunk *pb.Chunk
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if !assert.NoError(t, err) {
			return
		}
		chunks++
		lastChunk = chunk
		if chunk.Done {
			break
		}
		assert.NotEmpty(t, chunk.Data, "non-final chunk should carry data")
	}

	assert.Greater(t, chunks, 0, "should receive at least one chunk")
	if assert.NotNil(t, lastChunk) {
		assert.True(t, lastChunk.Done, "last chunk should have Done=true")

		var report agenttest.Report
		err := json.Unmarshal(lastChunk.Data, &report)
		assert.NoError(t, err, "final chunk should be valid Report JSON")
		if assert.NotNil(t, report.Summary) {
			assert.Equal(t, 2, report.Summary.Total, "report should reflect 2 test cases")
			assert.Equal(t, "tests.simple-greeting", report.Summary.AgentID)
		}
	}
}

func TestStream_DryRun_JSONMode(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	inputFile := writeTempJSONL(t)

	opts := map[string]interface{}{
		"agent_id":    "tests.simple-greeting",
		"input":       inputFile,
		"input_mode":  "file",
		"dry_run":     true,
		"json_output": true,
	}
	optsJSON, err := json.Marshal(opts)
	if !assert.NoError(t, err) {
		return
	}

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:stream")
	ctx := testutils.WithToken(context.Background(), token)

	stream, err := client.Stream(ctx, &pb.RunRequest{
		Process: "agent.test.Run",
		Args:    optsJSON,
	})
	if !assert.NoError(t, err) {
		return
	}

	var chunks int
	var eventChunks int
	var lastChunk *pb.Chunk
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if !assert.NoError(t, err) {
			return
		}
		chunks++
		lastChunk = chunk
		if chunk.Done {
			break
		}
		// Non-final chunks in JSON mode are NDJSON event lines
		var evt map[string]interface{}
		if json.Unmarshal(chunk.Data, &evt) == nil {
			eventChunks++
		}
	}

	assert.Greater(t, chunks, 0, "should receive at least one chunk")
	assert.Greater(t, eventChunks, 0, "should receive NDJSON event chunks")

	if assert.NotNil(t, lastChunk) {
		assert.True(t, lastChunk.Done, "last chunk should have Done=true")

		var report agenttest.Report
		err := json.Unmarshal(lastChunk.Data, &report)
		assert.NoError(t, err, "final chunk should be valid Report JSON")
		if assert.NotNil(t, report.Summary) {
			assert.Equal(t, 2, report.Summary.Total)
			assert.Equal(t, "tests.simple-greeting", report.Summary.AgentID)
		}
	}
}

// ── Layer 2b: A2A DryRun (agent-caller resolves, no LLM) ─────────────────

func TestStream_DryRun_A2A_TextMode(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	inputFile := writeTempA2AJSONL(t)

	opts := map[string]interface{}{
		"agent_id":   "tests.agent-caller",
		"input":      inputFile,
		"input_mode": "file",
		"dry_run":    true,
	}
	optsJSON, _ := json.Marshal(opts)

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:stream")
	ctx := testutils.WithToken(context.Background(), token)

	stream, err := client.Stream(ctx, &pb.RunRequest{
		Process: "agent.test.Run",
		Args:    optsJSON,
	})
	if !assert.NoError(t, err) {
		return
	}

	var chunks int
	var lastChunk *pb.Chunk
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if !assert.NoError(t, err) {
			return
		}
		chunks++
		lastChunk = chunk
		if chunk.Done {
			break
		}
	}

	assert.Greater(t, chunks, 0)
	if assert.NotNil(t, lastChunk) && assert.True(t, lastChunk.Done) {
		var report agenttest.Report
		err := json.Unmarshal(lastChunk.Data, &report)
		assert.NoError(t, err)
		if assert.NotNil(t, report.Summary) {
			assert.Equal(t, 3, report.Summary.Total)
			assert.Equal(t, "tests.agent-caller", report.Summary.AgentID)
		}
	}
}

func TestStream_DryRun_A2A_JSONMode(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	inputFile := writeTempA2AJSONL(t)

	opts := map[string]interface{}{
		"agent_id":    "tests.agent-caller",
		"input":       inputFile,
		"input_mode":  "file",
		"dry_run":     true,
		"json_output": true,
	}
	optsJSON, _ := json.Marshal(opts)

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:stream")
	ctx := testutils.WithToken(context.Background(), token)

	stream, err := client.Stream(ctx, &pb.RunRequest{
		Process: "agent.test.Run",
		Args:    optsJSON,
	})
	if !assert.NoError(t, err) {
		return
	}

	var chunks int
	var dryRunEvents int
	var lastChunk *pb.Chunk
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if !assert.NoError(t, err) {
			return
		}
		chunks++
		lastChunk = chunk
		if chunk.Done {
			break
		}
		var evt map[string]interface{}
		if json.Unmarshal(chunk.Data, &evt) == nil {
			if evt["type"] == "dry_run_case" {
				dryRunEvents++
			}
		}
	}

	assert.Greater(t, chunks, 0)
	assert.Equal(t, 3, dryRunEvents, "should emit one dry_run_case event per test case")

	if assert.NotNil(t, lastChunk) && assert.True(t, lastChunk.Done) {
		var report agenttest.Report
		err := json.Unmarshal(lastChunk.Data, &report)
		assert.NoError(t, err)
		if assert.NotNil(t, report.Summary) {
			assert.Equal(t, 3, report.Summary.Total)
			assert.Equal(t, "tests.agent-caller", report.Summary.AgentID)
		}
	}
}

// ── Layer 3 (optional): Real LLM integration ─────────────────────────────

func TestStream_RealAgent(t *testing.T) {
	if os.Getenv("OPENAI_TEST_KEY") == "" {
		t.Skip("OPENAI_TEST_KEY not set, skipping real agent stream test")
	}

	conn := testutils.Prepare(t)
	defer testutils.Clean()

	inputFile := writeTempJSONL(t)

	opts := map[string]interface{}{
		"agent_id":   "tests.simple-greeting",
		"input":      inputFile,
		"input_mode": "file",
	}
	optsJSON, _ := json.Marshal(opts)

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:stream")
	ctx := testutils.WithToken(context.Background(), token)

	stream, err := client.Stream(ctx, &pb.RunRequest{
		Process: "agent.test.Run",
		Args:    optsJSON,
	})
	if !assert.NoError(t, err) {
		return
	}

	var chunks int
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if !assert.NoError(t, err) {
			break
		}
		chunks++
		if chunk.Done {
			break
		}
		assert.NotEmpty(t, chunk.Data)
	}
	assert.Greater(t, chunks, 0)
}

func TestStream_RealAgent_A2A(t *testing.T) {
	if os.Getenv("OPENAI_TEST_KEY") == "" {
		t.Skip("OPENAI_TEST_KEY not set, skipping real A2A agent stream test")
	}

	conn := testutils.Prepare(t)
	defer testutils.Clean()

	inputFile := writeTempA2AJSONL(t)

	opts := map[string]interface{}{
		"agent_id":   "tests.agent-caller",
		"input":      inputFile,
		"input_mode": "file",
	}
	optsJSON, _ := json.Marshal(opts)

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:stream")
	ctx := testutils.WithToken(context.Background(), token)

	stream, err := client.Stream(ctx, &pb.RunRequest{
		Process: "agent.test.Run",
		Args:    optsJSON,
	})
	if !assert.NoError(t, err) {
		return
	}

	var chunks int
	var lastChunk *pb.Chunk
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if !assert.NoError(t, err) {
			break
		}
		chunks++
		lastChunk = chunk
		if chunk.Done {
			break
		}
		assert.NotEmpty(t, chunk.Data)
	}

	assert.Greater(t, chunks, 0)
	if assert.NotNil(t, lastChunk) && assert.True(t, lastChunk.Done) {
		var report agenttest.Report
		if assert.NoError(t, json.Unmarshal(lastChunk.Data, &report)) && assert.NotNil(t, report.Summary) {
			assert.Equal(t, "tests.agent-caller", report.Summary.AgentID)
			assert.Equal(t, 3, report.Summary.Total)
		}
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────

// writeTempJSONL creates a minimal JSONL test file in t.TempDir().
func writeTempJSONL(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "cases.jsonl")
	content := `{"id":"T001","input":"hello"}
{"id":"T002","input":"hi there"}`
	err := os.WriteFile(p, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write temp JSONL: %v", err)
	}
	return p
}

// writeTempA2AJSONL creates a JSONL file with inputs that trigger A2A calls
// via tests.agent-caller's ctx.agent.Call/All/Any hooks.
func writeTempA2AJSONL(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "a2a_cases.jsonl")
	content := `{"id":"A2A-001","input":"call single agent"}
{"id":"A2A-002","input":"call all agents"}
{"id":"A2A-003","input":"call any agent"}`
	err := os.WriteFile(p, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write temp A2A JSONL: %v", err)
	}
	return p
}
