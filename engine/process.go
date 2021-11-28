package engine

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/share"
	"github.com/yaoapp/xiang/xfs"
)

func init() {
	// 注册处理器
	gou.RegisterProcessHandler("xiang.main.Ping", processPing)
	gou.RegisterProcessHandler("xiang.main.FileContent", processFileContent)
	gou.RegisterProcessHandler("xiang.main.AppFileContent", processAppFileContent)
	gou.RegisterProcessHandler("xiang.main.Inspect", processInspect)
	gou.RegisterProcessHandler("xiang.main.Favicon", processFavicon)
}

// processCreate 运行模型 MustCreate
func processPing(process *gou.Process) interface{} {
	var input interface{}
	if process.NumOfArgs() > 0 {
		input = process.Args[0]
	}

	res := map[string]interface{}{
		"code":    200,
		"server":  "象传应用引擎",
		"version": share.VERSION,
		"domain":  share.DOMAIN,
		"allows":  config.Conf.Service.Allow,
		"args":    input,
	}
	return res
}

// processInspect 返回系统信息
func processInspect(process *gou.Process) interface{} {
	share.App.Icons.Set("favicon", "/api/xiang/favicon.ico")
	return share.App.Public()
}

// processFavicon 运行模型 MustCreate
func processFavicon(process *gou.Process) interface{} {
	return xfs.DecodeString(share.App.Icons.Get("png").(string))
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
	fs := xfs.New(config.Conf.RootData)
	filename := process.ArgsString(0)
	encode := process.ArgsBool(1, true)
	content := fs.MustReadFile(filename)
	if encode {
		return xfs.Encode(content)
	}
	return string(content)
}
