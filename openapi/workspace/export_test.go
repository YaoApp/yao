package workspace

import (
	"github.com/gin-gonic/gin"
	ws "github.com/yaoapp/yao/workspace"
)

// ExportCheckWSOwner exposes checkWSOwner for testing.
var ExportCheckWSOwner = func(c *gin.Context, w *ws.Workspace, owner string) bool {
	return checkWSOwner(c, w, owner)
}
