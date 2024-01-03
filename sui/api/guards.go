package api

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/helper"
)

// Guards middlewares
var Guards = map[string]func(c *Request) error{
	"bearer-jwt": guardBearerJWT, // Bearer JWT
	"query-jwt":  guardQueryJWT,  // Get JWT Token from query string  "__tk"
	"cookie-jwt": guardCookieJWT, // Get JWT Token from cookie "__tk"

}

// JWT Bearer JWT
func guardBearerJWT(r *Request) error {
	if r.context == nil {
		return fmt.Errorf("No permission")
	}
	c := r.context
	tokenString := c.Request.Header.Get("Authorization")
	tokenString = strings.TrimSpace(strings.TrimPrefix(tokenString, "Bearer "))
	if tokenString == "" {
		c.JSON(403, gin.H{"code": 403, "message": "No permission"})
		c.Abort()
		return fmt.Errorf("No permission")
	}

	claims := helper.JwtValidate(tokenString)
	c.Set("__sid", claims.SID)
	r.Sid = claims.SID
	return nil
}

// JWT Bearer JWT
func guardCookieJWT(r *Request) error {
	if r.context == nil {
		return fmt.Errorf("No permission")
	}
	c := r.context

	tokenString, err := c.Cookie("__tk")
	if err != nil {
		c.JSON(403, gin.H{"code": 403, "message": "No permission"})
		c.Abort()
		return fmt.Errorf("No permission")
	}

	if tokenString == "" {
		c.JSON(403, gin.H{"code": 403, "message": "No permission"})
		c.Abort()
		return fmt.Errorf("No permission")
	}

	claims := helper.JwtValidate(tokenString)
	c.Set("__sid", claims.SID)
	r.Sid = claims.SID
	return nil
}

// JWT Bearer JWT
func guardQueryJWT(r *Request) error {
	if r.context == nil {
		return fmt.Errorf("No permission")
	}
	c := r.context

	tokenString := c.Query("__tk")
	if tokenString == "" {
		c.JSON(403, gin.H{"code": 403, "message": "No permission"})
		c.Abort()
		return fmt.Errorf("No permission")
	}

	claims := helper.JwtValidate(tokenString)
	c.Set("__sid", claims.SID)
	r.Sid = claims.SID
	return nil
}

// ProcessGuard guard process
func (r *Request) processGuard(name string) error {
	var body interface{}
	c := r.context

	if c.Request.Body != nil {

		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err == nil {
			if strings.HasPrefix(strings.ToLower(c.Request.Header.Get("Content-Type")), "application/json") {
				jsoniter.Unmarshal(bodyBytes, &body)
			} else {
				body = string(bodyBytes)
			}
		}

		// Reset body
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	params := map[string]string{}
	for _, param := range c.Params {
		params[param.Key] = param.Value
	}

	args := []interface{}{
		r.URL,     // page url
		r.Params,  // page params
		r.Query,   // query string
		r.Payload, // payload
		r.Headers, // Request headers
	}

	process, err := process.Of(name, args...)
	if err != nil {
		c.JSON(403, gin.H{"code": 403, "message": fmt.Sprintf("Guard: %s %s", name, err.Error())})
		c.Abort()
		return err
	}

	if sid, has := c.Get("__sid"); has { // 设定会话ID
		if sid, ok := sid.(string); ok {
			process.WithSID(sid)
		}
	}

	if global, has := c.Get("__global"); has { // 设定全局变量
		if global, ok := global.(map[string]interface{}); ok {
			process.WithGlobal(global)
		}
	}

	v, err := process.Exec()
	if err != nil {
		return err
	}

	if data, ok := v.(map[string]interface{}); ok {
		if sid, ok := data["__sid"].(string); ok {
			c.Set("__sid", sid)
			r.Sid = sid
		}

		if global, ok := data["__global"].(map[string]interface{}); ok {
			c.Set("__global", global)
		}
	}

	return nil
}
