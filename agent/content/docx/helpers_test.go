package docx_test

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

func newTestContext() *agentContext.Context {
	authorized := &oauthTypes.AuthorizedInfo{
		Subject:  "test-user",
		ClientID: "test-client-id",
		UserID:   "test-user-123",
	}
	ctx := agentContext.New(stdContext.Background(), authorized, "test-chat")
	ctx.AssistantID = "test-assistant"
	ctx.Locale = "en-us"
	ctx.IDGenerator = message.NewIDGenerator()
	return ctx
}

func newTestOptions() *contentTypes.Options {
	return &contentTypes.Options{
		Capabilities: &openai.Capabilities{},
	}
}

func getTestFilePath(filename string) string {
	yaoRoot := os.Getenv("YAO_TEST_APPLICATION")
	if yaoRoot == "" {
		yaoRoot = os.Getenv("YAO_ROOT")
	}
	return filepath.Join(yaoRoot, testFilesDir, filename)
}
