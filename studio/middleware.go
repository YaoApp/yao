package studio

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/xun"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/helper"
)

// hdRecovered custom recovered
func hdRecovered(c *gin.Context, recovered interface{}) {

	var code = http.StatusInternalServerError

	if err, ok := recovered.(string); ok {
		c.JSON(code, xun.R{
			"code":    code,
			"message": fmt.Sprintf("%s", err),
		})
	} else if err, ok := recovered.(exception.Exception); ok {
		code = err.Code
		c.JSON(code, xun.R{
			"code":    code,
			"message": err.Message,
		})
	} else if err, ok := recovered.(*exception.Exception); ok {
		code = err.Code
		c.JSON(code, xun.R{
			"code":    code,
			"message": err.Message,
		})
	} else {
		c.JSON(code, xun.R{
			"code":    code,
			"message": fmt.Sprintf("%v", recovered),
		})
	}

	c.AbortWithStatus(code)
}

// cross domian
func hdCrossDomain(c *gin.Context) {
}

// studio API Auth
func hdAuth(c *gin.Context) {

	tokenString := c.Request.Header.Get("Authorization")
	if strings.HasPrefix(tokenString, "Bearer") {
		tokenString = strings.TrimSpace(strings.TrimPrefix(tokenString, "Bearer "))
		if tokenString == "" {
			c.JSON(403, gin.H{"code": 403, "message": "No permission"})
			c.Abort()
			return
		}

		claims := helper.JwtValidate(tokenString, config.Conf.Studio.Secret)
		c.Set("__sid", claims.SID)
		return

	} else if strings.HasPrefix(tokenString, "Signature ") { // For Yao Studio
		signature := strings.TrimSpace(strings.TrimPrefix(tokenString, "Signature "))
		nonce := c.Request.Header.Get("Studio-Nonce")
		ts := c.Request.Header.Get("Studio-Timestamp")
		query := c.Request.URL.Query()
		log.Trace("[Studio] %s, %s %s %v", signature, nonce, ts, query)
		return
	}

	c.JSON(403, gin.H{"code": 403, "message": "No permission"})
	c.Abort()
	return
}
