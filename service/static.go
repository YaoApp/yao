package service

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/yaoapp/kun/log"
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

var rewriteRules = []rewriteRule{}

type rewriteRule struct {
	Pattern     *regexp.Regexp
	Replacement string
}

// SetupStatic setup static file server
func SetupStatic() error {
	setupAdminRoot()
	setupRewrite()

	// Disable gzip compression for static files
	if share.App.Static.DisableGzip {
		AppFileServer = http.FileServer(fs.Dir("public"))
		return nil
	}

	AppFileServer = gzipHandler(http.FileServer(fs.Dir("public")))
	return nil
}

func setupRewrite() {
	if share.App.Static.Rewrite != nil {
		for _, rule := range share.App.Static.Rewrite {

			pattern := ""
			replacement := ""
			for key, value := range rule {
				pattern = key
				replacement = value
				break
			}

			re, err := regexp.Compile(pattern)
			if err != nil {
				log.Error("Invalid rewrite rule: %s", pattern)
				continue
			}

			rewriteRules = append(rewriteRules, rewriteRule{
				Pattern:     re,
				Replacement: replacement,
			})
		}
	}
}

// rewrite path
func setupAdminRoot() (string, int) {
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
