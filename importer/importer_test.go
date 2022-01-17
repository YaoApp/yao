package importer

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/utils"
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
	utils.Dump(mapping)
}
