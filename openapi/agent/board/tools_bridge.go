package board

import (
	boardsvc "github.com/yaoapp/yao/agent/board"
	boardtools "github.com/yaoapp/yao/tools/board"
)

func init() {
	boardtools.FnList = boardsvc.List
	boardtools.FnCreate = boardsvc.Create
}
