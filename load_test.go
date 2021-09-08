package main

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	assert.NotPanics(t, func() {
		Load(cfg)
	})
}

// 从文件系统载入引擎文件
func TestLoadEngineFS(t *testing.T) {
	root := "fs://" + path.Join(cfg.Source, "/xiang")
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
		LoadApp(cfg.RootAPI, cfg.RootFLow, cfg.RootModel, cfg.RootPlugin)
	})
}
