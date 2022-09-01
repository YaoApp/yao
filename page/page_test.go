package page

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/lang"
	"github.com/yaoapp/yao/model"
	"github.com/yaoapp/yao/query"
	"github.com/yaoapp/yao/share"
)

func TestLoad(t *testing.T) {

	os.Setenv("YAO_LANG", "zh-hk")
	lang.Load(config.Conf)

	share.DBConnect(config.Conf.DB)
	model.Load(config.Conf)
	query.Load(config.Conf)

	Load(config.Conf)
	LoadFrom("not a path", "404.")
	check(t)
}

func check(t *testing.T) {
	keys := []string{}
	for key := range Pages {
		keys = append(keys, key)
	}
	assert.Equal(t, 2, len(keys))

	lang := Pages["lang"]
	output := lang.Output.(map[string]interface{})
	assert.Equal(t, "{{$in}}", output["參數"])

	filters := lang.Page.Layout["filters"].([]interface{})
	begin := filters[0].(map[string]interface{})
	assert.Equal(t, "開始時間", begin["name"])

	assert.Equal(t, "請選擇開始時間", lang.Filters["開始時間"].Input.Props["placeholder"])
}
