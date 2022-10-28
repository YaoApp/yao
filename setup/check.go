package setup

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/yao/config"
)

// Check setup programe
func Check() bool {

	root := appRoot()

	appfile := filepath.Join(root, "app.json")
	if _, err := os.Stat(appfile); err != nil && os.IsNotExist(err) {
		return true
	}

	envfile := filepath.Join(root, ".env")
	if _, err := os.Stat(envfile); err != nil && os.IsNotExist(err) {
		cfg, err := getConfig()
		if err != nil || !hasInstalled(cfg) {
			return true
		}
	}

	return false
}

// Validate db link
func Validate() (err error) {

	root := appRoot()
	path := filepath.Join(root, "db", "yao.db")

	data := []byte(fmt.Sprintf(`{
		"type": "sqlite3",
		"options": {
			"file": "%s"
		}
	}`, path))

	data = []byte(`{
		"type": "mysql",
		"options": {
			"db": "test",
			"hosts": [{ "host": "127.0.0.1", "user":"root", "pass":"123456" }]
		}
	}`)

	_, err = connector.Load(string(data), "test")
	return err

}

func appRoot() string {

	root := os.Getenv("YAO_ROOT")
	if root == "" {
		path, err := os.Getwd()
		if err != nil {
			printError("无法获取应用目录: %s", err)
			Stop()
		}
		root = path
	}

	root, err := filepath.Abs(root)
	if err != nil {
		printError("无法获取应用目录: %s", err)
		Stop()
	}

	return root
}

func getConfig() (config.Config, error) {
	root := appRoot()
	envfile := filepath.Join(root, ".env")
	cfg := config.LoadFrom(envfile)
	return cfg, nil
}

func hasInstalled(cfg config.Config) bool {

	switch cfg.DB.Driver {

	case "sqlite3":
		if cfg.DB.Primary != nil {
			return true
		}
		break

	case "mysql":
		if cfg.DB.Primary != nil {
			return true
		}
		break
	}

	return false
}
