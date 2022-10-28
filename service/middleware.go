package service

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/share"
)

// Middlewares 服务中间件
var Middlewares = []gin.HandlerFunc{
	BinStatic,
}

// BinStatic 静态文件服务
func BinStatic(c *gin.Context) {

	length := len(c.Request.URL.Path)

	if (length >= 5 && c.Request.URL.Path[0:5] == "/api/") ||
		(length >= 11 && c.Request.URL.Path[0:11] == "/websocket/") { // API & websocket
		c.Next()
		return

	} else if share.App.XGen == "1.0" {

		// Xgen 1.0
		if length >= AdminRootLen && c.Request.URL.Path[0:AdminRootLen] == AdminRoot {
			c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, c.Request.URL.Path[0:AdminRootLen-1])
			XGenFileServerV1.ServeHTTP(c.Writer, c.Request)
			c.Abort()
			return
		}

		if length >= 18 && c.Request.URL.Path[0:18] == "/__yao_admin_root/" {
			c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, "/__yao_admin_root")
			XGenFileServerV1.ServeHTTP(c.Writer, c.Request)
			c.Abort()
			return
		}

	} else if share.App.XGen == "" && length >= 7 && c.Request.URL.Path[0:7] == "/xiang/" {
		// Xgen 0.9
		c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, "/xiang")
		XGenFileServerV0.ServeHTTP(c.Writer, c.Request)
		c.Abort()
		return

	}

	// 应用内静态文件目录(/ui or public)
	AppFileServer.ServeHTTP(c.Writer, c.Request)
	c.Abort()
}
