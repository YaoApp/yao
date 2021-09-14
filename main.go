package main

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/utils"
	"github.com/yaoapp/xiang/cmd"
)

// 主程序
func main() {
	cmd.Execute()

	// fmt.Printf("象传应用引擎 %s %s\n", VERSION, DOMAIN)
	// cfg := NewConfig()

	// // 加载脚本
	// capsule.AddConn("primary", "mysql", cfg.Database.Primary[0]).SetAsGlobal()

	// // 加密密钥
	// gou.LoadCrypt(fmt.Sprintf(`{"key":"%s"}`, cfg.Database.AESKey), "AES")
	// gou.LoadCrypt(`{}`, "PASSWORD")

	// // 加载模型
	// Load(cfg)

	// // 启动服务
	// for _, api := range gou.APIs {
	// 	for _, p := range api.HTTP.Paths {
	// 		utils.Dump(api.Name + ":" + p.Path)
	// 	}

	// }
	// gou.ServeHTTP(gou.Server{
	// 	Host:   cfg.Service.Host,
	// 	Port:   cfg.Service.Port,
	// 	Allows: cfg.Service.Allow,
	// 	Root:   "/api",
	// })
}

// Migrate 数据迁移
func Migrate() {
	for name, mod := range gou.Models {
		utils.Dump(name)
		mod.Migrate(true)
	}
}
