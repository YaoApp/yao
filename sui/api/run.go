package api

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/openapi/oauth/acl"
	"github.com/yaoapp/yao/sui/core"
)

var configs = map[string]*core.PageConfig{}
var chConfig = make(chan *core.PageConfig, 1)

func init() {
	go configWriter()
}

// Run the backend script, with Api prefix method
func Run(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	ctx, ok := process.Args[0].(*gin.Context)
	if !ok {
		exception.New("The context is required", 400).Throw()
		return nil
	}

	ctx.Header("Content-Type", "application/json; charset=utf-8")
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

	r, _, err := NewRequestContext(ctx)
	if err != nil {
		exception.Err(err, 500).Throw()
		return nil
	}

	// Load the script
	file := filepath.Join("/public", route)

	// Get the page config
	cfg, err := getPageConfig(file, r.Request.DisableCache())
	if err != nil {

		if err.Error() == "The config file not found" {
			exception.New("The page not found (%s)", 404, route).Throw()
			return nil
		}

		log.Error("Can't load the page config (%s), %s", route, err.Error())
		exception.New("Can't load the page config (%s), get more information from the log.", 500, route).Throw()
		return nil
	}

	// Config and guard
	prefix := "Api"
	if cfg != nil {
		_, err := r.apiGuard(method, cfg.API)
		if err != nil {
			log.Error("Guard error: %s", err.Error())
			r.context.Done()
			return nil
		}

		// Custom prefix
		if cfg.API != nil && cfg.API.Prefix != "" {
			prefix = cfg.API.Prefix
		}
	}

	script, err := core.LoadScript(file, r.Request.DisableCache())
	if err != nil {
		exception.New("Can't load the script (%s), get more information from the log.", 500, route).Throw()
		return nil
	}

	if script == nil {
		exception.New("Script not found (%s)", 404, route)
		return nil
	}

	scriptCtx, err := script.NewContext(r.Sid, nil)
	if err != nil {
		return nil
	}
	defer scriptCtx.Close()

	// Pass authorized info to V8 context
	// Priority: 1. From Request (set by SUI guard), 2. From Process (set by GOU handler)
	if len(r.Authorized) > 0 {
		scriptCtx.WithAuthorized(r.Authorized)
	} else if authorized := process.GetAuthorized(); authorized != nil {
		if authMap := authorized.AuthorizedToMap(); len(authMap) > 0 {
			scriptCtx.WithAuthorized(authMap)
		}
	}

	global := scriptCtx.Global()
	if !global.Has(prefix + method) {
		exception.New("Method %s not found", 500, method).Throw()
		return nil
	}

	res, err := scriptCtx.Call(prefix+method, args...)
	if err != nil {
		exception.Err(err, 500).Throw()
		return nil
	}

	return res
}

// getPageConfig get the page config
func getPageConfig(file string, disableCache ...bool) (*core.PageConfig, error) {

	// LOAD FROM CACHE
	base := strings.TrimSuffix(strings.TrimSuffix(file, ".sui"), ".jit")
	if disableCache == nil || !disableCache[0] {
		if cfg, has := configs[base]; has {
			return cfg, nil
		}
	}

	file = base + ".cfg"
	if exist, _ := application.App.Exists(file); !exist {
		return nil, fmt.Errorf("The config file not found")
	}

	source, err := application.App.Read(file)
	if err != nil {
		return nil, err
	}

	cfg := core.PageConfig{}
	err = jsoniter.Unmarshal(source, &cfg)
	if err != nil {
		return nil, err
	}

	// Save to cache
	go func() { chConfig <- &cfg }()
	return &cfg, nil
}

func (r *Request) apiGuard(method string, api *core.PageAPI) (int, error) {
	if api == nil {
		return 200, nil
	}

	guard := api.DefaultGuard
	if api.Guards != nil {
		if g, has := api.Guards[method]; has {
			guard = g
		}
	}

	if guard == "" || guard == "-" {
		return 200, nil
	}

	// Build in guard
	if guardFunc, has := Guards[guard]; has {
		err := guardFunc(r)
		if err != nil {
			return 403, err
		}

		// For OAuth guard, perform ACL check after authentication
		if guard == "oauth" {
			if err := r.enforceACL(); err != nil {
				return 403, err
			}
		}

		return 200, nil
	}

	// Developer custom guard
	err := r.processGuard(guard)
	if err != nil {
		return 403, err
	}

	return 200, nil
}

// enforceACL performs ACL permission check for API calls
func (r *Request) enforceACL() error {
	if r.context == nil {
		return nil
	}

	// Skip if ACL is not enabled
	if acl.Global == nil || !acl.Global.Enabled() {
		return nil
	}

	// Enforce ACL
	ok, err := acl.Global.Enforce(r.context)
	if err != nil {
		log.Error("[SUI] ACL enforcement failed: %v", err)
		return err
	}

	if !ok {
		return fmt.Errorf("Access denied")
	}

	return nil
}

func configWriter() {
	for config := range chConfig {
		configs[config.Root] = config
	}
}
