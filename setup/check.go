package setup

import (
	"os"
	"path/filepath"

	"github.com/yaoapp/yao/config"
)

// SourceExists check if the app source exists
func SourceExists() bool {
	return appSourceExists()
}

func appSourceExists() bool {

	root := appRoot()
	if isEmptyDir(root) {
		return false
	}

	// check app.yao/app.json/app.jsonc
	appfiles := []string{"app.yao", "app.json", "app.jsonc"}
	exist := false
	for _, appfile := range appfiles {
		appfile = filepath.Join(root, appfile)
		if _, err := os.Stat(appfile); err == nil {
			exist = true
			break
		}
	}

	return exist
}

func isEmptyDir(dir string) bool {
	f, err := os.Open(dir)
	if err != nil {
		return true
	}
	defer f.Close()

	files, err := f.Readdir(0)
	if err != nil {
		return true
	}
	return len(files) == 0
}

func appRoot() string {

	root := os.Getenv("YAO_ROOT")
	if root == "" {
		path, err := os.Getwd()
		if err != nil {
			printError("Can't get the application directory: %s", err)
		}
		root = path
	}

	root, err := filepath.Abs(root)
	if err != nil {
		printError("Can't get the application directory: %s", err)
	}

	return root
}

func getConfig() (config.Config, error) {
	root := appRoot()
	envfile := filepath.Join(root, ".env")
	cfg := config.LoadFrom(envfile)
	return cfg, nil
}
