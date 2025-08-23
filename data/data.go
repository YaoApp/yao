package data

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"strings"

	assetfs "github.com/elazarl/go-bindata-assetfs"
)

// CuiV0 CUI 0.9
func CuiV0() *assetfs.AssetFS {
	assetInfo := func(path string) (os.FileInfo, error) {
		return os.Stat(path)
	}
	for k := range _bintree.Children {
		k = "cui/v0.9"
		return &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: assetInfo, Prefix: k, Fallback: "index.html"}
	}
	panic("unreachable")
}

// CuiV1 CUI 1.0
func CuiV1() *assetfs.AssetFS {
	assetInfo := func(path string) (os.FileInfo, error) {
		return os.Stat(path)
	}
	for k := range _bintree.Children {
		k = "cui/v1.0"
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
		k = "cui/setup"
		return &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: assetInfo, Prefix: k, Fallback: "index.html"}
	}
	panic("unreachable")
}

// ReplaceCUI bindata file
func ReplaceCUI(search, replace string) error {
	err := replaceCUIIndex(search, replace)
	if err != nil {
		return err
	}

	err = replaceCUIUmi(search, replace)
	if err != nil {
		return err
	}

	return replaceCUILayouts(search, replace)
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

// ReplaceCUIIndex bindata file
func replaceCUIIndex(search, replace string) error {

	content, err := bindataRead(_cuiV10IndexHtml, "cui/v1.0/index.html")
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

	_cuiV10IndexHtml = b.Bytes()
	return nil
}

// replaceCUIUmi bindata file
func replaceCUIUmi(search, replace string) error {

	content, err := bindataRead(_cuiV10UmiJs, "cui/v1.0/umi.js")
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

	_cuiV10UmiJs = b.Bytes()
	return nil
}

// replaceCUILayouts bindata file
func replaceCUILayouts(search, replace string) error {

	content, err := bindataRead(_cuiV10Layouts__indexAsyncJs, "cui/v1.0/layouts__index.async.js")
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

	_cuiV10Layouts__indexAsyncJs = b.Bytes()
	return nil
}
