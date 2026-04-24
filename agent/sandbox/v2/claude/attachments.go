package claude

import (
	"context"

	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/sandbox/v2/shared"
	workspace "github.com/yaoapp/yao/tai/workspace"
)

// prepareAttachments resolves __yao.attachment:// URLs in messages,
// copies actual files into the workspace .attachments/{chatID}/ directory via ws.Copy,
// and replaces multimodal content parts with text references.
//
// Delegates to shared.PrepareAttachments; the returned text-replaced messages
// are used directly by the Claude CLI (which reads local files via text refs).
func prepareAttachments(ctx context.Context, messages []agentContext.Message, chatID string, ws workspace.FS) ([]agentContext.Message, error) {
	processed, _, err := shared.PrepareAttachments(ctx, messages, chatID, ws)
	return processed, err
}
