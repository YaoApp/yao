package service

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/data"
)

// AdminFileServer 数据管理平台
var AdminFileServer http.Handler = http.FileServer(data.AssetFS())

// AppFileServer 应用静态文件
var AppFileServer http.Handler = http.FileServer(http.Dir(filepath.Join(config.Conf.Root, "ui")))

// Middlewares 服务中间件
var Middlewares = []gin.HandlerFunc{
	// BindDomain,
	BinStatic,
}

// BinStatic 静态文件服务
func BinStatic(c *gin.Context) {

	length := len(c.Request.URL.Path)

	if length >= 5 && c.Request.URL.Path[0:5] == "/api/" { // API接口
		c.Next()
		return
	} else if length >= 7 && c.Request.URL.Path[0:7] == "/xiang/" { // 数据管理后台
		c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, "/xiang")
		AdminFileServer.ServeHTTP(c.Writer, c.Request)
		c.Abort()
		return
	}

	// 应用内静态文件目录(/ui)
	AppFileServer.ServeHTTP(c.Writer, c.Request)
	c.Abort()
}
