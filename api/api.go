package api

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load 加载API
func Load(cfg config.Config) error {
	if share.BUILDIN {
		return LoadBuildIn("apis", "")
	}
	return LoadFrom(filepath.Join(cfg.Root, "apis"), "")
}

// LoadFrom 从特定目录加载
func LoadFrom(dir string, prefix string) error {
	if share.DirNotExists(dir) {
		return fmt.Errorf("%s does not exists", dir)
	}

	messages := []string{}
	err := share.Walk(dir, ".http.json", func(root, filename string) {
		name := prefix + share.SpecName(root, filename)
		content := share.ReadFile(filename)
		_, err := gou.LoadAPIReturn(string(content), name, "bearer-jwt")
		if err != nil {
			log.With(log.F{"root": root, "file": filename}).Error(err.Error())
			messages = append(messages, fmt.Sprintf("%s %s", name, err.Error()))
		}
	})

	// Load WebSocket Server
	err = share.Walk(dir, ".ws.json", func(root, filename string) {
		name := prefix + share.SpecName(root, filename)
		content := share.ReadFile(filename)
		_, err := gou.LoadWebSocketServer(string(content), name)
		if err != nil {
			log.With(log.F{"root": root, "file": filename}).Error(err.Error())
			messages = append(messages, fmt.Sprintf("%s %s", name, err.Error()))
		}
	})

	if len(messages) > 0 {
		return fmt.Errorf("%s", strings.Join(messages, ";"))
	}

	return err
}

// LoadBuildIn 从制品中读取
func LoadBuildIn(dir string, prefix string) error {
	return nil
}
