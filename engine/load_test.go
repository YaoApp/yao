package engine

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
)

func TestLoad(t *testing.T) {
	defer Load(config.Conf)
	assert.NotPanics(t, func() {
		Load(config.Conf)
	})
}

// 从文件系统载入引擎文件
func TestLoadEngineFS(t *testing.T) {
	defer Load(config.Conf)
	root := path.Join(os.Getenv("YAO_DEV"), "/yao")
	assert.NotPanics(t, func() {
		LoadEngine(root)
	})

}

// 从BinDataz载入引擎文件
func TestLoadEngineBin(t *testing.T) {
	defer Load(config.Conf)
	assert.NotPanics(t, func() {
		LoadEngine()
	})
}
