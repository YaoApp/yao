package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/xun/capsule"
)

var cfg Config

func TestMain(m *testing.M) {

	cfg = NewConfig()

	// 数据库连接
	capsule.AddConn("primary", "mysql", cfg.Database.Primary[0]).SetAsGlobal()

	// 加密密钥
	gou.LoadCrypt(fmt.Sprintf(`{"key":"%s"}`, cfg.Database.AESKey), "AES")
	gou.LoadCrypt(`{}`, "PASSWORD")

	// 加载数据
	Load(cfg)

	// Run test suites
	exitVal := m.Run()

	// we can do clean up code here
	gou.KillPlugins()

	os.Exit(exitVal)
}
