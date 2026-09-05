package sandbox

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth/types"
	sandboxv2 "github.com/yaoapp/yao/sandbox/v2"
	taitypes "github.com/yaoapp/yao/tai/types"
)

// ExportNodeOwnedBy exposes nodeOwnedBy for testing.
var ExportNodeOwnedBy = func(snap *taitypes.NodeMeta, authInfo *types.AuthorizedInfo) bool {
	return nodeOwnedBy(snap, authInfo)
}

// ExportCheckBoxOwner exposes checkBoxOwner for testing.
var ExportCheckBoxOwner = func(c *gin.Context, box *sandboxv2.Box, owner string) bool {
	return checkBoxOwner(c, box, owner)
}
