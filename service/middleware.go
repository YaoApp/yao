package service

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/xiang/share"
)

// Middlewares 服务中间件
var Middlewares = []gin.HandlerFunc{
	BindDomain,
}

// BindDomain 绑定许可域名
func BindDomain(c *gin.Context) {

	for _, allow := range share.AllowHosts {
		if strings.Contains(c.Request.Host, allow) {
			c.Next()
			return
		}
	}

	c.JSON(403, gin.H{
		"code":    403,
		"message": fmt.Sprintf("%s is not allowed", c.Request.Host),
	})
	c.Abort()
}
