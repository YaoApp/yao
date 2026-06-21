package task

import (
	tasksvc "github.com/yaoapp/yao/agent/task"
	tasktools "github.com/yaoapp/yao/tools/task"
)

func init() {
	tasktools.FnList = tasksvc.List
	tasktools.FnCreate = tasksvc.Create
	tasktools.FnMove = tasksvc.Move
}
