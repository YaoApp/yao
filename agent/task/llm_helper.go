package task

import (
	"strings"

	"github.com/yaoapp/gou/process"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
)

// toOAuthInfo converts process.AuthorizedInfo to oauthTypes.AuthorizedInfo.
// Required because agentcontext.New() accepts *oauthTypes.AuthorizedInfo.
func toOAuthInfo(auth *process.AuthorizedInfo) *oauthTypes.AuthorizedInfo {
	if auth == nil {
		return nil
	}
	return &oauthTypes.AuthorizedInfo{
		Subject:    auth.Subject,
		ClientID:   auth.ClientID,
		Scope:      auth.Scope,
		SessionID:  auth.SessionID,
		UserID:     auth.UserID,
		TeamID:     auth.TeamID,
		TenantID:   auth.TenantID,
		RememberMe: auth.RememberMe,
	}
}

// ExtractFirstUserMessage extracts the content of the first user-role message.
// Exported for cross-package access from API layer.
func ExtractFirstUserMessage(msgs []InputMessage) string {
	for _, m := range msgs {
		if m.Role == "user" && m.Content != "" {
			return m.Content
		}
	}
	return ""
}

// cleanMarkdownFences removes markdown code block wrappers from LLM output
func cleanMarkdownFences(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

// isValidPriority checks if a string is a valid task priority value
func isValidPriority(s string) bool {
	switch s {
	case "high", "medium", "low", "none":
		return true
	}
	return false
}

// isValidMailPriority checks if a string is a valid mail priority value
func isValidMailPriority(s string) bool {
	switch s {
	case "high", "medium", "low":
		return true
	}
	return false
}
