package eval

import (
	stdContext "context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// NewTestContext creates a new context for testing
// This is similar to newAgentNextTestContext in agent_next_test.go
// but configurable via Environment
func NewTestContext(chatID, assistantID string, env *Environment) *context.Context {
	// Build authorized info from environment
	authorized := buildAuthorizedInfo(env)

	// Create context with standard initialization
	ctx := context.New(stdContext.Background(), authorized, chatID)
	ctx.ID = chatID
	ctx.AssistantID = assistantID
	ctx.Locale = env.Locale
	ctx.Client = context.Client{
		Type:      env.ClientType,
		UserAgent: "yao-agent-test/1.0",
		IP:        env.ClientIP,
	}
	ctx.Referer = env.Referer
	ctx.Accept = context.AcceptStandard
	ctx.IDGenerator = message.NewIDGenerator()
	ctx.Metadata = make(map[string]interface{})

	// Apply metadata from context config if available
	if env.ContextConfig != nil && env.ContextConfig.Metadata != nil {
		for k, v := range env.ContextConfig.Metadata {
			ctx.Metadata[k] = v
		}
	}

	// Initialize interrupt controller
	ctx.Interrupt = context.NewInterruptController()

	// Close the default logger created by context.New() and use noop logger
	// to suppress LLM debug output during tests
	if ctx.Logger != nil {
		ctx.Logger.Close()
	}
	ctx.Logger = context.Noop()

	return ctx
}

// buildAuthorizedInfo builds AuthorizedInfo from Environment
func buildAuthorizedInfo(env *Environment) *types.AuthorizedInfo {
	authorized := &types.AuthorizedInfo{
		Subject:  env.UserID,
		UserID:   env.UserID,
		TeamID:   env.TeamID,
		TenantID: env.TeamID,
	}

	// Apply custom authorized config if available
	if env.ContextConfig != nil && env.ContextConfig.Authorized != nil {
		authCfg := env.ContextConfig.Authorized

		if authCfg.Sub != "" {
			authorized.Subject = authCfg.Sub
		}
		if authCfg.ClientID != "" {
			authorized.ClientID = authCfg.ClientID
		}
		if authCfg.Scope != "" {
			authorized.Scope = authCfg.Scope
		}
		if authCfg.SessionID != "" {
			authorized.SessionID = authCfg.SessionID
		}
		if authCfg.UserID != "" {
			authorized.UserID = authCfg.UserID
		}
		if authCfg.TeamID != "" {
			authorized.TeamID = authCfg.TeamID
		}
		if authCfg.TenantID != "" {
			authorized.TenantID = authCfg.TenantID
		}
		authorized.RememberMe = authCfg.RememberMe

		// Apply constraints
		if authCfg.Constraints != nil {
			authorized.Constraints = types.DataConstraints{
				OwnerOnly:   authCfg.Constraints.OwnerOnly,
				CreatorOnly: authCfg.Constraints.CreatorOnly,
				EditorOnly:  authCfg.Constraints.EditorOnly,
				TeamOnly:    authCfg.Constraints.TeamOnly,
				Extra:       authCfg.Constraints.Extra,
			}
		}
	}

	return authorized
}

// NewTestContextFromOptions creates a test context from test options and test case
func NewTestContextFromOptions(chatID, assistantID string, opts *Options, tc *Case) *context.Context {
	// Get environment from test case (with options override)
	env := tc.GetEnvironment(opts)
	return NewTestContext(chatID, assistantID, env)
}

// GenerateChatID generates a unique chat ID for testing.
// Priority: ctx-specified ChatID > auto-generated unique ID.
func GenerateChatID(testID string, runNumber int, opts *Options) string {
	if opts != nil && opts.ContextData != nil && opts.ContextData.ChatID != "" {
		if runNumber > 1 {
			return fmt.Sprintf("%s-run%d", opts.ContextData.ChatID, runNumber)
		}
		return opts.ContextData.ChatID
	}
	suffix := fmt.Sprintf("%d-%s", time.Now().UnixNano(), randomHex(4))
	if runNumber > 1 {
		return fmt.Sprintf("test-%s-run%d-%s", testID, runNumber, suffix)
	}
	return fmt.Sprintf("test-%s-%s", testID, suffix)
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
