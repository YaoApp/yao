package data

import (
	"os"

	assetfs "github.com/elazarl/go-bindata-assetfs"
)

// AssetFS 静态文件处理服务器
func AssetFS() *assetfs.AssetFS {
	assetInfo := func(path string) (os.FileInfo, error) {
		return os.Stat(path)
	}
	for k := range _bintree.Children {
		k = "ui"
		return &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: assetInfo, Prefix: k, Fallback: "index.html"}
	}
	panic("unreachable")
}
