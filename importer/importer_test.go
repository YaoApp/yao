package importer

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/utils"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/importer/xlsx"
)

func TestLoad(t *testing.T) {
	LoadFrom("not a path", "404.")
	assert.IsType(t, &Importer{}, Select("order"))
}
func TestFingerprintSimple(t *testing.T) {
	simple := filepath.Join(config.Conf.Root, "data", "assets", "simple.xlsx")
	file := xlsx.Open(simple)
	defer file.Close()
	imp := Select("order")
	fingerprint := imp.Fingerprint(file)
	assert.Equal(t, "3451ca87d71801687abba8993e5a69af79482914435d7cc064236fd93160f999", fingerprint)
}

func TestAutoMappingSimple(t *testing.T) {
	simple := filepath.Join(config.Conf.Root, "data", "assets", "simple.xlsx")
	file := xlsx.Open(simple)
	defer file.Close()

	imp := Select("order")
	mapping := imp.AutoMapping(file)
	assert.Equal(t, true, mapping.AutoMatching)
	assert.Equal(t, false, mapping.TemplateMatching)
	assert.Equal(t, 1, mapping.ColStart)
	assert.Equal(t, 1, mapping.RowStart)
	assert.Equal(t, 10, len(mapping.Columns))
	assert.Equal(t, len(imp.Columns), len(mapping.Columns))
	for i, col := range mapping.Columns {
		dst := imp.Columns[i].ToMap()
		assert.Equal(t, dst["name"], col.Field)
		assert.Equal(t, dst["label"], col.Label)
		assert.Equal(t, dst["rules"], col.Rules)
		if col.Field != "remark" {
			assert.NotEmpty(t, col.Axis)
		}
	}
}

func TestDataGetSimple(t *testing.T) {
	simple := filepath.Join(config.Conf.Root, "data", "assets", "simple.xlsx")
	file := xlsx.Open(simple)
	defer file.Close()

	imp := Select("order")
	mapping := imp.AutoMapping(file)
	columns, data := imp.DataGet(file, 1, 2, mapping)

	assert.Equal(t, []string{
		"order_sn", "user.name", "user.sex", "user.age", "mobile", "skus[*].name",
		"skus[*].amount", "skus[*].price", "total", "remark", "__effected",
	}, columns)

	assert.Equal(t, [][]interface{}{
		{"SN202101120018", "张三", "男", "26", "13211000011", "彩绘湖北地图", "3", "65.5", "196.5", "自动添加备注 @From 张三", true},
		{"", "李四", "男", "42", "13211000011", "景祐遁甲符应经", "1", "34.8", "34.8", "自动添加备注 @From 李四", false},
	}, data)

}

func TestDataChunkSimple(t *testing.T) {
	simple := filepath.Join(config.Conf.Root, "data", "assets", "simple.xlsx")
	file := xlsx.Open(simple)
	defer file.Close()

	imp := Select("order")
	mapping := imp.AutoMapping(file)
	lines := []int{}
	imp.Chunk(file, mapping, func(line int, data [][]interface{}) {
		lines = append(lines, line)
	})
	assert.Equal(t, []int{3, 4}, lines)
}

func TestRunSimple(t *testing.T) {
	simple := filepath.Join(config.Conf.Root, "data", "assets", "simple.xlsx")
	file := xlsx.Open(simple)
	defer file.Close()

	imp := Select("order")
	mapping := imp.AutoMapping(file)
	res := imp.Run(file, mapping).(map[string]int)
	assert.Equal(t, 1, res["ignore"])
	assert.Equal(t, 1, res["failure"])
	assert.Equal(t, 2, res["success"])
	assert.Equal(t, 4, res["total"])
}

func TestDataPreviewSimple(t *testing.T) {
	simple := filepath.Join(config.Conf.Root, "data", "assets", "simple.xlsx")
	file := xlsx.Open(simple)
	defer file.Close()
	imp := Select("order")
	mapping := imp.AutoMapping(file)

	res := imp.DataPreview(file, 2, 3, mapping)
	data, ok := res["data"].([]map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, 2, res["page"])
	assert.Equal(t, 3, res["next"])
	assert.Equal(t, 10, res["pagecnt"])
	assert.Equal(t, 3, res["pagesize"])
	assert.Equal(t, 1, res["prev"])
	assert.Equal(t, 1, len(data))
	assert.Equal(t, 12, len(data[0]))
}

func TestMappingPreviewSimple(t *testing.T) {
	simple := filepath.Join(config.Conf.Root, "data", "assets", "simple.xlsx")
	file := xlsx.Open(simple)
	defer file.Close()

	imp := Select("order")
	mapping := imp.MappingPreview(file)
	assert.Equal(t, true, mapping.AutoMatching)
	assert.Equal(t, false, mapping.TemplateMatching)
	assert.Equal(t, 1, mapping.ColStart)
	assert.Equal(t, 1, mapping.RowStart)
	assert.Equal(t, 10, len(mapping.Columns))
	assert.Equal(t, len(imp.Columns), len(mapping.Columns))
	for i, col := range mapping.Columns {
		dst := imp.Columns[i].ToMap()
		assert.Equal(t, dst["name"], col.Field)
		assert.Equal(t, dst["label"], col.Label)
		assert.Equal(t, dst["rules"], col.Rules)
		if col.Field != "remark" {
			assert.NotEmpty(t, col.Axis)
		}
	}
}

func TestMappingSetting(t *testing.T) {
	simple := filepath.Join(config.Conf.Root, "data", "assets", "simple.xlsx")
	file := xlsx.Open(simple)
	defer file.Close()

	imp := Select("order")
	setting := imp.MappingSetting(file)
	utils.Dump(setting)
	assert.NotNil(t, setting)
}

func TestDataSetting(t *testing.T) {
	imp := Select("order")
	setting := imp.DataSetting()
	utils.Dump(setting)
	assert.NotNil(t, setting)
}
