package agent

import (
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/yao/sui/core"
)

// Agent is the struct for the agent sui storage
// It extends local storage with special page loading from /assistants/<name>/pages/
type Agent struct {
	root           string // /agent
	assistantsRoot string // /assistants
	fs             fs.FileSystem
	*core.DSL
}
