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
