package global

import (
	"github.com/yaoapp/gou"
)

var shutdown = make(chan bool)
var shutdownComplete = make(chan bool)

// ServiceStart 启动服务
func ServiceStart() {
	gou.SetHTTPGuards(Guards)
	gou.ServeHTTP(
		gou.Server{
			Host:   Conf.Service.Host,
			Port:   Conf.Service.Port,
			Allows: Conf.Service.Allow,
			Root:   "/api",
		},
		&shutdown, func(s gou.Server) {
			shutdownComplete <- true
		},
		Middlewares...)
}

// ServiceStop 关闭服务
func ServiceStop(onComplete func()) {
	shutdown <- true
	<-shutdownComplete
	onComplete()
}
