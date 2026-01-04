package service

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/api"
)

// DynamicAPIHandler is a dynamic API proxy handler that dispatches requests
// to the appropriate handler based on the route table.
// This enables hot-reloading of API definitions without server restart.
func DynamicAPIHandler(c *gin.Context) {
	path := c.Param("path")
	method := c.Request.Method

	// Ensure path starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Find handler from route table
	apiDef, pathDef, handler, params, err := api.FindHandler(method, path)
	if err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "API not found"})
		c.Abort()
		return
	}

	// Set path parameters to gin.Context
	for key, value := range params {
		c.Params = append(c.Params, gin.Param{Key: key, Value: value})
	}

	// Apply guard
	guard := pathDef.Guard
	if guard == "" {
		guard = apiDef.HTTP.Guard
	}

	if guard != "" && guard != "-" {
		if err := applyGuard(c, guard); err != nil {
			return // Guard already handled the response
		}
	}

	// Execute the actual handler
	handler(c)
}

// applyGuard applies the guard middleware(s) to the request
func applyGuard(c *gin.Context, guardName string) error {
	guards := strings.Split(guardName, ",")
	for _, name := range guards {
		name = strings.TrimSpace(name)
		if name == "" || name == "-" {
			continue
		}

		// Get guard from HTTPGuards (set at Start time)
		if handler, has := api.HTTPGuards[name]; has {
			handler(c)
			if c.IsAborted() {
				return fmt.Errorf("guard aborted")
			}
			continue
		}

		// Custom guard via process
		api.ProcessGuard(name)(c)
		if c.IsAborted() {
			return fmt.Errorf("guard aborted")
		}
	}
	return nil
}

// ReloadAPIs reloads all API definitions from the apis directory
// This function is thread-safe and can be called at runtime
func ReloadAPIs() error {
	err := api.ReloadAPIs("apis")
	if err != nil {
		return err
	}
	return nil
}
