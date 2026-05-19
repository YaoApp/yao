package pdf_test

import (
	stdContext "context"
	"os"
	"path/filepath"

	"github.com/yaoapp/gou/connector/openai"
	contentTypes "github.com/yaoapp/yao/agent/content/types"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
)

const testFilesDir = "assistants/tests/attachment-handler/testdata"

func newTestContext(capabilities *openai.Capabilities) *agentContext.Context {
	authorized := &oauthTypes.AuthorizedInfo{
		Subject:  "test-user",
		ClientID: "test-client-id",
		UserID:   "test-user-123",
		TeamID:   "test-team-456",
		TenantID: "test-tenant-789",
	}

	ctx := agentContext.New(stdContext.Background(), authorized, "test-chat")
	ctx.AssistantID = "test-assistant"
	ctx.Locale = "en-us"
	ctx.Theme = "light"
	ctx.Client = agentContext.Client{Type: "web", UserAgent: "TestAgent/1.0", IP: "127.0.0.1"}
	ctx.Referer = agentContext.RefererAPI
	ctx.Accept = agentContext.AcceptWebCUI
	ctx.Metadata = make(map[string]interface{})
	ctx.Capabilities = capabilities
	ctx.IDGenerator = message.NewIDGenerator()
	return ctx
}

func newTestOptions(capabilities *openai.Capabilities, completionOptions *agentContext.CompletionOptions) *contentTypes.Options {
	return &contentTypes.Options{
		Capabilities:      capabilities,
		CompletionOptions: completionOptions,
	}
}

func getTestFilePath(filename string) string {
	yaoRoot := os.Getenv("YAO_TEST_APPLICATION")
	if yaoRoot == "" {
		yaoRoot = os.Getenv("YAO_ROOT")
	}
	return filepath.Join(yaoRoot, testFilesDir, filename)
}
