package secret

import (
	"fmt"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/setting"
	"google.golang.org/grpc/metadata"
)

// PredefinedMeta holds label/description from the assistant's sandbox.yao DSL.
type PredefinedMeta struct {
	Label       string
	Description string
}

// LoadPredefinedFn is set during app init to provide predefined secrets
// metadata without circular imports. See openapi/setting/agent.go.
var LoadPredefinedFn func(assistantID string) map[string]PredefinedMeta

func resolveNamespace(assistantID string) string {
	return "agent." + assistantID
}

func taskNamespace(chatID string) string {
	return "task-config.task." + chatID
}

// extractAssistantID extracts the assistant ID from gRPC metadata.
// MCP tool calls arrive via gRPC, so the assistant ID is carried in
// the "x-assistant-id" metadata header (set by the runner's env
// CTX_ASSISTANT_ID -> tai gRPC client).
// Returns an error if the assistant ID cannot be determined.
func extractAssistantID(proc *process.Process) (string, error) {
	if proc.Context != nil {
		if md, ok := metadata.FromIncomingContext(proc.Context); ok {
			if ids := md.Get("x-assistant-id"); len(ids) > 0 && ids[0] != "" {
				return ids[0], nil
			}
		}
	}
	return "", fmt.Errorf("assistant_id not found in request context")
}

// extractChatID extracts the chat ID from gRPC metadata.
// Returns empty string (no error) if not present — chatID is optional.
func extractChatID(proc *process.Process) string {
	if proc.Context != nil {
		if md, ok := metadata.FromIncomingContext(proc.Context); ok {
			if ids := md.Get("x-chat-id"); len(ids) > 0 && ids[0] != "" {
				return ids[0]
			}
		}
	}
	return ""
}

// getMergedSecrets retrieves secrets from L2 (agent namespace) and optionally
// L3 (task-config namespace). L3 secrets override L2 secrets by key.
func getMergedSecrets(userID, teamID, assistantID, chatID string) (map[string]interface{}, error) {
	if setting.Global == nil {
		return nil, fmt.Errorf("setting registry not initialized")
	}

	ns := resolveNamespace(assistantID)
	merged, err := setting.Global.GetMerged(userID, teamID, ns)
	if err != nil {
		return nil, err
	}

	secretsMap := extractSecretsMap(merged)

	// L3: task-level secrets override
	if chatID != "" && userID != "" {
		taskNs := taskNamespace(chatID)
		taskData, _ := setting.Global.GetMerged(userID, teamID, taskNs)
		taskSecrets := extractSecretsMap(taskData)
		for k, v := range taskSecrets {
			if secretsMap == nil {
				secretsMap = make(map[string]interface{})
			}
			secretsMap[k] = v
		}
	}

	return secretsMap, nil
}

// extractSecretsMap pulls the "secrets" sub-map from a merged setting payload.
func extractSecretsMap(merged map[string]interface{}) map[string]interface{} {
	if merged == nil {
		return nil
	}
	secretsRaw, ok := merged["secrets"]
	if !ok {
		return nil
	}
	secretsMap, ok := secretsRaw.(map[string]interface{})
	if !ok {
		return nil
	}
	return secretsMap
}

// loadPredefinedSecrets calls LoadPredefinedFn if set.
func loadPredefinedSecrets(assistantID string) map[string]PredefinedMeta {
	if LoadPredefinedFn == nil || assistantID == "" {
		return nil
	}
	return LoadPredefinedFn(assistantID)
}
