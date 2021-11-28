package network

import "github.com/yaoapp/gou"

func init() {
	// 注册处理器
	gou.RegisterProcessHandler("xiang.network.ip", ProcessIP)
}
