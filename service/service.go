package service

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

var shutdown = make(chan bool)
var shutdownComplete = make(chan bool)

// Start 启动服务
func Start() error {

	err := share.SessionStart()
	if err != nil {
		return err
	}

	gou.SetHTTPGuards(Guards)
	gou.ServeHTTP(
		gou.Server{
			Host:   config.Conf.Host,
			Port:   config.Conf.Port,
			Root:   "/api",
			Allows: config.Conf.AllowFrom,
		},
		&shutdown, func(s gou.Server) {
			shutdownComplete <- true
		},
		Middlewares...)

	return nil
}

// StartWithouttSession 启动服务
func StartWithouttSession() {

	gou.SetHTTPGuards(Guards)
	gou.ServeHTTP(
		gou.Server{
			Host:   config.Conf.Host,
			Port:   config.Conf.Port,
			Root:   "/api",
			Allows: config.Conf.AllowFrom,
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
	share.SessionStop()
	gou.KillPlugins()
	onComplete()
}
