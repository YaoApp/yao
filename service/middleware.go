package service

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// Middlewares the middlewares
var Middlewares = []gin.HandlerFunc{
	gin.Logger(),
	withStaticFileServer,
}

// withStaticFileServer static file server
func withStaticFileServer(c *gin.Context) {

	// Handle API & websocket
	length := len(c.Request.URL.Path)
	if (length >= 5 && c.Request.URL.Path[0:5] == "/api/") ||
		(length >= 11 && c.Request.URL.Path[0:11] == "/websocket/") { // API & websocket
		c.Next()
		return
	}

	// Xgen 1.0
	if length >= AdminRootLen && c.Request.URL.Path[0:AdminRootLen] == AdminRoot {
		c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, c.Request.URL.Path[0:AdminRootLen-1])
		XGenFileServerV1.ServeHTTP(c.Writer, c.Request)
		c.Abort()
		return
	}

	// __yao_admin_root
	if length >= 18 && c.Request.URL.Path[0:18] == "/__yao_admin_root/" {
		c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, "/__yao_admin_root")
		XGenFileServerV1.ServeHTTP(c.Writer, c.Request)
		c.Abort()
		return
	}

	// Rewrite
	for _, rewrite := range rewriteRules {
		if matches := rewrite.Pattern.FindStringSubmatch(c.Request.URL.Path); matches != nil {
			c.Request.URL.Path = rewrite.Pattern.ReplaceAllString(c.Request.URL.Path, rewrite.Replacement)
			break
		}
	}

	// static file server
	AppFileServer.ServeHTTP(c.Writer, c.Request)
	c.Abort()
}
