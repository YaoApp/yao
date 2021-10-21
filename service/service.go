package service

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/xiang/config"
)

var shutdown = make(chan bool)
var shutdownComplete = make(chan bool)

// Start 启动服务
func Start() {
	gou.SetHTTPGuards(Guards)
	gou.ServeHTTP(
		gou.Server{
			Host:   config.Conf.Service.Host,
			Port:   config.Conf.Service.Port,
			Allows: config.Conf.Service.Allow,
			Root:   "/api",
		},
		&shutdown, func(s gou.Server) {
			shutdownComplete <- true
		},
		Middlewares...)
}

// Stop 关闭服务
func Stop(onComplete func()) {
	shutdown <- true
	<-shutdownComplete
	onComplete()
}
