package service

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yaoapp/yao/helper"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/oauth"

	"github.com/yaoapp/yao/widgets/chart"
	"github.com/yaoapp/yao/widgets/dashboard"
	"github.com/yaoapp/yao/widgets/form"
	"github.com/yaoapp/yao/widgets/list"
	"github.com/yaoapp/yao/widgets/table"
)

// Guards middlewares
var Guards = map[string]gin.HandlerFunc{
	"bearer-jwt":       guardBearerJWT,   // Bearer JWT
	"query-jwt":        guardQueryJWT,    // Get JWT Token from query string  "__tk"
	"cross-origin":     guardCrossOrigin, // Cross-Origin Resource Sharing
	"cookie-trace":     guardCookieTrace, // Set sid cookie
	"cookie-jwt":       guardCookieJWT,   // Get JWT Token from cookie "__tk"
	"widget-table":     table.Guard,      // Widget Table Guard
	"widget-list":      list.Guard,       // Widget List Guard
	"widget-form":      form.Guard,       // Widget Form Guard
	"widget-chart":     chart.Guard,      // Widget Chart Guard
	"widget-dashboard": dashboard.Guard,  // Widget Dashboard Guard
}

// guardCookieTrace set sid cookie
func guardCookieTrace(c *gin.Context) {
	sid, err := c.Cookie("sid")
	if err != nil {
		sid = uuid.New().String()
		c.SetCookie("sid", sid, 0, "/", "", false, true)
		c.Set("__sid", sid)
		c.Next()
		return
	}
	c.Set("__sid", sid)
}

// Cookie Cookie JWT
func guardCookieJWT(c *gin.Context) {

	// OpenAPI OAuth
	if openapi.Server != nil {
		guardOpenapiOauth(c)
		return
	}

	// Backward compatibility
	tokenString, err := c.Cookie("__tk")
	if err != nil {
		c.JSON(403, gin.H{"code": 403, "message": "Not Authorized"})
		c.Abort()
		return
	}

	if tokenString == "" {
		c.JSON(403, gin.H{"code": 403, "message": "Not Authorized"})
		c.Abort()
		return
	}

	claims := helper.JwtValidate(tokenString)
	c.Set("__sid", claims.SID)
}

// JWT Bearer JWT
func guardBearerJWT(c *gin.Context) {

	// OpenAPI OAuth
	if openapi.Server != nil {
		guardOpenapiOauth(c)
		return
	}

	// Backward compatibility
	tokenString := c.Request.Header.Get("Authorization")
	tokenString = strings.TrimSpace(strings.TrimPrefix(tokenString, "Bearer "))
	if tokenString == "" {
		c.JSON(403, gin.H{"code": 403, "message": "Not Authorized"})
		c.Abort()
		return
	}

	claims := helper.JwtValidate(tokenString)
	c.Set("__sid", claims.SID)
}

// JWT Bearer JWT
func guardQueryJWT(c *gin.Context) {
	tokenString := c.Query("__tk")
	if tokenString == "" {
		c.JSON(403, gin.H{"code": 403, "message": "Not Authorized"})
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

// Openapi Oauth
func guardOpenapiOauth(c *gin.Context) {
	s := oauth.OAuth
	token := getAccessToken(c)
	if token == "" {
		c.JSON(403, gin.H{"code": 403, "message": "Not Authorized"})
		c.Abort()
		return
	}

	// Validate the token
	_, err := s.VerifyToken(token)
	if err != nil {
		c.JSON(403, gin.H{"code": 403, "message": "Not Authorized"})
		c.Abort()
		return
	}

	// Get the session ID
	sid := getSessionID(c)
	if sid == "" {
		c.JSON(403, gin.H{"code": 403, "message": "Not Authorized"})
		c.Abort()
		return
	}

	c.Set("__sid", sid)
}

func getAccessToken(c *gin.Context) string {
	token := c.GetHeader("Authorization")
	if token == "" || token == "Bearer undefined" {
		cookie, err := c.Cookie("__Host-access_token")
		if err != nil {
			return ""
		}
		token = cookie
	}
	return strings.TrimPrefix(token, "Bearer ")
}

func getSessionID(c *gin.Context) string {
	sid, err := c.Cookie("__Host-session_id")
	if err != nil {
		return ""
	}
	return sid
}
