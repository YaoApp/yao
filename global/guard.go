package global

import (
	"fmt"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/xiang/user"
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
	if Conf.Mode == "debug" {
		xlog.Printf("JWT: %s Secret: %s", tokenString, Conf.JWT.Secret)
	}
	token, err := jwt.ParseWithClaims(tokenString, &user.JwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(Conf.JWT.Secret), nil
	})

	if err != nil {
		xlog.Printf("JWT ParseWithClaims Error: %s", err)
		c.JSON(403, gin.H{"code": 403, "message": fmt.Sprintf("登录已过期或令牌失效(%s)", err)})
		c.Abort()
		return
	}

	if claims, ok := token.Claims.(*user.JwtClaims); ok && token.Valid {
		c.Set("id", claims.Subject)
		c.Set("type", claims.Type)
		c.Set("name", claims.Name)
		c.Next()
		return
	}

	// fmt.Println("bearer-JWT", token.Claims.Valid())
	c.JSON(403, gin.H{"code": 403, "message": "无权访问该页面"})
	c.Abort()
	return
}
