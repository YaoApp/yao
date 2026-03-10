package sandboxv2

import (
	"github.com/yaoapp/yao/agent/sandbox/v2/claude"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	yaorunner "github.com/yaoapp/yao/agent/sandbox/v2/yao"
)

func init() {
	Register("claude", func() types.Runner { return claude.New() })
	Register("claude/cli", func() types.Runner { return claude.New() })
	Register("yao", func() types.Runner { return yaorunner.New() })
}
