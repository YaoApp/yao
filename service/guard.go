package service

import (
	"fmt"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/helper"
	"github.com/yaoapp/xiang/xlog"
)

// Guards 服务中间件
var Guards = map[string]gin.HandlerFunc{
	"bearer-jwt": bearerJWT, // JWT 鉴权
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
	if config.Conf.Mode == "debug" {
		xlog.Printf("JWT: %s Secret: %s", tokenString, config.Conf.JWT.Secret)
	}
	token, err := jwt.ParseWithClaims(tokenString, &helper.JwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.Conf.JWT.Secret), nil
	})

	if err != nil {
		xlog.Printf("JWT ParseWithClaims Error: %s", err)
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
