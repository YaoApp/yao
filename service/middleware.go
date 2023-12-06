package service

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/sui/api"
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
		log.Debug("Rewrite: %s => %s", c.Request.URL.Path, rewrite.Replacement)
		if matches := rewrite.Pattern.FindStringSubmatch(c.Request.URL.Path); matches != nil {
			log.Debug("Rewrite FindStringSubmatch: %s => %s", c.Request.URL.Path, rewrite.Replacement)
			c.Request.URL.Path = rewrite.Pattern.ReplaceAllString(c.Request.URL.Path, rewrite.Replacement)
			break
		}
	}

	// Sui file server
	if strings.HasSuffix(c.Request.URL.Path, ".sui") {

		r, code, err := api.NewRequestContext(c)
		if err != nil {
			c.AbortWithError(code, err)
			return
		}

		html, code, err := r.Render()
		if err != nil {
			c.AbortWithError(code, err)
			return
		}

		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(200, html)
		c.Done()
		return
	}

	// static file server
	AppFileServer.ServeHTTP(c.Writer, c.Request)
	c.Abort()
}
