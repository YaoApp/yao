package lang

import (
	"os"
	"path/filepath"

	"github.com/yaoapp/gou/lang"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
)

func init() {
	lang.RegisterWidget("tables", "table")
	lang.RegisterWidget("charts", "chart")
	lang.RegisterWidget("kanban", "page")
	lang.RegisterWidget("screen", "page")
	lang.RegisterWidget("pages", "page")
}

// Load language packs
func Load(cfg config.Config) error {
	root := filepath.Join(cfg.Root, "langs")
	err := lang.Load(root)
	if err != nil {
		return err
	}

	name := os.Getenv("YAO_LANG")
	if name != "" {
		if _, has := lang.Dicts[name]; !has {
			log.Error("The language pack %s does not found", name)
			return nil
		}
		lang.Pick(name).AsDefault()
	}

	return nil
}
