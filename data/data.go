package data

import (
	"os"

	assetfs "github.com/elazarl/go-bindata-assetfs"
)

// XgenV0 XGen 0.9
func XgenV0() *assetfs.AssetFS {
	assetInfo := func(path string) (os.FileInfo, error) {
		return os.Stat(path)
	}
	for k := range _bintree.Children {
		k = "xgen/v0.9"
		return &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: assetInfo, Prefix: k, Fallback: "index.html"}
	}
	panic("unreachable")
}

// XgenV1 XGen 1.0
func XgenV1() *assetfs.AssetFS {
	assetInfo := func(path string) (os.FileInfo, error) {
		return os.Stat(path)
	}
	for k := range _bintree.Children {
		k = "xgen/v1.0"
		return &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: assetInfo, Prefix: k, Fallback: "index.html"}
	}
	panic("unreachable")
}
