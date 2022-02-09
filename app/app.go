package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/data"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/xfs"
)

// langs 语言包
var langs = map[string]map[string]string{}

// Load 加载应用信息
func Load(cfg config.Config) {
	Init(cfg)
	LoadInfo(cfg.Root)
	LoadLang(cfg)

	lang := strings.ToLower(share.App.Lang)
	share.App.L = map[string]string{}
	if l, has := langs[lang]; has {
		share.App.L = l
	}
}

// L 语言包
func L(word string) string {
	if trans, has := share.App.L[word]; has {
		return trans
	}
	return word
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

// LoadLang 加载语言包
func LoadLang(cfg config.Config) error {

	var defaults = []share.Script{}
	if os.Getenv("YAO_DEV") != "" {
		defaults = share.GetFilesFS(filepath.Join(os.Getenv("YAO_DEV"), "xiang", "langs"), ".json")
	} else {
		defaults = share.GetFilesBin("/xiang/langs", ".json")
	}

	for _, lang := range defaults {
		content := lang.Content
		name := strings.ToLower(lang.Type) // 这个读取函数需要优化
		lang := map[string]string{}
		err := jsoniter.Unmarshal(content, &lang)
		if err != nil {
			log.With(log.F{"name": name, "content": content}).Error(err.Error())
		}
		langs[name] = lang
	}

	if share.BUILDIN {
		return LoadLangBuildIn("langs")
	}
	return LoadLangFrom(filepath.Join(cfg.Root, "langs"))
}

// LoadLangBuildIn 从制品中读取
func LoadLangBuildIn(dir string) error {
	return nil
}

// LoadLangFrom 从特定目录加载
func LoadLangFrom(dir string) error {

	if share.DirNotExists(dir) {
		return fmt.Errorf("%s does not exists", dir)
	}

	err := share.Walk(dir, ".json", func(root, filename string) {
		name := strings.ToLower(share.SpecName(root, filename))
		content := share.ReadFile(filename)
		lang := map[string]string{}
		err := jsoniter.Unmarshal(content, &lang)
		if err != nil {
			log.With(log.F{"root": root, "file": filename}).Error(err.Error())
		}
		if _, has := langs[name]; !has {
			langs[name] = map[string]string{}
		}
		for src, dst := range lang {
			langs[name][src] = dst
		}
	})

	return err

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
