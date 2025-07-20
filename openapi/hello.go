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
	hello.GET("/hello", openapi.helloWorldHello)
	hello.POST("/hello", openapi.helloWorldHello)
}

// helloWorldHello is the handler for the hello world endpoint
func (openapi *OpenAPI) helloWorldHello(c *gin.Context) {
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
