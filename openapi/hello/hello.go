package hello

import (
	"io"
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

	// Get query string as raw string
	queryString := c.Request.URL.RawQuery

	// Get post payload
	var postPayload string
	if body, err := io.ReadAll(c.Request.Body); err == nil {
		postPayload = string(body)
	}

	c.JSON(http.StatusOK, gin.H{
		"MESSAGE":      "HELLO, WORLD",
		"SERVER_TIME":  serverTime,
		"VERSION":      share.VERSION,
		"PRVERSION":    share.PRVERSION,
		"CUI":          share.CUI,
		"PRCUI":        share.PRCUI,
		"APP":          share.App.Name,
		"APP_VERSION":  share.App.Version,
		"QUERYSTRING":  queryString,
		"POST_PAYLOAD": postPayload,
	})
}

// helloWorldHello is the handler for the hello world endpoint
func helloWorldProtected(c *gin.Context) {
	serverTime := time.Now().Format(time.RFC3339)

	// Get query string as raw string
	queryString := c.Request.URL.RawQuery

	// Get post payload
	var postPayload string
	if body, err := io.ReadAll(c.Request.Body); err == nil {
		postPayload = string(body)
	}

	c.JSON(http.StatusOK, gin.H{
		"MESSAGE":      "HELLO, WORLD",
		"SERVER_TIME":  serverTime,
		"VERSION":      share.VERSION,
		"PRVERSION":    share.PRVERSION,
		"CUI":          share.CUI,
		"PRCUI":        share.PRCUI,
		"APP":          share.App.Name,
		"APP_VERSION":  share.App.Version,
		"QUERYSTRING":  queryString,
		"POST_PAYLOAD": postPayload,
	})
}
