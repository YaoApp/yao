package share

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
	OSS     *AppStorageOSS         `json:"oss,omitempty"`
	COS     map[string]interface{} `json:"cos,omitempty"`
}

// AppStorageOSS 阿里云存储
type AppStorageOSS struct {
	Endpoint    string `json:"endpoint,omitempty"`
	ID          string `json:"id,omitempty"`
	Secret      string `json:"secret,omitempty"`
	RoleArn     string `json:"roleArn,omitempty"`
	SessionName string `json:"sessionName,omitempty"`
}

// Public 输出公共信息
func (app AppInfo) Public() AppInfo {
	app.Storage.COS = nil
	app.Storage.OSS = nil
	app.Storage.S3 = nil
	return app
}
