package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/maps"
)

var testRecords = []map[string]interface{}{
	{"id": 1, "name": "云服务", "category_id": nil, "type_id": nil, "rank": 1, "parent_id": 0},
	{"id": 2, "name": "基础服务", "category_id": nil, "type_id": nil, "rank": 1, "parent_id": 1},
	{"id": 3, "name": "云主机", "category_id": nil, "type_id": 4, "rank": 1, "parent_id": 2},
	{"id": 4, "name": "对象存储", "category_id": nil, "type_id": 5, "rank": 2, "parent_id": 2},
	{"id": 5, "name": "云数据库", "category_id": nil, "type_id": 6, "rank": 3, "parent_id": 2},
	{"id": 6, "name": "块存储", "category_id": nil, "type_id": 7, "rank": 4, "parent_id": 2},
	{"id": 7, "name": "应用托管容器", "category_id": nil, "type_id": 8, "rank": 5, "parent_id": 2},
	{"id": 8, "name": "云缓存", "category_id": nil, "type_id": 9, "rank": 6, "parent_id": 2},
	{"id": 9, "name": "本地负载均衡", "category_id": nil, "type_id": 10, "rank": 7, "parent_id": 2},
	{"id": 10, "name": "全局负载均衡", "category_id": nil, "type_id": 13, "rank": 8, "parent_id": 2},
	{"id": 11, "name": "云分发", "category_id": nil, "type_id": 11, "rank": 9, "parent_id": 2},
	{"id": 12, "name": "企业级SaaS", "category_id": nil, "type_id": 12, "rank": 10, "parent_id": 2},
	{"id": 13, "name": "云桌面", "category_id": nil, "type_id": 14, "rank": 11, "parent_id": 2},
	{"id": 14, "name": "云备份", "category_id": nil, "type_id": 17, "rank": 12, "parent_id": 2},
	{"id": 15, "name": "GPU云主机", "category_id": nil, "type_id": 18, "rank": 13, "parent_id": 2},
	{"id": 16, "name": "物理云主机", "category_id": nil, "type_id": 20, "rank": 14, "parent_id": 2},
	{"id": 17, "name": "智能云", "category_id": 41, "type_id": nil, "rank": 2, "parent_id": 2},
	{"id": 18, "name": "软件和开发", "category_id": nil, "type_id": nil, "rank": 2, "parent_id": 0},
	{"id": 19, "name": "虚拟化及管理", "category_id": 59, "type_id": nil, "rank": 1, "parent_id": 18},
	{"id": 20, "name": "容器解决方案", "category_id": 65, "type_id": nil, "rank": 2, "parent_id": 18},
	{"id": 21, "name": "微服务解决方案", "category_id": 76, "type_id": nil, "rank": 3, "parent_id": 18},
	{"id": 22, "name": "serverless解决方案", "category_id": 82, "type_id": nil, "rank": 4, "parent_id": 18},
	{"id": 23, "name": "云管理和云运营", "category_id": nil, "type_id": nil, "rank": 3, "parent_id": 0},
	{"id": 24, "name": "混合云", "category_id": nil, "type_id": nil, "rank": 1, "parent_id": 23},
	{"id": 25, "name": "混合云解决方案", "category_id": 88, "type_id": nil, "rank": 1, "parent_id": 24},
	{"id": 26, "name": "混合云安全", "category_id": 671, "type_id": nil, "rank": 2, "parent_id": 24},
	{"id": 27, "name": "多云管理", "category_id": 94, "type_id": nil, "rank": 2, "parent_id": 23},
	{"id": 28, "name": "金牌运维", "category_id": 120, "type_id": nil, "rank": 3, "parent_id": 23},
	{"id": 29, "name": "研发运营一体化", "category_id": 443, "type_id": nil, "rank": 4, "parent_id": 23},
	{"id": 30, "name": "MSP", "category_id": 112, "type_id": nil, "rank": 5, "parent_id": 23},
	{"id": 31, "name": "安全与保险", "category_id": nil, "type_id": nil, "rank": 4, "parent_id": 0},
	{"id": 32, "name": "风险管理", "category_id": 141, "type_id": nil, "rank": 1, "parent_id": 31},
	{"id": 33, "name": "云服务用户数据保护", "category_id": 152, "type_id": nil, "rank": 2, "parent_id": 31},
	{"id": 34, "name": "业务风控", "category_id": nil, "type_id": nil, "rank": 3, "parent_id": 31},
	{"id": 35, "name": "内容安全", "category_id": 159, "type_id": nil, "rank": 1, "parent_id": 34},
	{"id": 36, "name": "反交易欺诈", "category_id": 164, "type_id": nil, "rank": 2, "parent_id": 34},
	{"id": 37, "name": "反信贷欺诈", "category_id": 165, "type_id": nil, "rank": 3, "parent_id": 34},
	{"id": 38, "name": "反营销欺诈", "category_id": 166, "type_id": nil, "rank": 4, "parent_id": 34},
	{"id": 39, "name": "反钓鱼欺诈", "category_id": 167, "type_id": nil, "rank": 5, "parent_id": 34},
	{"id": 40, "name": "云主机安全", "category_id": 184, "type_id": nil, "rank": 4, "parent_id": 31},
	{"id": 41, "name": "态势感知", "category_id": 190, "type_id": nil, "rank": 5, "parent_id": 31},
	{"id": 42, "name": "云保险", "category_id": 272, "type_id": nil, "rank": 6, "parent_id": 31},
	{"id": 43, "name": "云网&云边", "category_id": nil, "type_id": nil, "rank": 5, "parent_id": 0},
	{"id": 44, "name": "云平台网络能力", "category_id": 102, "type_id": nil, "rank": 1, "parent_id": 43},
	{"id": 45, "name": "SD-WAN", "category_id": 106, "type_id": nil, "rank": 2, "parent_id": 43},
	{"id": 46, "name": "物联网", "category_id": 234, "type_id": nil, "rank": 3, "parent_id": 43},
	{"id": 47, "name": "行业云", "category_id": nil, "type_id": nil, "rank": 6, "parent_id": 0},
	{"id": 48, "name": "政务", "category_id": nil, "type_id": nil, "rank": 1, "parent_id": 47},
	{"id": 49, "name": "政务云综合水平评估", "category_id": 215, "type_id": nil, "rank": 1, "parent_id": 48},
	{"id": 50, "name": "可信政务云评估", "category_id": 216, "type_id": nil, "rank": 2, "parent_id": 48},
	{"id": 51, "name": "金融", "category_id": 225, "type_id": nil, "rank": 2, "parent_id": 47},
	{"id": 52, "name": "开源治理", "category_id": nil, "type_id": nil, "rank": 7, "parent_id": 0},
	{"id": 53, "name": "面向开源用户企业", "category_id": 196, "type_id": nil, "rank": 1, "parent_id": 52},
	{"id": 54, "name": "面向自发开源企业", "category_id": 202, "type_id": nil, "rank": 2, "parent_id": 52},
	{"id": 55, "name": "开源项目评估", "category_id": 711, "type_id": nil, "rank": 3, "parent_id": 52},
	{"id": 56, "name": "开源工具评估", "category_id": 720, "type_id": nil, "rank": 4, "parent_id": 52},
	{"id": 57, "name": "检测平台", "category_id": nil, "type_id": nil, "rank": 8, "parent_id": 0},
	{"id": 58, "name": "云主机分级", "category_id": 371, "type_id": nil, "rank": 1, "parent_id": 57},
	{"id": 59, "name": "监管支撑", "category_id": nil, "type_id": nil, "rank": 9, "parent_id": 0},
	{"id": 60, "name": "企业上云效果成熟度", "category_id": 250, "type_id": nil, "rank": 1, "parent_id": 59},
	{"id": 61, "name": "综合信用评估", "category_id": nil, "type_id": nil, "rank": 2, "parent_id": 59},
	{"id": 62, "name": "云服务企业", "category_id": 261, "type_id": nil, "rank": 1, "parent_id": 61},
	{"id": 63, "name": "CDN服务企业", "category_id": 271, "type_id": nil, "rank": 2, "parent_id": 61},
}

func TestArrayPluck(t *testing.T) {
	columns := []string{"城市", "行业", "计费"}
	pluck := map[string]interface{}{
		"行业": map[string]interface{}{"key": "city", "value": "数量", "items": []map[string]interface{}{{"city": "北京", "数量": 32}, {"city": "上海", "数量": 20}}},
		"计费": map[string]interface{}{"key": "city", "value": "计费种类", "items": []map[string]interface{}{{"city": "北京", "计费种类": 6}, {"city": "西安", "计费种类": 3}}},
	}
	items := ArrayPluck(columns, pluck)
	assert.Equal(t, 3, len(items))
	for _, item := range items {
		maps.Of(item).Has("城市")
		maps.Of(item).Has("行业")
		maps.Of(item).Has("计费")
	}
}

func TestArraySplit(t *testing.T) {
	records := []map[string]interface{}{
		{"name": "阿里云计算有限公司", "short_name": "阿里云"},
		{"name": "世纪互联蓝云", "short_name": "上海蓝云"},
	}
	columns, values := ArraySplit(records)
	assert.Equal(t, 2, len(columns))
	assert.Equal(t, 2, len(values))
	for _, value := range values {
		assert.Equal(t, 2, len(value))
	}
}

func TestArrayTree(t *testing.T) {
	records := testRecords
	res := ArrayTree(records, map[string]interface{}{"parent": "parent_id"})
	assert.Equal(t, 9, len(res))
}

func TestProcessArrayPluck(t *testing.T) {
	args := []interface{}{
		[]interface{}{"城市", "行业", "计费"},
		map[string]interface{}{
			"行业": map[string]interface{}{"key": "city", "value": "数量", "items": []map[string]interface{}{{"city": "北京", "数量": 32}, {"city": "上海", "数量": 20}}},
			"计费": map[string]interface{}{"key": "city", "value": "计费种类", "items": []map[string]interface{}{{"city": "北京", "计费种类": 6}, {"city": "西安", "计费种类": 3}}},
		},
	}
	process := process.New("xiang.helper.ArrayPluck", args...)
	response := ProcessArrayPluck(process)
	assert.NotNil(t, response)
	items, ok := response.([]map[string]interface{})
	assert.True(t, ok)

	assert.Equal(t, 3, len(items))
	for _, item := range items {
		maps.Of(item).Has("城市")
		maps.Of(item).Has("行业")
		maps.Of(item).Has("计费")
	}
}

func TestProcessArraySplit(t *testing.T) {
	args := []interface{}{
		[]map[string]interface{}{
			{"name": "阿里云计算有限公司", "short_name": "阿里云"},
			{"name": "世纪互联蓝云", "short_name": "上海蓝云"},
		},
	}
	process := process.New("xiang.helper.ArraySplit", args...)
	response := process.Run()
	assert.NotNil(t, response)
	res, ok := response.(map[string]interface{})
	assert.True(t, ok)

	columns, ok := res["columns"].([]string)
	assert.True(t, ok)

	values, ok := res["values"].([][]interface{})
	assert.True(t, ok)
	assert.Equal(t, 2, len(columns))
	assert.Equal(t, 2, len(values))
	for _, value := range values {
		assert.Equal(t, 2, len(value))
	}
}

func TestProcessArrayUnique(t *testing.T) {
	args := []interface{}{
		[]interface{}{1, 2, 3, 3},
	}
	process := process.New("xiang.helper.ArrayUnique", args...)
	response := process.Run()
	assert.NotNil(t, response)
	res, ok := response.([]interface{})
	assert.True(t, ok)
	assert.Equal(t, []interface{}{1, 2, 3}, res)
}

func TestProcessArrayIndexes(t *testing.T) {
	args := []interface{}{
		[]interface{}{1, 2, 3, 3},
	}
	response := process.New("xiang.helper.ArrayIndexes", args...).Run()
	assert.NotNil(t, response)
	res, ok := response.([]int)
	assert.True(t, ok)
	assert.Equal(t, []int{0, 1, 2, 3}, res)
}

func TestProcessArrayGet(t *testing.T) {

	response := process.New("xiang.helper.ArrayGet", []interface{}{1, 2, 3, 3}, 2).Run()
	assert.Equal(t, 3, response)

	response = process.New("xiang.helper.ArrayGet", []interface{}{1, 2, 3, 3}, 4).Run()
	assert.Nil(t, response)

}
