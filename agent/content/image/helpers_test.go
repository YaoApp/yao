package image_test

import (
	stdContext "context"

	"github.com/yaoapp/gou/connector/openai"
	contentTypes "github.com/yaoapp/yao/agent/content/types"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
)

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

func createTestPNG() []byte {
	return []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
		0x00, 0x03, 0x01, 0x01, 0x00, 0x18, 0xDD, 0x8D,
		0xB4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E,
		0x44, 0xAE, 0x42, 0x60, 0x82,
	}
}
