package service

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/helper"
	table_v0 "github.com/yaoapp/yao/table"

	"github.com/yaoapp/yao/widgets/chart"
	"github.com/yaoapp/yao/widgets/form"
	"github.com/yaoapp/yao/widgets/table"
)

// Guards middlewares
var Guards = map[string]gin.HandlerFunc{
	"bearer-jwt":   guardBearerJWT,   // Bearer JWT
	"cross-origin": guardCrossOrigin, // Cross-Origin Resource Sharing
	"table-guard":  table_v0.Guard,   // Table Guard ( v0.9 table)
	"widget-table": table.Guard,      // Widget Table Guard
	"widget-form":  form.Guard,       // Widget Form Guard
	"widget-chart": chart.Guard,      // Widget Chart Guard
}

// JWT Bearer JWT
func guardBearerJWT(c *gin.Context) {
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

// CORS Cross Origin
func guardCrossOrigin(c *gin.Context) {
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
