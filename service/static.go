package service

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/yaoapp/yao/data"
	"github.com/yaoapp/yao/service/fs"
	"github.com/yaoapp/yao/share"
)

// AppFileServer static file server
var AppFileServer http.Handler

// XGenFileServerV1 XGen v1.0
var XGenFileServerV1 http.Handler = http.FileServer(data.XgenV1())

// AdminRoot cache
var AdminRoot = ""

// AdminRootLen cache
var AdminRootLen = 0

// SetupStatic setup static file server
func SetupStatic() error {

	// SetAdmin Root
	adminRoot()

	if isPWA() {
		AppFileServer = http.FileServer(fs.DirPWA("public"))
		return nil
	}

	AppFileServer = http.FileServer(fs.Dir("public"))
	return nil
}

// rewrite path
func isPWA() bool {
	if share.App.Static == nil {
		return false
	}
	return share.App.Static.PWA
}

// SetupAdmin setup admin static root
func adminRoot() (string, int) {
	if AdminRoot != "" {
		return AdminRoot, AdminRootLen
	}

	adminRoot := "/yao/"
	if share.App.AdminRoot != "" {
		root := strings.TrimPrefix(share.App.AdminRoot, "/")
		root = strings.TrimSuffix(root, "/")
		adminRoot = fmt.Sprintf("/%s/", root)
	}
	adminRootLen := len(adminRoot)
	AdminRoot = adminRoot
	AdminRootLen = adminRootLen
	return AdminRoot, AdminRootLen
}
