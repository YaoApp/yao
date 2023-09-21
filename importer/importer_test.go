package importer

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/importer/xlsx"
	"github.com/yaoapp/yao/script"
	"github.com/yaoapp/yao/test"
)

func TestLoad(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t, config.Conf)
	assert.IsType(t, &Importer{}, Select("order"))
}
func TestFingerprintSimple(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	root := prepare(t, config.Conf)
	simple := filepath.Join(root, "assets", "simple.xlsx")
	file := xlsx.Open(simple)
	defer file.Close()

	imp := Select("order")
	fingerprint := imp.Fingerprint(file)
	assert.Equal(t, "2187b40d1e1819ffc27114caf0e80655fe44ffc4e072b07ec18611ca23951ac4", fingerprint)
}

func TestAutoMappingSimple(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	root := prepare(t, config.Conf)
	simple := filepath.Join(root, "assets", "simple.xlsx")
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

	test.Prepare(t, config.Conf)
	defer test.Clean()

	root := prepare(t, config.Conf)
	simple := filepath.Join(root, "assets", "simple.xlsx")
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
	test.Prepare(t, config.Conf)
	defer test.Clean()

	root := prepare(t, config.Conf)
	simple := filepath.Join(root, "assets", "simple.xlsx")
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
	test.Prepare(t, config.Conf)
	defer test.Clean()

	root := prepare(t, config.Conf)
	simple := filepath.Join(root, "assets", "simple.xlsx")
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
	test.Prepare(t, config.Conf)
	defer test.Clean()

	root := prepare(t, config.Conf)
	simple := filepath.Join(root, "assets", "simple.xlsx")
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
	test.Prepare(t, config.Conf)
	defer test.Clean()

	root := prepare(t, config.Conf)
	simple := filepath.Join(root, "assets", "simple.xlsx")
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
	test.Prepare(t, config.Conf)
	defer test.Clean()

	root := prepare(t, config.Conf)
	simple := filepath.Join(root, "assets", "simple.xlsx")
	file := xlsx.Open(simple)
	defer file.Close()

	imp := Select("order")
	setting := imp.MappingSetting(file)
	// utils.Dump(setting)
	assert.NotNil(t, setting)
}

func TestDataSetting(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()
	prepare(t, config.Conf)

	imp := Select("order")
	setting := imp.DataSetting()
	// utils.Dump(setting)
	assert.NotNil(t, setting)
}

func prepare(t *testing.T, cfg config.Config) string {
	err := Load(cfg)
	if err != nil {
		t.Fatal(err)
	}

	fs := fs.MustGet("system")
	dataRoot := fs.Root()
	fmt.Println("prepare", dataRoot)

	if err != nil {
		t.Fatal(err)
	}

	err = script.Load(cfg)
	if err != nil {
		t.Fatal(err)
	}

	return dataRoot
}
