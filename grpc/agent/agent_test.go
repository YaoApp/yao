package agent_test

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/yaoapp/yao/grpc/pb"
	"github.com/yaoapp/yao/grpc/tests/testutils"
)

func TestAgentStream_InvalidAssistant(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:agent")
	ctx := testutils.WithToken(context.Background(), token)

	msgs, _ := json.Marshal([]map[string]interface{}{
		{"role": "user", "content": "hello"},
	})

	stream, err := client.AgentStream(ctx, &pb.AgentRequest{
		AssistantId: "nonexistent-assistant-id",
		Messages:    msgs,
	})
	if err != nil {
		st, _ := status.FromError(err)
		assert.Equal(t, codes.NotFound, st.Code())
		return
	}
	_, err = stream.Recv()
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestAgentStream_EmptyAssistantID(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:agent")
	ctx := testutils.WithToken(context.Background(), token)

	stream, err := client.AgentStream(ctx, &pb.AgentRequest{
		AssistantId: "",
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

func TestAgentStream_EmptyMessages(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:agent")
	ctx := testutils.WithToken(context.Background(), token)

	msgs, _ := json.Marshal([]map[string]interface{}{})

	stream, err := client.AgentStream(ctx, &pb.AgentRequest{
		AssistantId: "some-assistant",
		Messages:    msgs,
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

func TestAgentStream_NilMessages(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:agent")
	ctx := testutils.WithToken(context.Background(), token)

	stream, err := client.AgentStream(ctx, &pb.AgentRequest{
		AssistantId: "some-assistant",
		Messages:    nil,
	})
	if err != nil {
		st, _ := status.FromError(err)
		assert.NotEqual(t, codes.OK, st.Code())
		return
	}
	_, err = stream.Recv()
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.NotEqual(t, codes.OK, st.Code())
}

func TestAgentStream_BadMessagesJSON(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:agent")
	ctx := testutils.WithToken(context.Background(), token)

	stream, err := client.AgentStream(ctx, &pb.AgentRequest{
		AssistantId: "some-assistant",
		Messages:    []byte("{bad-json"),
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

func TestAgentStream_BadOptionsJSON(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:agent")
	ctx := testutils.WithToken(context.Background(), token)

	msgs, _ := json.Marshal([]map[string]interface{}{
		{"role": "user", "content": "hello"},
	})

	stream, err := client.AgentStream(ctx, &pb.AgentRequest{
		AssistantId: "some-assistant",
		Messages:    msgs,
		Options:     []byte("{bad-options"),
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

func TestAgentStream_RealAgent(t *testing.T) {
	if os.Getenv("OPENAI_TEST_KEY") == "" {
		t.Skip("OPENAI_TEST_KEY not set, skipping real agent test")
	}

	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:agent")
	ctx := testutils.WithToken(context.Background(), token)

	msgs, _ := json.Marshal([]map[string]interface{}{
		{"role": "user", "content": "Say hello in one word."},
	})

	stream, err := client.AgentStream(ctx, &pb.AgentRequest{
		AssistantId: "tests.nested.demo",
		Messages:    msgs,
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
