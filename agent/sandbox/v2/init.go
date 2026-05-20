package sandboxv2

import (
	"github.com/yaoapp/yao/agent/sandbox/v2/claude"
	"github.com/yaoapp/yao/agent/sandbox/v2/opencode"
	tairunner "github.com/yaoapp/yao/agent/sandbox/v2/tai"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	"github.com/yaoapp/yao/agent/sandbox/v2/yaocode"
)

func init() {
	Register("claude", func() types.Runner { return claude.New() })
	Register("claude/cli", func() types.Runner { return claude.New() })
	Register("opencode", func() types.Runner { return opencode.New() })
	Register("opencode/cli", func() types.Runner { return opencode.New() })
	Register("yaocode", func() types.Runner { return yaocode.New() })
	Register("tai", func() types.Runner { return tairunner.New() })
}
