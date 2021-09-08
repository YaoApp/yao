package main

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	err := Load()
	assert.Nil(t, err)
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
