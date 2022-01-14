package importer

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/xiang/importer/xlsx"
)

func TestLoad(t *testing.T) {
	LoadFrom("not a path", "404.")
	assert.IsType(t, &Importer{}, Select("manu"))
}
func TestFingerprint(t *testing.T) {
	file := xlsx.Open()
	imp := Select("manu")
	fp := imp.Fingerprint(file)
	fmt.Println(fp)
}
