package setup

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xun"
)

// recovered custom recovered
func recovered(c *gin.Context, recovered interface{}) {

	var code = http.StatusInternalServerError

	if err, ok := recovered.(string); ok {
		c.JSON(code, xun.R{
			"code":    code,
			"message": fmt.Sprintf("%s", err),
		})
	} else if err, ok := recovered.(exception.Exception); ok {
		code = err.Code
		c.JSON(code, xun.R{
			"code":    code,
			"message": err.Message,
		})
	} else if err, ok := recovered.(*exception.Exception); ok {
		code = err.Code
		c.JSON(code, xun.R{
			"code":    code,
			"message": err.Message,
		})
	} else {
		c.JSON(code, xun.R{
			"code":    code,
			"message": fmt.Sprintf("%v", recovered),
		})
	}

	c.AbortWithStatus(code)
}

//	{
//	  "env": {
//	    "YAO_LANG": "中文",
//	    "YAO_ENV": "开发模式(推荐)",
//	    "YAO_PORT": "5099",
//	    "YAO_STUDIO_PORT": "5077"
//	  },
//	  "db": {
//	    "type": "sqlite",
//	    "option.file": "db/yao.db"
//	  }
//	}
func runSetup(c *gin.Context) {

	payload := getPayload(c)

	cfg, err := getConfig()
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}

	if hasInstalled(cfg) {
		c.JSON(403, gin.H{"code": 400, "message": "应用已安装, 删除 .env 文件后重试"})
		return
	}

	err = Install(payload)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}

	// Reload Config
	cfg, err = getConfig()
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}

	// Return
	urls, err := AdminURL(cfg)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"code":    200,
		"message": "安装成功",
		"urls":    urls,
	})

	Complete()
}

func runCheck(c *gin.Context) {
	time.Sleep(2 * time.Second)
	c.JSON(200, gin.H{"code": 200})
}

func getPayload(c *gin.Context) map[string]map[string]string {
	var payload map[string]map[string]string
	err := c.ShouldBindJSON(&payload)
	if err != nil {
		c.JSON(500, gin.H{"code": 400, "message": err.Error()})
		return nil
	}

	return payload
}
