package importer

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/importer/xlsx"
)

func TestLoad(t *testing.T) {
	LoadFrom("not a path", "404.")
	assert.IsType(t, &Importer{}, Select("order"))
}
func TestFingerprintSimple(t *testing.T) {
	simple := filepath.Join(config.Conf.Root, "imports", "assets", "simple.xlsx")
	file := xlsx.Open(simple)
	defer file.Close()
	imp := Select("order")
	fingerprint := imp.Fingerprint(file)
	assert.Equal(t, "3451ca87d71801687abba8993e5a69af79482914435d7cc064236fd93160f999", fingerprint)
}

func TestAutoMappingSimple(t *testing.T) {
	simple := filepath.Join(config.Conf.Root, "imports", "assets", "simple.xlsx")
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
		assert.NotEmpty(t, col.Axis)
	}
}
