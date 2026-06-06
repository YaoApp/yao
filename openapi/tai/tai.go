package tai

import (
	"github.com/gin-gonic/gin"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	yaoTai "github.com/yaoapp/yao/tai"
	taitunnel "github.com/yaoapp/yao/tai/tunnel"
)

// Attach registers Tai forward routes on the given group.
//
//   - ANY /tai/:taiID/proxy/*path            — HTTP forward (tunnel or local)
//   - GET /tai/:taiID/vnc/*path              — VNC WebSocket forward (tunnel or local)
//   - GET /tai/:taiID/webproxy/bindings      — list webproxy bindings (OAuth protected)
//   - POST /tai/:taiID/webproxy/bindings     — create webproxy binding(s) (OAuth protected)
//   - DELETE /tai/:taiID/webproxy/bindings/:hostPort — delete webproxy binding (OAuth protected)
func Attach(group *gin.RouterGroup, oauth oauthTypes.OAuth) {
	group.Any("/tai/:taiID/proxy/*path", handleProxy)
	group.GET("/tai/:taiID/vnc/*path", handleVNC)

	group.GET("/tai/:taiID/webproxy/bindings", oauth.Guard, handleListBindings)
	group.POST("/tai/:taiID/webproxy/bindings", oauth.Guard, handleCreateBindings)
	group.DELETE("/tai/:taiID/webproxy/bindings/:hostPort", oauth.Guard, handleDeleteBinding)
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
