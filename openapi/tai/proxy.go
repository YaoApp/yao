package tai

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	yaoTai "github.com/yaoapp/yao/tai"
)

// handleLocalProxy resolves the container's HTTP address via Docker socket
// and reverse-proxies the request.
func handleLocalProxy(c *gin.Context, taiID string) {
	res, ok := yaoTai.GetResources(taiID)
	if !ok || res.Proxy == nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "proxy not available for node " + taiID})
		return
	}

	// path format: /{containerID}:{port}/{rest...}
	raw := strings.TrimPrefix(c.Param("path"), "/")
	colonIdx := strings.Index(raw, ":")
	if colonIdx < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid proxy path, expected /{containerID}:{port}/{path}"})
		return
	}

	containerID := raw[:colonIdx]
	rest := raw[colonIdx+1:]
	slashIdx := strings.Index(rest, "/")
	var portStr, subPath string
	if slashIdx >= 0 {
		portStr = rest[:slashIdx]
		subPath = rest[slashIdx:]
	} else {
		portStr = rest
		subPath = "/"
	}

	var port int
	for _, ch := range portStr {
		if ch < '0' || ch > '9' {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid port in proxy path"})
			return
		}
		port = port*10 + int(ch-'0')
	}
	if port == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing port in proxy path"})
		return
	}

	targetURL, err := res.Proxy.URL(c.Request.Context(), containerID, port, subPath)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "resolve proxy target: " + err.Error()})
		return
	}

	reverseProxy(c, targetURL)
}
