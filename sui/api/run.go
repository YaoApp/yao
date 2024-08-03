package api

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/sui/core"
)

// Run the backend script, with Api prefix method
func Run(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	ctx, ok := process.Args[0].(*gin.Context)
	if !ok {
		exception.New("The context is required", 400).Throw()
		return nil
	}

	ctx.Header("Content-Type", "text/html; charset=utf-8")
	route := process.ArgsString(1)
	payload := process.ArgsMap(2)
	if route == "" {
		exception.New("The route is required", 400).Throw()
		return nil
	}

	if payload["method"] == nil {
		exception.New("The method is required", 400).Throw()
		return nil
	}

	method, ok := payload["method"].(string)
	if !ok {
		exception.New("The method must be a string", 400).Throw()
		return nil
	}

	args := []interface{}{}
	if payload["args"] != nil {
		args, ok = payload["args"].([]interface{})
		if !ok {
			exception.New("The args must be an array", 400).Throw()
			return nil
		}
	}

	ctx.Request.URL.Path = route
	r, _, err := NewRequestContext(ctx)
	if err != nil {
		exception.Err(err, 500).Throw()
		return nil
	}

	var c *core.Cache = nil
	if !r.Request.DisableCache() {
		c = core.GetCache(r.File)
	}

	if c == nil {
		c, _, err = r.MakeCache()
		if err != nil {
			log.Error("[SUI] Can't make cache, %s %s error: %s", route, method, err.Error())
			exception.New("Can't make cache, please the route and method is correct, get more information from the log.", 500).Throw()
			return nil
		}
	}

	// Guard the page
	code, err := r.Guard(c)
	if err != nil {
		exception.Err(err, code).Throw()
		return nil
	}

	if c == nil {
		exception.New("Cache not found", 500).Throw()
		return nil
	}

	if c.Script == nil {
		exception.New("Script not found", 500).Throw()
		return nil
	}

	scriptCtx, err := c.Script.NewContext(process.Sid, nil)
	if err != nil {
		exception.Err(err, 500).Throw()
		return nil
	}
	defer scriptCtx.Close()

	global := scriptCtx.Global()
	if !global.Has("Api" + method) {
		exception.New("Method %s not found", 500, method).Throw()
		return nil
	}

	res, err := scriptCtx.Call("Api"+method, args...)
	if err != nil {
		exception.Err(err, 500).Throw()
		return nil
	}

	return res
}
