package studio

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/neo"
)

var regExcp = regexp.MustCompile(`Exception\|(\d+):(.*)`)

// Serve start the api server
func setRouter(router *gin.Engine) {

	router.Use(gin.CustomRecovery(hdRecovered), hdCORS, hdAuth)

	// DSL ReadDir, ReadFile
	router.GET("/dsl/:method", func(c *gin.Context) {
		method := strings.ToLower(c.Param("method"))
		switch method {

		case "readfile":
			name := c.Query("name")
			if name == "" {
				throw(c, 400, "file name is required")
				return
			}

			data, err := dfs.ReadFile(name)
			if err != nil {
				throw(c, 500, err.Error())
				return
			}

			res := map[string]interface{}{}
			err = jsoniter.Unmarshal(data, &res)
			if err != nil {
				throw(c, 500, err.Error())
				return
			}
			c.JSON(200, res)
			c.Done()
			return

		case "readdir":
			name := c.Query("name")
			if name == "" {
				throw(c, 400, "dir name is required")
				return
			}

			recursive := false
			if c.Query("recursive") == "1" || strings.ToLower(c.Query("recursive")) == "true" {
				recursive = true
			}
			data, err := dfs.ReadDir(name, recursive)
			if err != nil {
				throw(c, 500, err.Error())
				return
			}
			c.JSON(200, data)
			c.Done()
			return
		}

		throw(c, 404, fmt.Sprintf("%s method does not found", c.Param("method")))
	})

	// DSL WriteFile, Mkdir, MkdirAll, Remove, RemoveAll ...
	router.POST("/dsl/:method", func(c *gin.Context) {

		method := strings.ToLower(c.Param("method"))
		switch method {
		case "writefile":
			name := c.Query("name")
			if name == "" {
				throw(c, 400, "dir name is required")
				return
			}

			payload, err := io.ReadAll(c.Request.Body)
			if err != nil {
				throw(c, 500, err.Error())
				return
			}

			if payload == nil || len(payload) == 0 {
				throw(c, 500, "file content is required")
				return
			}

			length, err := dfs.WriteFile(name, payload, 0644)
			if err != nil {
				throw(c, 500, err.Error())
				return
			}

			c.JSON(200, length)
			c.Done()
			return

		case "mkdir":
			name := c.Query("name")
			if name == "" {
				throw(c, 400, "dir name is required")
				return
			}

			err := dfs.Mkdir(name, uint32(os.ModePerm))
			if err != nil {
				throw(c, 500, err.Error())
				return
			}
			c.Status(200)
			c.Done()
			return

		case "mkdirall":
			name := c.Query("name")
			if name == "" {
				throw(c, 400, "dir name is required")
				return
			}

			err := dfs.MkdirAll(name, uint32(os.ModePerm))
			if err != nil {
				throw(c, 500, err.Error())
				return
			}
			c.Status(200)
			c.Done()
			return

		case "remove":
			name := c.Query("name")
			if name == "" {
				throw(c, 400, "name is required")
				return
			}

			err := dfs.Remove(name)
			if err != nil {
				throw(c, 500, err.Error())
				return
			}
			c.Status(200)
			c.Done()
			return

		case "removeall":
			name := c.Query("name")
			if name == "" {
				throw(c, 400, "name is required")
				return
			}

			err := dfs.RemoveAll(name)
			if err != nil {
				throw(c, 500, err.Error())
				return
			}
			c.Status(200)
			c.Done()
			return
		}

		throw(c, 404, fmt.Sprintf("%s method does not found", c.Param("method")))
	})

	// Cloud Functions
	router.POST("/service/:name", func(c *gin.Context) {

		name := c.Param("name")
		if name == "" {
			throw(c, 400, "service name is required")
			return
		}

		service := c.Param("name")

		payload, err := io.ReadAll(c.Request.Body)
		if err != nil {
			throw(c, 500, err.Error())
			return
		}

		if payload == nil || len(payload) == 0 {
			throw(c, 400, "file content is required")
			return
		}

		var fun cfunc
		err = jsoniter.Unmarshal(payload, &fun)
		if err != nil {
			throw(c, 500, err.Error())
			return
		}

		sid, _ := c.Get("__sid")
		script, err := v8.SelectRoot(service)
		if err != nil {
			throw(c, 500, err.Error())
			return
		}

		ctx, err := script.NewContext(fmt.Sprintf("%v", sid), nil)
		if err != nil {
			code := 500
			message := err.Error()
			match := regExcp.FindStringSubmatch(message)
			if len(match) > 0 {
				code, err = strconv.Atoi(match[1])
				if err == nil {
					message = strings.TrimSpace(match[2])
				}
			}
			throw(c, code, message)
			return
		}
		defer ctx.Close()

		res, err := ctx.Call(fun.Method, fun.Args...)
		if err != nil {
			code := 500
			message := err.Error()
			match := regExcp.FindStringSubmatch(message)
			if len(match) > 0 {
				code, err = strconv.Atoi(match[1])
				if err == nil {
					message = strings.TrimSpace(match[2])
				}
			}
			throw(c, code, message)
			return
		}

		c.JSON(200, res)
		c.Done()
	})

	// Neo API for studio
	if neo.Neo != nil {
		neo.Neo.API(router, "/neo")
	}

}

func throw(c *gin.Context, code int, message string) {
	c.JSON(code, map[string]interface{}{
		"message": message,
		"code":    code,
	})
	c.Done()
}
