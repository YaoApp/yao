package service

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/helper"
	"github.com/yaoapp/yao/table"
)

// Guards 服务中间件
var Guards = map[string]gin.HandlerFunc{
	"bearer-jwt":   bearerJWT,   // JWT 鉴权
	"cross-domain": crossDomain, // 跨域许可
	"table-guard":  table.Guard, // Table Guard
}

// JWT 鉴权
func bearerJWT(c *gin.Context) {
	tokenString := c.Request.Header.Get("Authorization")
	tokenString = strings.TrimSpace(strings.TrimPrefix(tokenString, "Bearer "))
	if tokenString == "" {
		c.JSON(403, gin.H{"code": 403, "message": "无权访问该页面"})
		c.Abort()
		return
	}

	claims := helper.JwtValidate(tokenString)
	c.Set("__sid", claims.SID)
	c.Next()
}

// crossDomain 跨域访问
func crossDomain(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

	if c.Request.Method == "OPTIONS" {
		c.AbortWithStatus(204)
		return
	}

	c.Next()
}
