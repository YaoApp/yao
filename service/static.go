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
	"github.com/yaoapp/yao/sui/core"
)

// AppFileServer static file server
var AppFileServer http.Handler

// CUIFileServerV1 CUI v1.0
var CUIFileServerV1 http.Handler = http.FileServer(data.CuiV1())

// AdminRoot cache
var AdminRoot = ""

// AdminRootLen cache
var AdminRootLen = 0

var rewriteRules = []RewriteRule{}

// RewriteRule is a URL rewrite rule defined in app.yao
type RewriteRule struct {
	Pattern     *regexp.Regexp
	Replacement string
}

// GetRewriteRules returns the loaded rewrite rules for use by other packages (e.g., sui/api)
func GetRewriteRules() []RewriteRule {
	return rewriteRules
}

// ResolveRoute applies rewrite rules to the given route path.
// Returns the rewritten path (with .sui suffix removed) and matched parameter values, or empty string if no rule matches.
func ResolveRoute(route string) (string, []string) {
	for _, rule := range rewriteRules {
		if matches := rule.Pattern.FindStringSubmatch(route); matches != nil {
			rewritten := rule.Pattern.ReplaceAllString(route, rule.Replacement)
			rewritten = strings.TrimSuffix(rewritten, ".sui")
			return rewritten, matches
		}
	}
	return "", nil
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

			rewriteRules = append(rewriteRules, RewriteRule{
				Pattern:     re,
				Replacement: replacement,
			})
		}
	}

	// Register the route resolver for $Backend().Call() dynamic route support
	core.RouteResolver = ResolveRoute
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
