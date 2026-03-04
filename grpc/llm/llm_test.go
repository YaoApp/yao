package llm_test

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

func TestChatCompletions_InvalidConnector(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:llm")
	ctx := testutils.WithToken(context.Background(), token)

	msgs, _ := json.Marshal([]map[string]interface{}{
		{"role": "user", "content": "hello"},
	})

	_, err := client.ChatCompletions(ctx, &pb.ChatRequest{
		Connector: "nonexistent-connector",
		Messages:  msgs,
	})
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestChatCompletions_EmptyConnector(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:llm")
	ctx := testutils.WithToken(context.Background(), token)

	_, err := client.ChatCompletions(ctx, &pb.ChatRequest{
		Connector: "",
	})
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestChatCompletionsStream_EmptyConnector(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:llm")
	ctx := testutils.WithToken(context.Background(), token)

	stream, err := client.ChatCompletionsStream(ctx, &pb.ChatRequest{
		Connector: "",
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

func TestChatCompletions_BadMessagesJSON(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:llm")
	ctx := testutils.WithToken(context.Background(), token)

	_, err := client.ChatCompletions(ctx, &pb.ChatRequest{
		Connector: "openai",
		Messages:  []byte("{bad-json"),
	})
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestChatCompletions_EmptyMessages(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:llm")
	ctx := testutils.WithToken(context.Background(), token)

	_, err := client.ChatCompletions(ctx, &pb.ChatRequest{
		Connector: "openai",
		Messages:  nil,
	})
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestChatCompletions_EmptyMessageArray(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:llm")
	ctx := testutils.WithToken(context.Background(), token)

	_, err := client.ChatCompletions(ctx, &pb.ChatRequest{
		Connector: "openai",
		Messages:  []byte("[]"),
	})
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestChatCompletions_BadOptionsJSON(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:llm")
	ctx := testutils.WithToken(context.Background(), token)

	msgs, _ := json.Marshal([]map[string]interface{}{
		{"role": "user", "content": "hello"},
	})
	_, err := client.ChatCompletions(ctx, &pb.ChatRequest{
		Connector: "openai",
		Messages:  msgs,
		Options:   []byte("{bad-options"),
	})
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestChatCompletionsStream_InvalidConnector(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:llm")
	ctx := testutils.WithToken(context.Background(), token)

	msgs, _ := json.Marshal([]map[string]interface{}{
		{"role": "user", "content": "hello"},
	})

	stream, err := client.ChatCompletionsStream(ctx, &pb.ChatRequest{
		Connector: "nonexistent-connector",
		Messages:  msgs,
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

func TestChatCompletionsStream_BadMessages(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:llm")
	ctx := testutils.WithToken(context.Background(), token)

	stream, err := client.ChatCompletionsStream(ctx, &pb.ChatRequest{
		Connector: "openai",
		Messages:  []byte("{bad-json"),
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

// TestChatCompletions_RealLLM tests against a real LLM if OPENAI_TEST_KEY is set.
func TestChatCompletions_RealLLM(t *testing.T) {
	if os.Getenv("OPENAI_TEST_KEY") == "" {
		t.Skip("OPENAI_TEST_KEY not set, skipping real LLM test")
	}

	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:llm")
	ctx := testutils.WithToken(context.Background(), token)

	msgs, _ := json.Marshal([]map[string]interface{}{
		{"role": "user", "content": "Say hello in one word."},
	})

	resp, err := client.ChatCompletions(ctx, &pb.ChatRequest{
		Connector: "gpt-4o-mini",
		Messages:  msgs,
	})
	assert.NoError(t, err)
	if assert.NotNil(t, resp) {
		assert.NotEmpty(t, resp.Data)
	}
}

// TestChatCompletionsStream_RealLLM tests streaming against a real LLM if OPENAI_TEST_KEY is set.
func TestChatCompletionsStream_RealLLM(t *testing.T) {
	if os.Getenv("OPENAI_TEST_KEY") == "" {
		t.Skip("OPENAI_TEST_KEY not set, skipping real LLM stream test")
	}

	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:llm")
	ctx := testutils.WithToken(context.Background(), token)

	msgs, _ := json.Marshal([]map[string]interface{}{
		{"role": "user", "content": "Count from 1 to 3."},
	})

	stream, err := client.ChatCompletionsStream(ctx, &pb.ChatRequest{
		Connector: "gpt-4o-mini",
		Messages:  msgs,
	})
	assert.NoError(t, err)

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
