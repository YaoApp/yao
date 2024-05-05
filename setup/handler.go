package setup

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xun"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/widgets/app"
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

	payload := getSetting(c)

	cfg, err := getConfig()
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}

	if !Check() {
		c.JSON(403, gin.H{"code": 400, "message": "应用已安装, 删除 .env 文件和 db 目录后重试"})
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

	err = engine.Load(cfg, engine.LoadOption{})
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

	adminRoot := "yao"
	if app.Setting.AdminRoot != "" {
		adminRoot = app.Setting.AdminRoot
	}
	adminRoot = strings.Trim(adminRoot, "/")

	c.JSON(200, gin.H{
		"code":    200,
		"message": "安装成功",
		"urls":    urls,
		"port":    cfg.Port,
		"root":    adminRoot,
	})

	Complete()
}

func runCheck(c *gin.Context) {

	payload := getCheck(c)
	dbOption, err := getDBOption(map[string]map[string]string{"db": payload})
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}

	err = ValidateDB(dbOption)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 200})
}

func getCheck(c *gin.Context) map[string]string {
	var payload map[string]string
	err := c.ShouldBindJSON(&payload)
	if err != nil {
		c.JSON(500, gin.H{"code": 400, "message": err.Error()})
		c.Abort()
		return nil
	}
	return payload
}

func getSetting(c *gin.Context) map[string]map[string]string {
	var payload map[string]map[string]string
	err := c.ShouldBindJSON(&payload)
	if err != nil {
		c.JSON(500, gin.H{"code": 400, "message": err.Error()})
		c.Abort()
		return nil
	}
	return payload
}

func getENVOption(payload map[string]map[string]string) (map[string]string, error) {
	env, has := payload["env"]
	if !has {
		return nil, fmt.Errorf("缺少服务配置信息")
	}

	if env["YAO_ENV"] == "开发模式(推荐)" {
		env["YAO_ENV"] = "development"
	} else {
		env["YAO_ENV"] = "production"
	}

	if env["YAO_LANG"] == "中文" {
		env["YAO_LANG"] = "zh-cn"
	} else {
		env["YAO_LANG"] = "en-us"
	}
	return env, nil
}

func getDBOption(payload map[string]map[string]string) (map[string]string, error) {

	db, has := payload["db"]
	if !has {
		return nil, fmt.Errorf("缺少数据库配置信息")
	}

	dbOption := map[string]string{}
	switch db["type"] {
	case "", "sqlite", "sqlite3":
		dbOption["type"] = "sqlite3"
		dbOption["file"] = db["option.file"]
		return dbOption, nil

	case "mysql":
		dbOption["type"] = "mysql"
		dbOption["db"] = db["option.db"]
		dbOption["host"] = db["option.host.host"]
		dbOption["port"] = db["option.host.port"]
		dbOption["user"] = db["option.host.user"]
		dbOption["pass"] = db["option.host.pass"]
		return dbOption, nil
	}

	return nil, fmt.Errorf("数据库驱动暂不支持")
}

func getDSN(dbOption map[string]string) (string, string, error) {

	switch dbOption["type"] {
	case "", "sqlite", "sqlite3":
		root := appRoot()
		var err error
		db := filepath.Join("db", "yao.db")
		if v, has := dbOption["file"]; has {
			db = v
		}

		if !strings.HasPrefix(db, "/") {
			db = filepath.Join(root, db)
			db, err = filepath.Abs(db)
			if err != nil && !os.IsNotExist(err) {
				return "", "", err
			}
		}

		dir := filepath.Dir(db)
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil && !os.IsExist(err) {
			return "", "", err
		}

		return "sqlite3", db, nil

	case "mysql":

		db := "yao"
		if v, has := dbOption["db"]; has {
			db = v
		}

		host := "127.0.0.1"
		if v, has := dbOption["host"]; has {
			host = v
		}

		port := "3306"
		if v, has := dbOption["port"]; has {
			port = v
		}

		user := "root"
		if v, has := dbOption["user"]; has {
			user = v
		}

		pass := ""
		if v, has := dbOption["pass"]; has {
			pass = v
		}

		return "mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, pass, host, port, db), nil
	}

	return "", "", fmt.Errorf("driver does not support")

}
