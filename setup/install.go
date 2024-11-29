package setup

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/data"
	"github.com/yaoapp/yao/widgets/app"
)

// Install the app to the root directory
func Install(root string) error {

	// Copy the init source files
	err := makeInit(root)
	if err != nil {
		return err
	}

	return nil
}

// Initialize the installed app
func Initialize(root string, cfg config.Config) error {

	// Migration
	err := makeMigrate()
	if err != nil {
		return err
	}

	// Execute the setup hook
	err = makeSetup(cfg)
	if err != nil {
		return err
	}

	return nil
}

func makeInit(root string) error {

	if !IsEmptyDir(root) {
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

func makeMigrate() error {

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

func makeSetup(cfg config.Config) error {

	if app.Setting != nil && app.Setting.Setup != "" {

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
