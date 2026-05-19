//go:build e2e

package standard_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
)

func TestAgentCallerCallStream(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test (requires LLM)")
	}

	testutils.PrepareAgent(t)
	testutils.RequireE2EKeys(t)
	defer testutils.Clean(t)

	caller := standard.NewAgentCaller()
	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("streams text chunks and returns result", func(t *testing.T) {
		var mu sync.Mutex
		var chunks []string

		streamFn := func(chunk *standard.StreamChunk) int {
			mu.Lock()
			defer mu.Unlock()
			if chunk.Type == "text" && chunk.Delta {
				chunks = append(chunks, chunk.Content)
			}
			return 0
		}

		result, err := caller.CallWithMessagesStream(ctx, "tests.robot-single", "Hello, test message", streamFn)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, result.IsEmpty())

		mu.Lock()
		combined := strings.Join(chunks, "")
		chunkCount := len(chunks)
		mu.Unlock()

		t.Logf("Received %d text chunks, total length: %d", chunkCount, len(combined))
		assert.Greater(t, chunkCount, 0, "should have received at least one text chunk")
		assert.NotEmpty(t, combined, "combined chunks should not be empty")
	})

	t.Run("nil callback works like non-stream call", func(t *testing.T) {
		result, err := caller.CallStream(ctx, "tests.robot-single",
			[]agentcontext.Message{{Role: "user", Content: "Hello"}},
			nil,
		)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, result.IsEmpty())
	})

	t.Run("stream returns parseable JSON", func(t *testing.T) {
		var mu sync.Mutex
		var chunks []string

		streamFn := func(chunk *standard.StreamChunk) int {
			mu.Lock()
			defer mu.Unlock()
			if chunk.Type == "text" && chunk.Delta {
				chunks = append(chunks, chunk.Content)
			}
			return 0
		}

		result, err := caller.CallWithMessagesStream(ctx, "tests.robot-single", "Generate inspiration report", streamFn)

		require.NoError(t, err)
		require.NotNil(t, result)

		data, err := result.GetJSON()
		require.NoError(t, err)
		assert.NotNil(t, data)
		assert.Contains(t, data, "type")

		mu.Lock()
		chunkCount := len(chunks)
		mu.Unlock()
		t.Logf("Received %d chunks for JSON response", chunkCount)
	})

	t.Run("assistant not found returns error", func(t *testing.T) {
		result, err := caller.CallWithMessagesStream(ctx, "non.existent", "hello", nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "assistant not found")
	})
}
