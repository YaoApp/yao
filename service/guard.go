package service

import (
	"fmt"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/helper"
)

// Guards 服务中间件
var Guards = map[string]gin.HandlerFunc{
	"bearer-jwt":   bearerJWT,   // JWT 鉴权
	"cross-domain": crossDomain, // 跨域许可
}

// JWT 鉴权
func bearerJWT(c *gin.Context) {
	tokenString := c.Request.Header.Get("Authorization")
	if tokenString == "" {
		c.JSON(403, gin.H{"code": 403, "message": "无权访问该页面"})
		c.Abort()
		return
	}

	tokenString = strings.TrimSpace(strings.TrimPrefix(tokenString, "Bearer "))
	log.Debug("JWT: %s Secret: %s", tokenString, config.Conf.JWTSecret)
	token, err := jwt.ParseWithClaims(tokenString, &helper.JwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.Conf.JWTSecret), nil
	})

	if err != nil {
		log.Error("JWT ParseWithClaims Error: %s", err)
		c.JSON(403, gin.H{"code": 403, "message": fmt.Sprintf("登录已过期或令牌失效(%s)", err)})
		c.Abort()
		return
	}

	if claims, ok := token.Claims.(*helper.JwtClaims); ok && token.Valid {
		c.Set("__sid", claims.SID)
		c.Next()
		return
	}

	// fmt.Println("bearer-JWT", token.Claims.Valid())
	c.JSON(403, gin.H{"code": 403, "message": "无权访问该页面"})
	c.Abort()
	return
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
