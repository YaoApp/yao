package server

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/xiang/global"
)

// Start 启动服务
func Start() {
	gou.SetHTTPGuards(Guards)
	gou.ServeHTTP(gou.Server{
		Host:   global.Conf.Service.Host,
		Port:   global.Conf.Service.Port,
		Allows: global.Conf.Service.Allow,
		Root:   "/api",
	}, Middlewares...)
}
