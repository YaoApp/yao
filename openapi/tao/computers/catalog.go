package computers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	taocomputer "github.com/yaoapp/yao/modules/tao/computer"
	"github.com/yaoapp/yao/openapi/oauth/acl/role"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

// AttachCatalog registers POST /v1/tao/catalog/sync.
func AttachCatalog(group *gin.RouterGroup, oauthSvc oauthtypes.OAuth, service *taocomputer.Service, keys taocomputer.KeyResolver) {
	h := &Handler{Service: service, Keys: keys}
	group.POST("/sync", oauthSvc.Guard, h.SyncCatalog)
}

// SyncCatalog syncs Tao Catalog into Yao products.
func (h *Handler) SyncCatalog(c *gin.Context) {
	info := authorized.GetInfo(c)
	if info == nil || info.UserID == "" || !isAdmin(c, info.UserID) {
		forbidden(c)
		return
	}
	key, err := h.resolveKey(c, info)
	if err != nil {
		forbidden(c)
		return
	}
	count, err := h.Service.SyncCatalog(c.Request.Context(), key)
	if err != nil {
		apiError(c, err)
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"synced": count})
}

func isAdmin(c *gin.Context, userID string) bool {
	if role.RoleManager == nil {
		return false
	}
	roleID, err := role.RoleManager.GetUserRole(c.Request.Context(), userID)
	return err == nil && (roleID == "system:root" || roleID == "system:admin")
}
