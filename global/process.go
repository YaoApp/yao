package global

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/xiang/xfs"
)

func init() {
	// 注册处理器
	gou.RegisterProcessHandler("xiang.global.Ping", processPing)
	gou.RegisterProcessHandler("xiang.global.FileContent", processFileContent)
	gou.RegisterProcessHandler("xiang.global.AppFileContent", processAppFileContent)
	gou.RegisterProcessHandler("xiang.global.Inspect", processInspect)
}

// processCreate 运行模型 MustCreate
func processPing(process *gou.Process) interface{} {
	res := map[string]interface{}{
		"code":    200,
		"server":  "象传应用引擎",
		"version": VERSION,
		"domain":  DOMAIN,
		"allows":  Conf.Service.Allow,
	}
	return res
}

// processCreate 运行模型 MustCreate
func processInspect(process *gou.Process) interface{} {
	return App
}

// processFileContent 返回文件内容
func processFileContent(process *gou.Process) interface{} {
	process.ValidateArgNums(2)
	filename := process.ArgsString(0)
	encode := process.ArgsBool(1, true)
	content := xfs.Stor.MustReadFile(filename)
	if encode {
		return xfs.Encode(content)
	}
	return string(content)
}

// processAppFileContent 返回应用文件内容
func processAppFileContent(process *gou.Process) interface{} {
	process.ValidateArgNums(2)
	fs := xfs.New(Conf.RootData)
	filename := process.ArgsString(0)
	encode := process.ArgsBool(1, true)
	content := fs.MustReadFile(filename)
	if encode {
		return xfs.Encode(content)
	}
	return string(content)
}
