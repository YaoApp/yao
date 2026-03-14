package tai

import (
	"github.com/gin-gonic/gin"
	yaoTai "github.com/yaoapp/yao/tai"
	taitunnel "github.com/yaoapp/yao/tai/tunnel"
)

// Attach registers Tai forward routes on the given group.
//
//   - ANY /tai/:taiID/proxy/*path — HTTP forward (tunnel or local)
//   - GET /tai/:taiID/vnc/*path   — VNC WebSocket forward (tunnel or local)
func Attach(group *gin.RouterGroup) {
	group.Any("/tai/:taiID/proxy/*path", handleProxy)
	group.GET("/tai/:taiID/vnc/*path", handleVNC)
}

func handleProxy(c *gin.Context) {
	taiID := c.Param("taiID")
	if isLocalNode(taiID) {
		handleLocalProxy(c, taiID)
		return
	}
	taitunnel.HandleForwardLazy(c)
}

func handleVNC(c *gin.Context) {
	taiID := c.Param("taiID")
	if isLocalNode(taiID) {
		handleLocalVNC(c, taiID)
		return
	}
	taitunnel.HandleForwardLazy(c)
}

func isLocalNode(taiID string) bool {
	meta, ok := yaoTai.GetNodeMeta(taiID)
	return ok && meta.Mode == "local"
}
