package engine

import (
	"fmt"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

func init() {
	// 注册处理器
	process.Register("xiang.main.Ping", processPing)   // deprecated → utils.app.Ping  @/utils/process.go
	process.Alias("xiang.main.Ping", "xiang.sys.Ping") // deprecated

	process.Register("xiang.main.FileContent", processFileContent)       // deprecated
	process.Register("xiang.main.AppFileContent", processAppFileContent) // deprecated

	process.Register("xiang.main.Inspect", processInspect)   // deprecated → utils.app.Inspect @/utils/process.go
	process.Alias("xiang.main.Inspect", "xiang.sys.Inspect") // deprecated

	process.Register("xiang.main.Favicon", processFavicon) // deprecated

	// Application
	process.Alias("xiang.main.Ping", "utils.app.Ping")
	process.Alias("xiang.main.Inspect", "utils.app.Inspect")
}

// processCreate 运行模型 MustCreate
func processPing(process *process.Process) interface{} {
	res := map[string]interface{}{
		"engine":  share.BUILDNAME,
		"version": share.VERSION,
	}
	return res
}

// processInspect 返回系统信息
func processInspect(process *process.Process) interface{} {
	return map[string]interface{}{
		"VERSION":   fmt.Sprintf("%s %s", share.VERSION, share.PRVERSION),
		"BUILDNAME": share.BUILDNAME,
		"CONFIG":    config.Conf,
	}
}

// processFavicon 运行模型 MustCreate
func processFavicon(process *process.Process) interface{} {
	// return xfs.DecodeString(share.App.Icons.Get("png").(string))
	return nil
}

// processFileContent 返回文件内容
func processFileContent(process *process.Process) interface{} {
	// process.ValidateArgNums(2)
	// filename := process.ArgsString(0)
	// encode := process.ArgsBool(1, true)
	// content := xfs.Stor.MustReadFile(filename)
	// if encode {
	// 	return xfs.Encode(content)
	// }
	// return string(content)
	return nil
}

// processAppFileContent 返回应用文件内容
func processAppFileContent(process *process.Process) interface{} {
	// process.ValidateArgNums(2)
	// fs := xfs.New(filepath.Join(config.Conf.Root, "data"))
	// filename := process.ArgsString(0)
	// encode := process.ArgsBool(1, true)
	// content := fs.MustReadFile(filename)
	// if encode {
	// 	return xfs.Encode(content)
	// }
	// return string(content)
	return nil
}
