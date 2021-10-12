package global

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xiang/data"
	"github.com/yaoapp/xiang/xfs"
)

// App 应用信息
var App AppInfo

// AppInfo 应用信息
type AppInfo struct {
	Name        string                 `json:"name,omitempty"`
	Short       string                 `json:"short,omitempty"`
	Version     string                 `json:"version,omitempty"`
	Description string                 `json:"description,omitempty"`
	Icons       map[string]string      `json:"icons,omitempty"`
	Storage     AppStorage             `json:"storage,omitempty"`
	Option      map[string]interface{} `json:"option,omitempty"`
}

// AppStorage 应用存储
type AppStorage struct {
	Default string                 `json:"default"`
	Buckets map[string]string      `json:"buckets,omitempty"`
	S3      map[string]interface{} `json:"s3,omitempty"`
	OSS     map[string]interface{} `json:"oss,omitempty"`
	COS     map[string]interface{} `json:"cos,omitempty"`
}

// LoadAppInfo 读取应用信息
func LoadAppInfo(root string) {
	info := defaultAppInfo()
	fs := xfs.New(root)
	if fs.MustExists("/app.json") {
		err := jsoniter.Unmarshal(fs.MustReadFile("/app.json"), &info)
		if err != nil {
			exception.New("解析应用失败 %s", 500, err).Throw()
		}
	}

	if fs.MustExists("/xiang/icons/icon.icns") {
		info.Icons["icns"] = xfs.Encode(fs.MustReadFile("/xiang/icons/icon.icns"))
	}

	if fs.MustExists("/xiang/icons/icon.ico") {
		info.Icons["ico"] = xfs.Encode(fs.MustReadFile("/xiang/icons/icon.ico"))
	}

	if fs.MustExists("/xiang/icons/icon.png") {
		info.Icons["png"] = xfs.Encode(fs.MustReadFile("/xiang/icons/icon.png"))
	}

	App = info
}

// defaultAppInfo 读取默认应用信息
func defaultAppInfo() AppInfo {
	info := AppInfo{}
	err := jsoniter.Unmarshal(data.MustAsset("xiang/data/app.json"), &info)
	if err != nil {
		exception.New("解析默认应用失败 %s", 500, err).Throw()
	}

	info.Version = VERSION
	info.Icons["icns"] = xfs.Encode(data.MustAsset("xiang/data/icons/icon.icns"))
	info.Icons["ico"] = xfs.Encode(data.MustAsset("xiang/data/icons/icon.ico"))
	info.Icons["png"] = xfs.Encode(data.MustAsset("xiang/data/icons/icon.png"))

	return info
}

// Public 输出公共信息
func (app AppInfo) Public() AppInfo {
	app.Storage.COS = nil
	app.Storage.OSS = nil
	app.Storage.S3 = nil
	return app
}
