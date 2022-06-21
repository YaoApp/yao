package xfs

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/sts"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/xun"
	"github.com/yaoapp/yao/data"
	"github.com/yaoapp/yao/helper"
	"github.com/yaoapp/yao/share"
)

func init() {
	// 注册处理器
	gou.RegisterProcessHandler("xiang.fs.Upload", processUpload)
	gou.RegisterProcessHandler("xiang.fs.GetToken", processGetToken)
	gou.RegisterProcessHandler("xiang.fs.GetURL", processGetURL)
	gou.RegisterProcessHandler("xiang.fs.ReadFile", processReadFile)
}

// processUpload 上传文件到本地服务器
func processUpload(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	tmpfile, ok := process.Args[0].(xun.UploadFile)
	if !ok {
		exception.New("上传文件参数错误", 400, process.Args[0]).Throw()
	}

	hash := md5.Sum([]byte(time.Now().Format("20060102-15:04:05")))
	fingerprint := string(hex.EncodeToString(hash[:]))
	fingerprint = strings.ToUpper(fingerprint)

	dir := time.Now().Format("20060102")
	ext := filepath.Ext(tmpfile.Name)
	filename := filepath.Join(dir, fmt.Sprintf("%s%s", fingerprint, ext))
	Stor.MustMkdirAll(dir, os.ModePerm)

	content, err := New("/").ReadFile(tmpfile.TempFile)
	if err != nil {
		exception.New("不能读取上传文件 %s", 500, err.Error()).Throw()
	}

	Stor.MustWriteFile(filename, content, os.ModePerm)
	return filename
}

// processGetContent 返回文件正文
func processReadFile(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	filename := process.ArgsString(0)
	encode := process.ArgsBool(1, true)

	stats, err := Stor.Stat(filename)
	if err != nil {
		exception.New("读取文件信息失败 %s", 500, err.Error()).Throw()
	}

	var content string
	body := Stor.MustReadFile(filename)
	if encode {
		content = Encode(body)
	} else {
		content = string(body)
	}

	return maps.Map{
		"size":    stats.Size(),
		"content": content,
	}
}

// processGetToken 上传文件到腾讯云对象存储 COS
func processGetToken(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	name := process.ArgsString(0, "oss")
	// bucket := process.ArgsString(1)
	if name != "oss" {
		exception.New("暂时支持 oss 存储", 400).Throw()
	}

	if share.App.Storage.OSS == nil {
		exception.New("未配置 OSS 存储", 400).Throw()
	}

	app := share.App.Storage.OSS
	id := strings.Split(app.Endpoint, ".")[0]
	client, err := sts.NewClientWithAccessKey(id, app.ID, app.Secret)
	if err != nil {
		exception.New("配置错误 %s", 400, err.Error()).Throw()
	}

	//构建请求对象。
	request := sts.CreateAssumeRoleRequest()
	request.Scheme = "https"
	request.Domain = "sts.aliyuncs.com"

	//设置参数。关于参数含义和设置方法，请参见API参考。
	request.RoleArn = app.RoleArn
	request.RoleSessionName = app.SessionName

	//发起请求，并得到响应。
	response, err := client.AssumeRole(request)
	if err != nil {
		exception.New("配置错误 %s", 400, err.Error()).Throw()
	}

	res := any.Of(response.Credentials).Map()
	res.Set("Endpoint", app.Endpoint)
	return res.MapStrAny
}

// 临时
const svg404 = `
	<?xml version="1.0" encoding="UTF-8" standalone="no" ?>
	<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">
	<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" version="1.1" width="200" height="200" viewBox="0 0 200 200" xml:space="preserve">
	<desc>Created with Fabric.js 4.2.0</desc>
	<defs>
	</defs>
	<g transform="matrix(1 0 0 1 100 100)" id="5119fde9-97f8-4fee-9a96-c400a7cca9df"  >
	<rect style="stroke: none; stroke-width: 1; stroke-dasharray: none; stroke-linecap: butt; stroke-dashoffset: 0; stroke-linejoin: miter; stroke-miterlimit: 4; fill: rgb(255,255,255); fill-rule: nonzero; opacity: 1;" vector-effect="non-scaling-stroke"  x="-100" y="-100" rx="0" ry="0" width="200" height="200" />
	</g>
	<g transform="matrix(Infinity NaN NaN Infinity 0 0)" id="cea4fb49-00ad-46dc-9ce3-9126a7052542"  >
	</g>
	<g transform="matrix(0.67 0 0 0.67 100 100)" style="" id="076b642e-21e1-4732-b23b-3ee381fc6ace"  >
			<text xml:space="preserve" font-family="Raleway" font-size="105" font-style="normal" font-weight="900" style="stroke: none; stroke-width: 1; stroke-dasharray: none; stroke-linecap: butt; stroke-dashoffset: 0; stroke-linejoin: miter; stroke-miterlimit: 4; fill: rgb(0,0,0); fill-rule: nonzero; opacity: 1; white-space: pre;" ><tspan x="-93.77" y="32.98" >404</tspan></text>
	</g>
	</svg>
`

// processGetURL 返回文件CDN地址
func processGetURL(process *gou.Process) interface{} {
	if len(process.Args) < 1 {
		return string(data.MustAsset("yao/data/icons/404.png"))
	}

	token := process.ArgsString(2)
	if token == "" {
		exception.New("token is null", 403).Throw()
	}

	claims := helper.JwtValidate(token)
	userID, err := session.Global().ID(claims.SID).Get("user_id")
	if err != nil || userID == nil {
		exception.New("session expired", 403).Throw()
	}

	filename := process.ArgsString(0)
	body, err := Stor.ReadFile(filename)
	if err != nil {
		return string(data.MustAsset("yao/data/icons/404.png"))
	}
	return string(body)
}
