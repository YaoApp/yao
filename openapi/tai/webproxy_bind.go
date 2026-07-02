package tai

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/tai/webproxy"
)

// BindAndRespond is the shared core for single-port binding.
// It binds the port, retrieves the access token, and returns a flat JSON response.
// Used by both POST /tai/:taiID/webproxy/bindings (single port mode)
// and POST /agent/tasks/:chatid/proxy.
func BindAndRespond(c *gin.Context, opts webproxy.BindOptions) {
	info, err := webproxy.WP().Bind(opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	domain, prefix := webproxy.WP().GetConfig()

	token := ""
	if oauth.OAuth != nil {
		token = oauth.OAuth.GetAccessToken(c)
	}

	c.JSON(http.StatusOK, gin.H{
		"host_port":   info.HostPort,
		"target_port": info.TargetPort,
		"label":       info.Label,
		"status":      info.Status,
		"url":         BuildProxyURL(info.HostPort, domain, prefix),
		"token":       token,
	})
}

// BuildProxyURL constructs the proxy URL from host port and config.
func BuildProxyURL(hostPort int, domain, prefix string) string {
	if domain != "" {
		return fmt.Sprintf("http://%s%d.%s", prefix, hostPort, domain)
	}
	return fmt.Sprintf("http://127.0.0.1:%d", hostPort)
}
