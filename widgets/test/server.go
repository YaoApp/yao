package test

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/helper"
	"github.com/yaoapp/yao/share"
)

var shutdown = make(chan bool, 1)
var shutdownComplete = make(chan bool, 1)

// Start the api server
func Start(t *testing.T, guards map[string]gin.HandlerFunc, port int) error {

	err := share.SessionStart()
	if err != nil {
		return err
	}

	gin.SetMode(gin.ReleaseMode)
	gou.SetHTTPGuards(guards)
	gou.ServeHTTP(
		gou.Server{
			Host:   "127.0.0.1",
			Port:   port,
			Root:   "/api",
			Allows: config.Conf.AllowFrom,
		},
		shutdown, func(s gou.Server) {
			shutdownComplete <- true
		},
	)

	return nil
}

// Stop the api server
func Stop(onComplete func()) {
	shutdown <- true
	select {
	case <-shutdownComplete:
		share.SessionStop()
		onComplete()
	}
}

// GuardBearerJWT test guard
func GuardBearerJWT(c *gin.Context) {
	tokenString := c.Request.Header.Get("Authorization")
	tokenString = strings.TrimSpace(strings.TrimPrefix(tokenString, "Bearer "))

	if tokenString == "" {
		c.JSON(403, gin.H{"code": 403, "message": "No permission"})
		c.Abort()
		return
	}

	claims := helper.JwtValidate(tokenString)
	c.Set("__sid", claims.SID)
}
