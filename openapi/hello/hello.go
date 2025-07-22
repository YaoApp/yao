package hello

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/share"
)

// Attach attaches the hello world handlers to the router
func Attach(group *gin.RouterGroup, oauth types.OAuth) {

	// Health check
	group.GET("/public", helloWorldPublic)
	group.POST("/public", helloWorldPublic)

	// OAuth Protected Resource
	group.GET("/protected", oauth.Guard, helloWorldProtected)
	group.POST("/protected", oauth.Guard, helloWorldProtected)
}

func helloWorldPublic(c *gin.Context) {
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
func helloWorldProtected(c *gin.Context) {
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
