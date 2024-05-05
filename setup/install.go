package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/process"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/data"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/studio"
	"github.com/yaoapp/yao/widgets/app"
)

// Install app install
//
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
//
//	{
//		"env": {
//		  "YAO_LANG": "中文",
//		  "YAO_ENV": "开发模式(推荐)",
//		  "YAO_PORT": "5099",
//		  "YAO_STUDIO_PORT": "5077"
//		},
//		"db": {
//		  "type": "mysql",
//		  "option.db": "yao",
//		  "option.host.host": "127.0.0.1",
//		  "option.host.port": "3306",
//		  "option.host.user": "root",
//		  "option.host.pass": "123456"
//		}
//	}
func Install(payload map[string]map[string]string) error {

	dbOption, err := getDBOption(payload)
	if err != nil {
		return err
	}

	err = ValidateDB(dbOption)
	if err != nil {
		return err
	}

	envOption, err := getENVOption(payload)
	if err != nil {
		return err
	}

	err = ValidateHosting(envOption)
	if err != nil {
		return err
	}

	root := appRoot()
	err = makeService(root, "0.0.0.0", envOption["YAO_PORT"], envOption["YAO_STUDIO_PORT"], envOption["YAO_LANG"])
	if err != nil {
		return err
	}

	err = makeDB(root, dbOption)
	if err != nil {
		return err
	}

	err = makeSession(root)
	if err != nil {
		return err
	}

	err = makeLog(root)
	if err != nil {
		return err
	}

	err = makeDirs(root)
	if err != nil {
		return err
	}

	err = makeInit(root)
	if err != nil {
		return err
	}

	cfg, err := getConfig()
	if err != nil {
		return err
	}

	// Load engine
	err = engine.Load(cfg, engine.LoadOption{
		Action: "install",
	})

	if err != nil {
		return err
	}

	defer func() {
		engine.Unload()
		time.Sleep(time.Millisecond * 200)
	}()

	// Load Studio
	err = studio.Load(cfg)
	if err != nil {
		return err
	}

	//  Migrage & Setup
	err = makeMigrate(root, cfg)
	if err != nil {
		return err
	}

	err = makeSetup(root, cfg)
	if err != nil {
		return err
	}

	return nil
}

func makeService(root string, host string, port string, studioPort string, lang string) error {
	file := filepath.Join(root, ".env")
	err := envSet(file, "YAO_HOST", host)
	if err != nil {
		return err
	}

	err = envSet(file, "YAO_PORT", port)
	if err != nil {
		return err
	}

	err = envSet(file, "YAO_STUDIO_PORT", studioPort)
	if err != nil {
		return err
	}

	return envSet(file, "YAO_LANG", lang)
}

func makeDB(root string, option map[string]string) error {

	driver, dsn, err := getDSN(option)
	if driver != "mysql" && driver != "sqlite3" {
		return fmt.Errorf("数据库驱动应该为: mysql/sqlite3")
	}

	file := filepath.Join(root, ".env")
	if err != nil {
		return err
	}

	err = envSet(file, "YAO_DB_DRIVER", driver)
	if err != nil {
		return err
	}

	return envSet(file, "YAO_DB_PRIMARY", dsn)
}

func makeSession(root string) error {
	file := filepath.Join(root, ".env")
	if has, _ := envHas(file, "YAO_SESSION_STORE"); has {
		return nil
	}

	if has, _ := envHas(file, "YAO_SESSION_FILE"); has {
		return nil
	}

	err := os.MkdirAll(filepath.Join(root, "data"), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return err
	}

	err = envSet(file, "YAO_SESSION_STORE", "file")
	if err != nil {
		return err
	}

	ssfile := filepath.Join(root, "db", ".session")
	return envSet(file, "YAO_SESSION_FILE", ssfile)
}

func makeLog(root string) error {
	file := filepath.Join(root, ".env")
	if has, _ := envHas(file, "YAO_LOG_MODE"); has {
		return nil
	}

	if has, _ := envHas(file, "YAO_LOG"); has {
		return nil
	}

	err := os.MkdirAll(filepath.Join(root, "data"), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return err
	}

	err = envSet(file, "YAO_LOG_MODE", "TEXT")
	if err != nil {
		return err
	}

	logfile := filepath.Join(root, "logs", "application.log")
	return envSet(file, "YAO_LOG", logfile)
}

func makeDirs(root string) error {

	if !appSourceExists() {
		err := os.MkdirAll(filepath.Join(root, "public"), os.ModePerm)
		if err != nil && !os.IsExist(err) {
			return err
		}
	}

	err := os.MkdirAll(filepath.Join(root, "logs"), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return err
	}

	err = os.MkdirAll(filepath.Join(root, "db"), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return err
	}

	return nil
}

func makeInit(root string) error {

	if appSourceExists() {
		return nil
	}

	files := data.AssetNames()
	for _, file := range files {
		if strings.HasPrefix(file, "init/") {
			dst := filepath.Join(root, strings.TrimPrefix(file, "init/"))
			content, err := data.Read(file)
			if err != nil {
				return err
			}

			if _, err := os.Stat(dst); err == nil { // exists
				log.Error("[setup] %s exists", dst)
				continue
			}

			dir := filepath.Dir(dst)
			if err := os.MkdirAll(dir, os.ModePerm); err != nil {
				return err
			}

			if err = os.WriteFile(dst, content, 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

func makeMigrate(root string, cfg config.Config) error {

	// Do Stuff Here
	for _, mod := range model.Models {
		has, err := mod.HasTable()
		if err != nil {
			return err
		}

		if has {
			log.Warn("%s (%s) table already exists", mod.ID, mod.MetaData.Table.Name)
			continue
		}

		err = mod.Migrate(false)
		if err != nil {
			return err
		}
	}

	return nil
}

func makeSetup(root string, cfg config.Config) error {

	if app.Setting != nil && app.Setting.Setup != "" {

		if strings.HasPrefix(app.Setting.Setup, "studio.") {
			names := strings.Split(app.Setting.Setup, ".")
			if len(names) < 3 {
				return fmt.Errorf("setup studio script %s error", app.Setting.Setup)
			}

			service := strings.Join(names[1:len(names)-1], ".")
			method := names[len(names)-1]

			script, err := v8.SelectRoot(service)
			if err != nil {
				return err
			}

			sid := uuid.NewString()
			ctx, err := script.NewContext(fmt.Sprintf("%v", sid), nil)
			if err != nil {
				return err
			}
			defer ctx.Close()

			_, err = ctx.Call(method)
			return err
		}

		p, err := process.Of(app.Setting.Setup, cfg)
		if err != nil {
			return err
		}
		_, err = p.Exec()
		if err != nil {
			return err
		}
	}

	return nil
}
