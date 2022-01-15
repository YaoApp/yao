package importer

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/importer/xlsx"
)

func TestLoad(t *testing.T) {
	LoadFrom("not a path", "404.")
	assert.IsType(t, &Importer{}, Select("manu"))
}
func TestFingerprint(t *testing.T) {
	simple := filepath.Join(config.Conf.Root, "imports", "assets", "simple.xlsx")
	file := xlsx.Open(simple)
	defer file.Close()
	imp := Select("manu")
	fp := imp.Fingerprint(file)
	fmt.Println(fp)
}
