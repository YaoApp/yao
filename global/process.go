package global

import (
	"github.com/yaoapp/gou"
)

// 注册处理器
func init() {
	gou.RegisterProcessHandler("xiang.global.ping", processPing)
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
