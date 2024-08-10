package api

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/process"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/helper"
	"rogchap.com/v8go"
)

// Guards middlewares
var Guards = map[string]func(c *Request) error{
	"bearer-jwt":   guardBearerJWT,   // Bearer JWT
	"query-jwt":    guardQueryJWT,    // Get JWT Token from query string  "__tk"
	"cookie-jwt":   guardCookieJWT,   // Get JWT Token from cookie "__tk"
	"cookie-trace": guardCookieTrace, // Set sid cookie
}

// JWT Bearer JWT
func guardBearerJWT(r *Request) error {
	if r.context == nil {
		return fmt.Errorf("Not authenticated")
	}
	c := r.context
	tokenString := c.Request.Header.Get("Authorization")
	tokenString = strings.TrimSpace(strings.TrimPrefix(tokenString, "Bearer "))
	if tokenString == "" {
		c.JSON(401, gin.H{"code": 401, "message": "Not authenticated"})
		c.Abort()
		return fmt.Errorf("Not authenticated")
	}

	claims := helper.JwtValidate(tokenString)
	c.Set("__sid", claims.SID)
	r.Sid = claims.SID
	return nil
}

// JWT Bearer JWT
func guardCookieJWT(r *Request) error {
	if r.context == nil {
		return fmt.Errorf("Context is nil")
	}
	c := r.context

	tokenString, err := c.Cookie("__tk")
	if err != nil {
		// c.JSON(403, gin.H{"code": 403, "message": "No permission"})
		// c.Abort()
		return fmt.Errorf("Not authenticated")
	}

	if tokenString == "" {
		// c.JSON(403, gin.H{"code": 403, "message": "No permission"})
		// c.Abort()
		return fmt.Errorf("Not authenticated")
	}

	claims := helper.JwtValidate(tokenString)
	c.Set("__sid", claims.SID)
	r.Sid = claims.SID
	return nil
}

func guardCookieTrace(r *Request) error {
	if r.context == nil {
		return fmt.Errorf("Context is nil")
	}

	c := r.context
	sid, err := c.Cookie("sid")
	if err != nil {
		sid = uuid.New().String()
		c.SetCookie("sid", sid, 0, "/", "", false, true)
		c.Set("__sid", sid)
		r.Sid = sid
		return nil
	}
	c.Set("__sid", sid)
	r.Sid = sid
	return nil
}

// JWT Bearer JWT
func guardQueryJWT(r *Request) error {
	if r.context == nil {
		return fmt.Errorf("Not authenticated")
	}
	c := r.context

	tokenString := c.Query("__tk")
	if tokenString == "" {
		c.JSON(401, gin.H{"code": 401, "message": "Not authenticated"})
		c.Abort()
		return fmt.Errorf("Not authenticated")
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

	if strings.HasPrefix(name, "scripts.") {
		return r.scriptGuardExec(c, name, args)
	}
	return r.processGuardExec(c, name, args)
}

func (r *Request) scriptGuardExec(c *gin.Context, name string, args []interface{}) error {

	namer := strings.Split(strings.TrimPrefix(name, "scripts."), ".")
	id := strings.Join(namer[:len(namer)-1], ".")
	method := namer[len(namer)-1]

	script, err := v8.Select(id)
	if err != nil {
		c.JSON(403, gin.H{"code": 403, "message": err.Error()})
		c.Abort()
		return err
	}

	sid := ""
	global := map[string]interface{}{}
	if v, has := c.Get("__sid"); has { // 设定会话ID
		if v, ok := v.(string); ok {
			sid = v
		}
	}

	if v, has := c.Get("__global"); has { // 设定全局变量
		if v, ok := v.(map[string]interface{}); ok {
			global = v
		}
	}

	ctx, err := script.NewContext(sid, global)
	if err != nil {
		c.JSON(403, gin.H{"code": 403, "message": fmt.Sprintf("Guard: %s %s", name, err.Error())})
		c.Abort()
		return err
	}
	defer ctx.Close()

	// Should be refector after the runtime refector
	// Add the context object
	ctx.WithFunction("SetSid", func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		if len(info.Args()) < 1 {
			log.Error("SetSid no sid")
			return v8go.Undefined(info.Context().Isolate())
		}

		sid, err := bridge.GoValue(info.Args()[0], info.Context())
		if err != nil {
			log.Error("SetSid %s", err.Error())
			return v8go.Undefined(info.Context().Isolate())
		}

		c.Set("__sid", sid)
		return v8go.Undefined(info.Context().Isolate())
	})

	ctx.WithFunction("SetGlobal", func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		if len(info.Args()) < 1 {
			log.Error("SetGlobal no global")
			return v8go.Undefined(info.Context().Isolate())
		}

		global, err := bridge.GoValue(info.Args()[0], info.Context())
		if err != nil {
			log.Error("SetGlobal %s", err.Error())
			return v8go.Undefined(info.Context().Isolate())
		}

		if global, ok := global.(map[string]interface{}); ok {
			c.Set("__global", global)
		}

		return v8go.Undefined(info.Context().Isolate())
	})

	ctx.WithFunction("Redirect", func(info *v8go.FunctionCallbackInfo) *v8go.Value {

		if len(info.Args()) < 2 {
			log.Error("Redirect no url")
			return v8go.Undefined(info.Context().Isolate())
		}

		var ok = false
		var code = 0
		var url = ""
		v, err := bridge.GoValue(info.Args()[0], info.Context())
		if err != nil {
			log.Error("Redirect %s", err.Error())
			return v8go.Undefined(info.Context().Isolate())
		}

		if code, ok = v.(int); !ok {
			log.Error("Redirect code error")
			return v8go.Undefined(info.Context().Isolate())
		}

		v, err = bridge.GoValue(info.Args()[1], info.Context())
		if err != nil {
			log.Error("Redirect %s", err.Error())
			return v8go.Undefined(info.Context().Isolate())
		}

		if url, ok = v.(string); !ok {
			log.Error("Redirect url error")
			return v8go.Undefined(info.Context().Isolate())
		}

		c.Redirect(code, url)
		c.Abort()
		return nil
	})

	ctx.WithFunction("Abort", func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		c.Abort()

		return nil
	})

	ctx.WithFunction("Cookie", func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		if len(info.Args()) < 1 {
			log.Error("SetGlobal no global")
			return v8go.Undefined(info.Context().Isolate())
		}

		name, err := bridge.GoValue(info.Args()[0], info.Context())
		if err != nil {
			log.Error("Cookie %s", err.Error())
			return v8go.Undefined(info.Context().Isolate())
		}

		if name, ok := name.(string); ok {
			value, err := c.Cookie(name)
			if err != nil {
				log.Error("Cookie %s", err.Error())
				return v8go.Undefined(info.Context().Isolate())
			}

			jsValue, err := bridge.JsValue(info.Context(), value)
			if err != nil {
				log.Error("Cookie %s", err.Error())
				return v8go.Undefined(info.Context().Isolate())
			}
			return jsValue
		}

		return v8go.Undefined(info.Context().Isolate())

	})

	// This function should be refector after the next version
	ctx.WithFunction("SetCookie", func(info *v8go.FunctionCallbackInfo) *v8go.Value {

		if len(info.Args()) < 7 {
			log.Error("SetCookie no enough params")
			return v8go.Undefined(info.Context().Isolate())
		}

		var ok = false
		var name = ""
		var value = ""
		var maxAge = 0
		var path = ""
		var domain = ""
		var secure = false
		var httpOnly = false

		v, err := bridge.GoValue(info.Args()[0], info.Context())
		if err != nil {
			log.Error("SetCookie %s", err.Error())
			return v8go.Undefined(info.Context().Isolate())
		}

		if name, ok = v.(string); !ok {
			log.Error("SetCookie name error")
			return v8go.Undefined(info.Context().Isolate())
		}

		v, err = bridge.GoValue(info.Args()[1], info.Context())
		if err != nil {
			log.Error("SetCookie %s", err.Error())
			return v8go.Undefined(info.Context().Isolate())
		}

		if value, ok = v.(string); !ok {
			log.Error("SetCookie value error")
			return v8go.Undefined(info.Context().Isolate())
		}

		v, err = bridge.GoValue(info.Args()[2], info.Context())
		if err != nil {
			log.Error("SetCookie %s", err.Error())
			return v8go.Undefined(info.Context().Isolate())
		}

		if maxAge, ok = v.(int); !ok {
			log.Error("SetCookie maxAge error")
			return v8go.Undefined(info.Context().Isolate())
		}

		v, err = bridge.GoValue(info.Args()[3], info.Context())
		if err != nil {
			log.Error("SetCookie %s", err.Error())
			return v8go.Undefined(info.Context().Isolate())
		}
		if path, ok = v.(string); !ok {
			log.Error("SetCookie path error")
			return v8go.Undefined(info.Context().Isolate())
		}

		v, err = bridge.GoValue(info.Args()[4], info.Context())
		if err != nil {
			log.Error("SetCookie %s", err.Error())
			return v8go.Undefined(info.Context().Isolate())
		}

		if domain, ok = v.(string); !ok {
			log.Error("SetCookie domain error")
			return v8go.Undefined(info.Context().Isolate())
		}

		v, err = bridge.GoValue(info.Args()[5], info.Context())
		if err != nil {
			log.Error("SetCookie %s", err.Error())
			return v8go.Undefined(info.Context().Isolate())
		}

		if secure, ok = v.(bool); !ok {
			log.Error("SetCookie secure error")
			return v8go.Undefined(info.Context().Isolate())
		}

		v, err = bridge.GoValue(info.Args()[6], info.Context())
		if err != nil {
			log.Error("SetCookie %s", err.Error())
			return v8go.Undefined(info.Context().Isolate())
		}

		if httpOnly, ok = v.(bool); !ok {
			log.Error("SetCookie httpOnly error")
			return v8go.Undefined(info.Context().Isolate())
		}

		c.SetCookie(name, value, maxAge, path, domain, secure, httpOnly)
		return nil
	})

	_, err = ctx.Call(method, args...)
	if err != nil {

		message := err.Error()
		if strings.HasPrefix(message, "Exception|") {
			parts := strings.Split(message, ": ")
			if len(parts) > 1 {
				codestr := strings.TrimPrefix(parts[0], "Exception|")
				message := parts[1]
				code := 403
				if codestr != "" {
					if v, err := strconv.Atoi(codestr); err == nil {
						code = v
					}
				}
				c.JSON(code, gin.H{"code": code, "message": message})
				c.Abort()
				return err
			}
		}

		c.JSON(403, gin.H{"code": 403, "message": message})
		c.Abort()
		return err
	}
	return nil
}

func (r *Request) processGuardExec(c *gin.Context, name string, args []interface{}) error {
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
