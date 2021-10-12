package xfs

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/xun"
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
	return nil
}

// processGetURL 返回文件CDN地址
func processGetURL(process *gou.Process) interface{} {
	return nil
}
