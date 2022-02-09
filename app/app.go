package app

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/data"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/xfs"
)

// Load 加载应用信息
func Load(cfg config.Config) {
	Init(cfg)
	LoadInfo(cfg.Root)
}

// Init 应用初始化
func Init(cfg config.Config) {

	// // UI文件目录
	// if _, err := os.Stat(cfg.RootUI); os.IsNotExist(err) {
	// 	err := os.MkdirAll(cfg.RootUI, os.ModePerm)
	// 	if err != nil {
	// 		xlog.Printf("创建目录失败(%s) %s", cfg.RootUI, err)
	// 		os.Exit(1)
	// 	}

	// 	content, err := data.Asset("xiang/data/index.html")
	// 	if err != nil {
	// 		xlog.Printf("读取文件失败(%s) %s", cfg.RootUI, err)
	// 		os.Exit(1)
	// 	}

	// 	err = ioutil.WriteFile(filepath.Join(cfg.RootUI, "/index.html"), content, os.ModePerm)
	// 	if err != nil {
	// 		xlog.Printf("复制默认文件失败(%s) %s", cfg.RootUI, err)
	// 		os.Exit(1)
	// 	}
	// }

	// // 数据库目录
	// if _, err := os.Stat(cfg.RootDB); os.IsNotExist(err) {
	// 	err := os.MkdirAll(cfg.RootDB, os.ModePerm)
	// 	if err != nil {
	// 		xlog.Printf("创建目录失败(%s) %s", cfg.RootDB, err)
	// 		os.Exit(1)
	// 	}
	// }

	// // 文件数据目录
	// if _, err := os.Stat(cfg.RootData); os.IsNotExist(err) {
	// 	err := os.MkdirAll(cfg.RootData, os.ModePerm)
	// 	if err != nil {
	// 		xlog.Printf("创建目录失败(%s) %s", cfg.RootData, err)
	// 		os.Exit(1)
	// 	}
	// }
}

// LoadInfo 应用信息
func LoadInfo(root string) {
	info := defaultInfo()
	fs := xfs.New(root)
	if fs.MustExists("/app.json") {
		err := jsoniter.Unmarshal(fs.MustReadFile("/app.json"), &info)
		if err != nil {
			exception.New("解析应用失败 %s", 500, err).Throw()
		}
	}

	if fs.MustExists("/xiang/icons/icon.icns") {
		info.Icons.Set("icns", xfs.Encode(fs.MustReadFile("/xiang/icons/icon.icns")))
	}

	if fs.MustExists("/xiang/icons/icon.ico") {
		info.Icons.Set("ico", xfs.Encode(fs.MustReadFile("/xiang/icons/icon.ico")))
	}

	if fs.MustExists("/xiang/icons/icon.png") {
		info.Icons.Set("png", xfs.Encode(fs.MustReadFile("/xiang/icons/icon.png")))
	}

	share.App = info
}

// defaultInfo 读取默认应用信息
func defaultInfo() share.AppInfo {
	info := share.AppInfo{
		Icons: maps.MakeSync(),
	}
	err := jsoniter.Unmarshal(data.MustAsset("xiang/data/app.json"), &info)
	if err != nil {
		exception.New("解析默认应用失败 %s", 500, err).Throw()
	}

	info.Icons.Set("icns", xfs.Encode(data.MustAsset("xiang/data/icons/icon.icns")))
	info.Icons.Set("ico", xfs.Encode(data.MustAsset("xiang/data/icons/icon.ico")))
	info.Icons.Set("png", xfs.Encode(data.MustAsset("xiang/data/icons/icon.png")))

	return info
}
