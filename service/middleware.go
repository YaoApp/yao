package service

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// Middlewares the middlewares
var Middlewares = []func(c *gin.Context){
	withStaticFileServer,
}

// withStaticFileServer static file server
func withStaticFileServer(c *gin.Context) {

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

	if length >= 18 && c.Request.URL.Path[0:18] == "/__yao_admin_root/" {
		c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, "/__yao_admin_root")
		XGenFileServerV1.ServeHTTP(c.Writer, c.Request)
		c.Abort()
		return
	}

	// PWA app static file server
	for root, length := range SpaRoots {
		if length >= length && c.Request.URL.Path[0:length] == root {
			c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, root)
			spaFileServers[root].ServeHTTP(c.Writer, c.Request)
			c.Abort()
			return
		}
	}

	// static file server
	AppFileServer.ServeHTTP(c.Writer, c.Request)
	c.Abort()
}
