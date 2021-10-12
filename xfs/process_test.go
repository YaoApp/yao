package xfs

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/xiang/share"
)

func init() {
	share.App = share.AppInfo{
		Storage: share.AppStorage{
			Default: "oss",
			OSS: &share.AppStorageOSS{
				Endpoint:    "oss-cn-hangzhou.aliyuncs.com",
				ID:          "LTAI5tHXfWAMpcsobfqfeXsm",
				Secret:      "4NeGNw97nFrfmgBWZTlFE0XtYQdMQO",
				RoleArn:     "acs:ram::31524094:role/ramosstest",
				SessionName: "SessionTest",
			},
		},
	}
}
func TestProcessGetToken(t *testing.T) {

	args := []interface{}{
		"oss",
		"hh",
	}
	process := gou.NewProcess("xiang.fs.GetToken", args...)
	response := processGetToken(process)
	assert.NotNil(t, response)
	res := any.Of(response).Map()
	assert.True(t, res.Has("AccessKeyId"))
	assert.True(t, res.Has("AccessKeySecret"))
	assert.True(t, res.Has("Expiration"))
	assert.True(t, res.Has("SecurityToken"))
	assert.True(t, res.Has("Endpoint"))

	// 使用token
	client, err := oss.New(
		res.Get("Endpoint").(string),
		res.Get("AccessKeyId").(string),
		res.Get("AccessKeySecret").(string),
		oss.SecurityToken(res.Get("SecurityToken").(string)),
	)
	assert.Nil(t, err)

	bucket, err := client.Bucket("image-appcook")
	assert.Nil(t, err)

	// 上传字符串。
	now := fmt.Sprintf("%d", time.Now().UnixNano())
	err = bucket.PutObject("image-appcook/xiang/unit-test.txt", strings.NewReader(now))
	assert.Nil(t, err)
}
