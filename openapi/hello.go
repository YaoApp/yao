package openapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/share"
)

// attachHelloWorld attaches the hello world handlers to the router
func (openapi *OpenAPI) attachHelloWorld(base *gin.RouterGroup) {

	// hello handlers
	hello := base.Group("/helloworld")

	// Health check
	hello.GET("/public", openapi.helloWorldPublic)
	hello.POST("/public", openapi.helloWorldPublic)

	// OAuth Protected Resource
	hello.GET("/protected", openapi.OAuth.Guard, openapi.helloWorldProtected)
	hello.POST("/protected", openapi.OAuth.Guard, openapi.helloWorldProtected)
}

// helloWorldPublic is the handler for the hello world endpoint
func (openapi *OpenAPI) helloWorldPublic(c *gin.Context) {
	serverTime := time.Now().Format(time.RFC3339)
	c.JSON(http.StatusOK, gin.H{
		"MESSAGE":     "HELLO, WORLD",
		"SERVER_TIME": serverTime,
		"VERSION":     share.VERSION,
		"PRVERSION":   share.PRVERSION,
		"CUI":         share.CUI,
		"PRCUI":       share.PRCUI,
		"APP":         share.App.Name,
		"APP_VERSION": share.App.Version,
	})
}

// helloWorldHello is the handler for the hello world endpoint
func (openapi *OpenAPI) helloWorldProtected(c *gin.Context) {
	serverTime := time.Now().Format(time.RFC3339)
	c.JSON(http.StatusOK, gin.H{
		"MESSAGE":     "HELLO, WORLD",
		"SERVER_TIME": serverTime,
		"VERSION":     share.VERSION,
		"PRVERSION":   share.PRVERSION,
		"CUI":         share.CUI,
		"PRCUI":       share.PRCUI,
		"APP":         share.App.Name,
		"APP_VERSION": share.App.Version,
	})
}
