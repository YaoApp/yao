package xfs

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/afero/tarfs"
	"github.com/spf13/afero/zipfs"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/config"
)

// DEPRECATED

// Stor 文件系统实例
var Stor Xfs

// Xfs 文件系统
type Xfs struct {
	Root string // Root
	afero.Afero
}

// Fs 文件系统 Interface
type Fs interface {
	afero.Fs
}

// File file fs
type File interface {
	afero.File
}

func init() {
	Stor = New(filepath.Join(config.Conf.Root, "data"))
}

// New 创建文件系统
func New(path string) Xfs {
	pinfo := strings.Split(path, "://")
	fs := "fs"
	root := path
	if len(pinfo) == 2 {
		fs = pinfo[0]
		root = strings.TrimPrefix(path, fmt.Sprintf("%s://", fs))
	}

	xfs := Xfs{
		Root: root,
	}
	switch fs {
	case "fs", "file":
		xfs.Fs = afero.NewBasePathFs(afero.NewOsFs(), root)
		break
	case "mem", "memory":
		xfs.Fs = afero.NewMemMapFs()
		break
	default:
		exception.New("尚未支持 %s 文件系统", 500, fs).Throw()
	}

	// exists := xfs.MustDirExists("/")
	// if !exists && root != "/" {
	// 	xfs.MustMkdirAll("/", os.ModePerm)
	// }

	return xfs
}

// NewZip 创建Zip文件系统
func NewZip(zipfile string) Xfs {
	r, err := zip.OpenReader(zipfile)
	if err != nil {
		exception.New("ZIP文件打开失败 %s ", 500, err).Throw()
	}
	return Xfs{
		Root: "/",
		Afero: afero.Afero{
			Fs: zipfs.New(&r.Reader),
		},
	}
}

// NewTar tar文件系统
func NewTar(tarfile string) Xfs {
	var buf bytes.Buffer
	r := tar.NewReader(&buf)
	return Xfs{
		Root: "/",
		Afero: afero.Afero{
			Fs: tarfs.New(r),
		},
	}
}

// Encode Base64编码
func Encode(content []byte) string {
	return base64.StdEncoding.EncodeToString(content)
}

// Decode Base64编码
func Decode(content []byte) []byte {
	var data []byte
	_, err := base64.StdEncoding.Decode(data, content)
	if err != nil {
		exception.New("文件解码失败 %s ", 500, err).Throw()
	}
	return data
}

// DecodeString Base64编码
func DecodeString(content string) string {
	dst, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		exception.New("文件解码失败 %s ", 500, err).Throw()
	}
	return string(dst)
}

// MustOpen 打开文件
func (xfs *Xfs) MustOpen(filename string) File {
	file, err := xfs.Open(filename)
	if err != nil {
		exception.New("打开文件失败 %s 失败 ", 500, filename).Throw()
	}
	return file
}

// MustReadFile 阅读文件内容
func (xfs *Xfs) MustReadFile(filename string) []byte {
	bytes, err := xfs.ReadFile(filename)
	if err != nil {
		exception.New("阅读文件内容失败 %s 失败 %s ", 500, filename, err).Throw()
	}
	return bytes
}

// MustReadDir 阅读文件夹内容
func (xfs *Xfs) MustReadDir(dirname string) []os.FileInfo {
	fileinfos, err := xfs.ReadDir(dirname)
	if err != nil {
		exception.New("阅读文件夹失败 %s 失败 ", 500, dirname).Throw()
	}
	return fileinfos
}

// MustExists 检查文件是否存在
func (xfs *Xfs) MustExists(filename string) bool {
	has, err := xfs.Exists(filename)
	if err != nil {
		exception.New("检查文件失败 %s 失败 ", 500, filename).Throw()
	}
	return has
}

// MustDirExists 检查文件夹是否存在
func (xfs *Xfs) MustDirExists(dirname string) bool {
	res, err := xfs.DirExists(dirname)
	if err != nil {
		exception.New("检查文件夹 %s 失败 ", 500, dirname).Throw()
	}
	return res
}

// MustMkdirAll 创建文件夹及子文件夹
func (xfs *Xfs) MustMkdirAll(dirname string, pterm fs.FileMode) {
	err := xfs.MkdirAll(dirname, pterm)
	if err != nil {
		exception.New("创建文件夹及子文件 %s 失败 ", 500, dirname).Throw()
	}
}

// MustMkdir 创建文件夹
func (xfs *Xfs) MustMkdir(dirname string, pterm fs.FileMode) {
	err := xfs.Mkdir(dirname, pterm)
	if err != nil {
		exception.New("创建文件夹 %s 失败 ", 500, dirname).Throw()
	}
}

// MustIsDir 是否为文件夹
func (xfs *Xfs) MustIsDir(dirname string) bool {
	isdir, err := xfs.IsDir(dirname)
	if err != nil {
		exception.New("创建文件夹 %s 失败 ", 500, dirname).Throw()
	}
	return isdir
}

// MustIsEmpty 文件是否为空
func (xfs *Xfs) MustIsEmpty(filename string) bool {
	isEmpty, err := xfs.IsEmpty(filename)
	if err != nil {
		exception.New("检查文件是否为空 %s 失败 ", 500, filename).Throw()
	}
	return isEmpty

}

// GetTempDir 临时目录
func (xfs *Xfs) GetTempDir(subPath string) string {
	return afero.GetTempDir(xfs.Fs, subPath)
}

// MustTempDir 创建临时文件夹
func (xfs *Xfs) MustTempDir(dirname string, prefix string) string {
	name, err := xfs.TempDir(dirname, prefix)
	if err != nil {
		exception.New("创建文件夹 %s %s 失败 ", 500, dirname, prefix).Throw()
	}
	return name
}

// MustTempFile 临时文件
func (xfs *Xfs) MustTempFile(dirname, prefix string) File {
	f, err := xfs.TempFile(dirname, prefix)
	if err != nil {
		exception.New("创建临时文件 %s %s 失败 ", 500, dirname, prefix).Throw()
	}
	return f
}

// MustWalk 文件夹遍历
func (xfs *Xfs) MustWalk(root string, walkFn filepath.WalkFunc) {
	err := xfs.Walk(root, walkFn)
	if err != nil {
		exception.New("遍历文件夹 %s 失败 ", 500, root).Throw()
	}
}

// MustWriteFile 写入文件
func (xfs *Xfs) MustWriteFile(filename string, data []byte, perm os.FileMode) {
	err := xfs.WriteFile(filename, data, perm)
	if err != nil {
		exception.New("写入文件 %s 失败 ", 500, filename).Throw()
	}
}

// MustFileContainsBytes 文件是否包含内容
func (xfs *Xfs) MustFileContainsBytes(filename string, subslice []byte) bool {
	has, err := xfs.FileContainsBytes(filename, subslice)
	if err != nil {
		exception.New("检查文件是否包含内容 %s 失败 ", 500, filename).Throw()
	}
	return has
}

// MustSafeWriteReader 检查文件能否写入
func (xfs *Xfs) MustSafeWriteReader(filename string, r io.Reader) {
	err := xfs.SafeWriteReader(filename, r)
	if err != nil {
		exception.New("检查文件是否可以写入 Reader %s 失败 ", 500, filename).Throw()
	}
}
