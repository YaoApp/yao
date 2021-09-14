package main

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/xiang/global"
)

func TestLoad(t *testing.T) {
	assert.NotPanics(t, func() {
		global.Load(global.Conf)
	})
}

// 从文件系统载入引擎文件
func TestLoadEngineFS(t *testing.T) {
	root := "fs://" + path.Join(global.Conf.Source, "/xiang")
	assert.NotPanics(t, func() {
		global.LoadEngine(root)
	})

}

// 从BinDataz载入引擎文件
func TestLoadEngineBin(t *testing.T) {
	root := "bin://xiang"
	assert.NotPanics(t, func() {
		global.LoadEngine(root)
	})
}

// 从文件系统载入应用脚本
func TestLoadAppFS(t *testing.T) {
	assert.NotPanics(t, func() {
		global.LoadApp(global.Conf.RootAPI, global.Conf.RootFLow, global.Conf.RootModel, global.Conf.RootPlugin)
	})
}
