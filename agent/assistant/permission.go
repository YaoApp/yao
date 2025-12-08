package assistant

import (
	"fmt"

	"github.com/yaoapp/yao/agent/context"
)

func (ast *Assistant) checkPermissions(ctx *context.Context) error {
	if ctx.Authorized == nil {
		return fmt.Errorf("authorized information not found")
	}
	return nil
}
