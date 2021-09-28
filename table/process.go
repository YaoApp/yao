package table

import "github.com/yaoapp/gou"

func init() {
	// 注册处理器
	gou.RegisterProcessHandler("xiang.table.Search", processSearch)
}

func processSearch(process *gou.Process) interface{} {

	return nil
}
