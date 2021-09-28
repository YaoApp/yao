package global

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	assert.NotPanics(t, func() {
		Load(Conf)
	})
}

// 从文件系统载入引擎文件
func TestLoadEngineFS(t *testing.T) {
	root := "fs://" + path.Join(Conf.Source, "/xiang")
	assert.NotPanics(t, func() {
		LoadEngine(root)
	})

}

// 从BinDataz载入引擎文件
func TestLoadEngineBin(t *testing.T) {
	root := "bin://xiang"
	assert.NotPanics(t, func() {
		LoadEngine(root)
	})
}

// 从文件系统载入应用脚本
func TestLoadAppFS(t *testing.T) {
	assert.NotPanics(t, func() {
		LoadApp(AppRoot{
			APIs:    Conf.RootAPI,
			Flows:   Conf.RootFLow,
			Models:  Conf.RootModel,
			Plugins: Conf.RootPlugin,
			Tables:  Conf.RootTable,
			Charts:  Conf.RootChart,
			Screens: Conf.RootScreen,
		})
	})
}
