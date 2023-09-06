package service

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/yaoapp/yao/data"
	"github.com/yaoapp/yao/service/fs"
	"github.com/yaoapp/yao/share"
)

// AppFileServer static file server
var AppFileServer http.Handler

// spaFileServers spa static file server
var spaFileServers map[string]http.Handler = map[string]http.Handler{}

// SpaRoots SPA static file server
var SpaRoots map[string]int = map[string]int{}

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

	for _, root := range spaApps() {
		spaFileServers[root] = http.FileServer(fs.DirPWA(filepath.Join("public", root)))
		SpaRoots[root] = len(root)
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

// rewrite path
func spaApps() []string {
	if share.App.Static == nil {
		return []string{}
	}
	return share.App.Static.Apps
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
