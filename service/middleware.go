package service

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/data"
	"github.com/yaoapp/xiang/share"
)

// AdminFileServer 数据管理平台
var AdminFileServer http.Handler = http.FileServer(data.AssetFS())

// AppFileServer 应用管理平台
var AppFileServer http.Handler = http.FileServer(http.Dir(config.Conf.RootUI))

// Middlewares 服务中间件
var Middlewares = []gin.HandlerFunc{
	BindDomain,
	BinStatic,
}

// BindDomain 绑定许可域名
func BindDomain(c *gin.Context) {

	for _, allow := range share.AllowHosts {
		if strings.Contains(c.Request.Host, allow) {
			c.Next()
			return
		}
	}

	c.JSON(403, gin.H{
		"code":    403,
		"message": fmt.Sprintf("%s is not allowed", c.Request.Host),
	})
	c.Abort()
}

// BinStatic 静态文件服务
func BinStatic(c *gin.Context) {

	if len(c.Request.URL.Path) >= 5 && c.Request.URL.Path[0:5] == "/api/" { // API接口
		c.Next()
		return

	} else if len(c.Request.URL.Path) >= 7 && c.Request.URL.Path[0:7] == "/xiang/" { // 数据管理后台
		AdminFileServer.ServeHTTP(c.Writer, c.Request)
		c.Abort()
		return
	}

	// 应用静态文件请求
	AppFileServer.ServeHTTP(c.Writer, c.Request)
	c.Abort()
}
