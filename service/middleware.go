package service

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/data"
	"github.com/yaoapp/yao/share"
)

// XGenFileServerV0 XGen v0.9
var XGenFileServerV0 http.Handler = http.FileServer(data.XgenV0())

// XGenFileServerV1 XGen v1.0
var XGenFileServerV1 http.Handler = http.FileServer(data.XgenV1())

// AppFileServer 应用静态文件
var AppFileServer http.Handler = http.FileServer(http.Dir(filepath.Join(config.Conf.Root, "ui")))

// Middlewares 服务中间件
var Middlewares = []gin.HandlerFunc{
	// BindDomain,
	BinStatic,
}

// AdminRoot cache
var AdminRoot = ""

// AdminRootLen cache
var AdminRootLen = 0

// BinStatic 静态文件服务
func BinStatic(c *gin.Context) {

	length := len(c.Request.URL.Path)

	if (length >= 5 && c.Request.URL.Path[0:5] == "/api/") ||
		(length >= 11 && c.Request.URL.Path[0:11] == "/websocket/") { // API & websocket
		c.Next()
		return

	} else if share.App.XGen == "1.0" {
		// Xgen 1.0
		adminRoot, adminRootLen := adminRoot()
		if length >= adminRootLen && c.Request.URL.Path[0:adminRootLen] == adminRoot {
			c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, c.Request.URL.Path[0:adminRootLen-1])
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

	// 应用内静态文件目录(/ui)
	AppFileServer.ServeHTTP(c.Writer, c.Request)
	c.Abort()
}

func adminRoot() (string, int) {
	if AdminRoot != "" {
		return AdminRoot, AdminRootLen
	}

	adminRoot := "/yao/"
	if share.App.AdminRoot != "" {
		root := strings.TrimPrefix(share.App.AdminRoot, "/")
		root = strings.TrimSuffix(root, "/")
		adminRoot = fmt.Sprintf("/%s/", root)
	}
	adminRootLen := len(adminRoot)
	AdminRoot = adminRoot
	AdminRootLen = adminRootLen
	return AdminRoot, AdminRootLen
}
