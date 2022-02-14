package service

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

var shutdown = make(chan bool)
var shutdownComplete = make(chan bool)

// Start 启动服务
func Start() {

	if config.Conf.Session.Hosting && config.Conf.Session.IsCLI == false {
		share.SessionServerStart()
	}

	gou.SetHTTPGuards(Guards)
	gou.ServeHTTP(
		gou.Server{
			Host: config.Conf.Host,
			Port: config.Conf.Port,
			Root: "/api",
		},
		&shutdown, func(s gou.Server) {
			shutdownComplete <- true
		},
		Middlewares...)
}

// StartWithouttSession 启动服务
func StartWithouttSession() {

	gou.SetHTTPGuards(Guards)
	gou.ServeHTTP(
		gou.Server{
			Host: config.Conf.Host,
			Port: config.Conf.Port,
			Root: "/api",
		},
		&shutdown, func(s gou.Server) {
			shutdownComplete <- true
		},
		Middlewares...)
}

// StopWithouttSession 关闭服务
func StopWithouttSession(onComplete func()) {
	shutdown <- true
	<-shutdownComplete
	gou.KillPlugins()
	onComplete()
}

// Stop 关闭服务
func Stop(onComplete func()) {
	shutdown <- true
	<-shutdownComplete
	share.SessionServerStop()
	gou.KillPlugins()
	onComplete()
}
