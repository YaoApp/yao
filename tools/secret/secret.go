package secret

import (
	"fmt"

	"github.com/yaoapp/gou/process"
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

// loadPredefinedSecrets calls LoadPredefinedFn if set.
func loadPredefinedSecrets(assistantID string) map[string]PredefinedMeta {
	if LoadPredefinedFn == nil || assistantID == "" {
		return nil
	}
	return LoadPredefinedFn(assistantID)
}
