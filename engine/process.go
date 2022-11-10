package engine

import (
	"path/filepath"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/xfs"
)

func init() {
	// 注册处理器
	gou.RegisterProcessHandler("xiang.main.Ping", processPing) // deprecated → utils.app.Ping  @/utils/process.go
	gou.AliasProcess("xiang.main.Ping", "xiang.sys.Ping")      // deprecated

	gou.RegisterProcessHandler("xiang.main.FileContent", processFileContent)       // deprecated
	gou.RegisterProcessHandler("xiang.main.AppFileContent", processAppFileContent) // deprecated

	gou.RegisterProcessHandler("xiang.main.Inspect", processInspect) // deprecated → utils.app.Inspect @/utils/process.go
	gou.AliasProcess("xiang.main.Inspect", "xiang.sys.Inspect")      // deprecated

	gou.RegisterProcessHandler("xiang.main.Favicon", processFavicon) // deprecated
}

// processCreate 运行模型 MustCreate
func processPing(process *gou.Process) interface{} {
	res := map[string]interface{}{
		"engine":  share.BUILDNAME,
		"version": share.VERSION,
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
	fs := xfs.New(filepath.Join(config.Conf.Root, "data"))
	filename := process.ArgsString(0)
	encode := process.ArgsBool(1, true)
	content := fs.MustReadFile(filename)
	if encode {
		return xfs.Encode(content)
	}
	return string(content)
}
