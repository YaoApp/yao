// Package tao registers the Tao cloud purchase routes.
package tao

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	modulecomputer "github.com/yaoapp/yao/modules/computer"
	taocomputer "github.com/yaoapp/yao/modules/tao/computer"
	oauthpkg "github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/types"
	taoaccount "github.com/yaoapp/yao/openapi/tao/account"
	taocomputers "github.com/yaoapp/yao/openapi/tao/computers"
)

// Attach registers /v1/tao/* and starts the Computer lease sweeper.
func Attach(group *gin.RouterGroup, oauth types.OAuth) {
	computers := modulecomputer.New().WithTokens(oauthpkg.OAuth)
	go computers.RunLeaseSweeper(context.Background(), time.Minute)

	service := taocomputer.NewService(taocomputer.NewClient(""), computers).
		WithTai(modulecomputer.NewSSHTaiController()).
		WithSSHKeys(taocomputer.VaultSSHKeyResolver{}).
		WithProducts(taocomputer.ModelProductWriter{})
	keys := taocomputer.ModelKeyResolver{}

	taocomputers.Attach(group.Group("/computers"), oauth, service, keys)
	taoaccount.Attach(group.Group("/account"), oauth)
	taocomputers.AttachCatalog(group.Group("/catalog"), oauth, service, keys)
}
