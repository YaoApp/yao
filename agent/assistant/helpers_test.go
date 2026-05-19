package assistant_test

import (
	stdContext "context"
	"strings"

	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
)

func newTestContext(chatID, assistantID string) *agentContext.Context {
	authorized := &oauthTypes.AuthorizedInfo{
		Subject:    "test-user",
		ClientID:   "test-client-id",
		Scope:      "openid profile email",
		SessionID:  "test-session-id",
		UserID:     "test-user-123",
		TeamID:     "test-team-456",
		TenantID:   "test-tenant-789",
		RememberMe: true,
		Constraints: oauthTypes.DataConstraints{
			OwnerOnly:   false,
			CreatorOnly: false,
			EditorOnly:  false,
			TeamOnly:    true,
			Extra: map[string]interface{}{
				"department": "engineering",
				"region":     "us-west",
				"project":    "yao",
			},
		},
	}

	ctx := agentContext.New(stdContext.Background(), authorized, chatID)
	ctx.AssistantID = assistantID
	ctx.Locale = "en-us"
	ctx.Theme = "light"
	ctx.Client = agentContext.Client{
		Type:      "web",
		UserAgent: "TestAgent/1.0",
		IP:        "127.0.0.1",
	}
	ctx.Referer = agentContext.RefererAPI
	ctx.Accept = agentContext.AcceptWebCUI
	ctx.Route = ""
	ctx.IDGenerator = message.NewIDGenerator()
	ctx.Metadata = make(map[string]interface{})
	return ctx
}

func containsString(content interface{}, substr string) bool {
	switch v := content.(type) {
	case string:
		return strings.Contains(v, substr)
	default:
		return false
	}
}
