package sandboxv2_test

import (
	"context"
	"os"
	"testing"

	agentContext "github.com/yaoapp/yao/agent/context"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
	"github.com/yaoapp/yao/unit-test/agent/testprepare/sandboxtest"
)

func makeAgentCtx(teamID, userID, chatID, assistantID string, metadata map[string]any) *agentContext.Context {
	var auth *oauthTypes.AuthorizedInfo
	if teamID != "" || userID != "" {
		auth = &oauthTypes.AuthorizedInfo{TeamID: teamID, UserID: userID}
	}
	return &agentContext.Context{
		Context:     context.Background(),
		Authorized:  auth,
		ChatID:      chatID,
		AssistantID: assistantID,
		Metadata:    metadata,
	}
}

func TestMain(m *testing.M) {
	testprepare.MustLoadEnv()
	sandboxtest.PurgeStaleContainers("sb-prep-", "sb-lc-")
	code := m.Run()
	testprepare.Cleanup()
	os.Exit(code)
}
