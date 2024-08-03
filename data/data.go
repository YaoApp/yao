package data

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"strings"

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

// Setup Setup ui
func Setup() *assetfs.AssetFS {
	assetInfo := func(path string) (os.FileInfo, error) {
		return os.Stat(path)
	}
	for k := range _bintree.Children {
		k = "xgen/setup"
		return &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: assetInfo, Prefix: k, Fallback: "index.html"}
	}
	panic("unreachable")
}

// ReplaceXGen bindata file
func ReplaceXGen(search, replace string) error {
	err := replaceXGenIndex(search, replace)
	if err != nil {
		return err
	}

	err = replaceXGenUmi(search, replace)
	if err != nil {
		return err
	}

	return replaceXGenLayouts(search, replace)
}

// Read file from bin
func Read(name string) ([]byte, error) {
	return Asset(name)
}

// ReadApp read app from bin
func ReadApp() (io.Reader, error) {
	asset, err := yaoReleaseAppYazBytes()
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(asset), nil
}

// RemoveApp remove app from bin
func RemoveApp() {
	_yaoReleaseAppYaz = []byte{}
	delete(_bindata, "yao/release/app.yaz")
}

// ReplaceXGenIndex bindata file
func replaceXGenIndex(search, replace string) error {

	content, err := bindataRead(_xgenV10IndexHtml, "xgen/v1.0/index.html")
	if err != nil {
		return err
	}

	new := strings.ReplaceAll(string(content), search, replace)
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write([]byte(new)); err != nil {
		return err
	}

	if err := gz.Close(); err != nil {
		return err
	}

	_xgenV10IndexHtml = b.Bytes()
	return nil
}

// replaceXGenUmi bindata file
func replaceXGenUmi(search, replace string) error {

	content, err := bindataRead(_xgenV10UmiJs, "xgen/v1.0/umi.js")
	if err != nil {
		return err
	}

	new := strings.ReplaceAll(string(content), search, replace)
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write([]byte(new)); err != nil {
		return err
	}

	if err := gz.Close(); err != nil {
		return err
	}

	_xgenV10UmiJs = b.Bytes()
	return nil
}

// replaceXGenLayouts bindata file
func replaceXGenLayouts(search, replace string) error {

	content, err := bindataRead(_xgenV10Layouts__indexAsyncJs, "xgen/v1.0/layouts__index.async.js")
	if err != nil {
		return err
	}

	new := strings.ReplaceAll(string(content), search, replace)
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write([]byte(new)); err != nil {
		return err
	}

	if err := gz.Close(); err != nil {
		return err
	}

	_xgenV10Layouts__indexAsyncJs = b.Bytes()
	return nil
}
