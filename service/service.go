package service

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/data"
)

var shutdown = make(chan bool)
var shutdownComplete = make(chan bool)

// AdminFileServer 数据管理平台
var AdminFileServer http.Handler = http.FileServer(data.AssetFS())

// AppFileServer 应用静态文件
var AppFileServer http.Handler = http.FileServer(http.Dir(config.Conf.RootUI))

// Start 启动服务
func Start() {
	gou.SetHTTPGuards(Guards)
	gou.ServeHTTPCustomRouter(
		router(),
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

// router 返回路由
func router() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// 应用 UI 目录应用静态文件
	router.Any("/", func(c *gin.Context) {
		AppFileServer.ServeHTTP(c.Writer, c.Request)
	})

	// 数据管理后台
	router.Any("/xiang/*path", func(c *gin.Context) {
		c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, "/xiang")
		AdminFileServer.ServeHTTP(c.Writer, c.Request)
	})

	return router
}
