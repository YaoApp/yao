package trace_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/tests/testutils"
	"github.com/yaoapp/yao/trace"
	"github.com/yaoapp/yao/trace/types"
)

// testTraceData holds the prepared test trace and related information
type testTraceData struct {
	TraceID    string
	Manager    types.Manager
	RootNodeID string
	Node1ID    string
	Node2ID    string
	Node3ID    string
	TokenInfo  *testutils.TokenInfo
	TestClient *oauthtypes.ClientInfo
	ServerURL  string
	BaseURL    string
	Ctx        context.Context
}

// prepareTestTrace creates a test trace with sample nodes, logs, and spaces
// This provides consistent test data for all trace API tests
func prepareTestTrace(t *testing.T) *testTraceData {
	serverURL := testutils.Prepare(t)

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and obtain token with trace permissions
	testClient := testutils.RegisterTestClient(t, "Trace API Test Client", []string{"https://localhost/callback"})
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, testClient.ClientID, testClient.ClientSecret, "https://localhost/callback", "openid profile trace:traces:read:all")

	// Create a test trace with proper user info
	ctx := context.Background()
	traceOption := &types.TraceOption{
		CreatedBy: tokenInfo.UserID,
		Metadata: map[string]any{
			"test_type": "api_test",
			"test_name": "common_trace_data",
		},
	}

	traceID, manager, err := trace.New(ctx, trace.Local, traceOption)
	assert.NoError(t, err)
	assert.NotEmpty(t, traceID)

	// Add manager-level logs
	manager.Info("Manager info log", map[string]any{"level": "manager", "action": "init"})
	manager.Debug("Manager debug log", map[string]any{"level": "manager", "action": "debug"})

	// Create a memory space
	space, err := manager.CreateSpace(types.TraceSpaceOption{
		Label:       "Test Space",
		Type:        "memory",
		Icon:        "database",
		Description: "A test memory space",
	})
	assert.NoError(t, err)
	assert.NotNil(t, space)
	spaceID := space.ID

	// Add some data to the space
	err = manager.SetSpaceValue(spaceID, "key1", "value1")
	assert.NoError(t, err)
	err = manager.SetSpaceValue(spaceID, "key2", map[string]any{"nested": "data"})
	assert.NoError(t, err)

	// Add first node
	node1, err := manager.Add("test input 1", types.TraceNodeOption{
		Label:       "First Node",
		Type:        "agent",
		Icon:        "icon1",
		Description: "First test node",
		Metadata:    map[string]any{"node_order": 1},
	})
	assert.NoError(t, err)
	node1ID := node1.ID()

	node1.Info("Node 1 info log", map[string]any{"node": "1", "message": "info"})
	node1.Debug("Node 1 debug log", map[string]any{"node": "1", "message": "debug"})

	err = node1.SetOutput(map[string]any{"result": "node1_output", "status": "processing"})
	assert.NoError(t, err)

	// Add second node
	node2, err := manager.Add("test input 2", types.TraceNodeOption{
		Label:       "Second Node",
		Type:        "tool",
		Icon:        "icon2",
		Description: "Second test node",
		Metadata:    map[string]any{"node_order": 2},
	})
	assert.NoError(t, err)
	node2ID := node2.ID()

	node2.Info("Node 2 info log", map[string]any{"node": "2", "message": "info"})
	node2.Warn("Node 2 warn log", map[string]any{"node": "2", "message": "warning"})

	err = node2.Complete(map[string]any{"result": "node2_completed", "status": "success"})
	assert.NoError(t, err)

	// Add third node
	node3, err := manager.Add("test input 3", types.TraceNodeOption{
		Label:       "Third Node",
		Type:        "custom",
		Icon:        "icon3",
		Description: "Third test node",
		Metadata:    map[string]any{"node_order": 3},
	})
	assert.NoError(t, err)
	node3ID := node3.ID()

	node3.Debug("Node 3 debug log", map[string]any{"node": "3", "message": "debug"})
	node3.Error("Node 3 error log", map[string]any{"node": "3", "message": "error", "error_code": 500})

	err = node3.Complete(map[string]any{"result": "node3_completed"})
	assert.NoError(t, err)

	// Complete the trace to flush all data to storage
	err = manager.MarkComplete()
	assert.NoError(t, err)

	// Get root node ID
	rootNode, err := manager.GetRootNode()
	assert.NoError(t, err)
	rootNodeID := ""
	if rootNode != nil {
		rootNodeID = rootNode.ID
	}

	return &testTraceData{
		TraceID:    traceID,
		Manager:    manager,
		RootNodeID: rootNodeID,
		Node1ID:    node1ID,
		Node2ID:    node2ID,
		Node3ID:    node3ID,
		TokenInfo:  tokenInfo,
		TestClient: testClient,
		ServerURL:  serverURL,
		BaseURL:    baseURL,
		Ctx:        ctx,
	}
}

// cleanupTestTrace cleans up the test trace and related resources
func cleanupTestTrace(t *testing.T, data *testTraceData) {
	if data.TraceID != "" {
		trace.Release(data.TraceID)
		trace.Remove(data.Ctx, trace.Local, data.TraceID)
	}
	if data.TestClient != nil {
		testutils.CleanupTestClient(t, data.TestClient.ClientID)
	}
	testutils.Clean()
}
