package share

// App 应用信息
var App AppInfo

// Public 输出公共信息
func (app AppInfo) Public() AppInfo {
	app.Storage.COS = nil
	app.Storage.OSS = nil
	app.Storage.S3 = nil
	return app
}

// GetPrefix Get the prefix of the app with the default value "yao_"
func (app AppInfo) GetPrefix() string {
	if app.Prefix == "" {
		return "yao_"
	}
	return app.Prefix
}
