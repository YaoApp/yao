package service

import (
	"context"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

var shutdown = make(chan bool, 1)

var shutdownComplete = make(chan bool, 1)

// Start 启动服务
func Start() error {

	err := prepare()
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
		shutdown, func(s gou.Server) {
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
		shutdown, func(s gou.Server) {
			shutdownComplete <- true
		},
		Middlewares...)
}

// StopWithouttSession 关闭服务
func StopWithouttSession(onComplete func()) {
	shutdown <- true
	select {
	case <-shutdownComplete:
		onComplete()
	}
}

// Stop 关闭服务
func Stop(onComplete func()) {
	shutdown <- true
	select {
	case <-shutdownComplete:
		share.SessionStop()
		share.DBClose()
		onComplete()
	}
}

// StopWithContext stop with timeout
func StopWithContext(ctx context.Context, onComplete func()) {
	shutdown <- true
	select {
	case <-ctx.Done():
		log.Error("[STOP] canceled (%v)", ctx.Err())
		onComplete()
	case <-shutdownComplete:
		share.SessionStop()
		onComplete()
	}
}

func prepare() error {

	// Session server
	err := share.SessionStart()
	if err != nil {
		return err
	}

	err = SetupStatic()
	if err != nil {
		return err
	}

	return nil
}
