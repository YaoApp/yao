package setup

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yaoapp/yao/config"
)

// InYaoApp Check if the current directory is a yao app
func InYaoApp(root string) bool {
	// Check current directory and parent directories
	for root != "/" {
		if IsYaoApp(root) {
			return true
		}
		root = filepath.Dir(root)
	}
	return false
}

// IsYaoApp Check if the directory is a yao app
func IsYaoApp(root string) bool {
	appfiles := []string{"app.yao", "app.json", "app.jsonc"}
	yaoapp := false
	for _, appfile := range appfiles {
		appfile = filepath.Join(root, appfile)
		if _, err := os.Stat(appfile); err == nil {
			yaoapp = true
			break
		}
	}
	return yaoapp
}

// IsEmptyDir Check if the directory is empty
func IsEmptyDir(dir string) bool {
	f, err := os.Open(dir)
	if err != nil {
		fmt.Println("Can't open the directory: ", err)
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
