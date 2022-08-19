package app

import (
	jsoniter "github.com/json-iterator/go"
	l "github.com/yaoapp/gou/lang"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/data"
	"github.com/yaoapp/yao/lang"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/xfs"
)

// Load Application
func Load(cfg config.Config) {

	// Load language packs
	lang.Load(cfg)

	// Set language pack
	share.App.L = map[string]string{}
	if l.Default != nil {
		share.App.L = l.Default.Global
	}

	// Load Info
	LoadInfo(cfg.Root)

}

// L 语言包
func L(word string) string {
	if trans, has := share.App.L[word]; has {
		return trans
	}
	return word
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

	if fs.MustExists("/yao/icons/icon.icns") {
		info.Icons.Set("icns", xfs.Encode(fs.MustReadFile("/yao/icons/icon.icns")))
	}

	if fs.MustExists("/yao/icons/icon.ico") {
		info.Icons.Set("ico", xfs.Encode(fs.MustReadFile("/yao/icons/icon.ico")))
	}

	if fs.MustExists("/yao/icons/icon.png") {
		info.Icons.Set("png", xfs.Encode(fs.MustReadFile("/yao/icons/icon.png")))
	}

	info.L = share.App.L
	share.App = info
}

// LoadLangBuildIn 从制品中读取
func LoadLangBuildIn(dir string) error {
	return nil
}

// defaultInfo 读取默认应用信息
func defaultInfo() share.AppInfo {
	info := share.AppInfo{
		Icons: maps.MakeSync(),
	}
	err := jsoniter.Unmarshal(data.MustAsset("yao/data/app.json"), &info)
	if err != nil {
		exception.New("解析默认应用失败 %s", 500, err).Throw()
	}

	info.Icons.Set("icns", xfs.Encode(data.MustAsset("yao/data/icons/icon.icns")))
	info.Icons.Set("ico", xfs.Encode(data.MustAsset("yao/data/icons/icon.ico")))
	info.Icons.Set("png", xfs.Encode(data.MustAsset("yao/data/icons/icon.png")))

	return info
}
