package network

// *******************************************************
// * DEPRECATED	→ http								     *
// *******************************************************

import "github.com/yaoapp/gou"

func init() {
	// 注册处理器
	gou.RegisterProcessHandler("xiang.network.ip", ProcessIP)
	gou.RegisterProcessHandler("xiang.network.FreePort", ProcessFreePort)
	gou.RegisterProcessHandler("xiang.network.Get", ProcessGet)
	gou.RegisterProcessHandler("xiang.network.Post", ProcessPost)
	gou.RegisterProcessHandler("xiang.network.PostJSON", ProcessPostJSON)
	gou.RegisterProcessHandler("xiang.network.Put", ProcessPut)
	gou.RegisterProcessHandler("xiang.network.PutJSON", ProcessPutJSON)
	gou.RegisterProcessHandler("xiang.network.Send", ProcessSend)
}
