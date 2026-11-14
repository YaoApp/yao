// Package tao registers the Tao cloud purchase routes.
package tao

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/config"
	modulecomputer "github.com/yaoapp/yao/modules/computer"
	taocomputer "github.com/yaoapp/yao/modules/tao/computer"
	oauthpkg "github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/types"
	taoaccount "github.com/yaoapp/yao/openapi/tao/account"
	taocomputers "github.com/yaoapp/yao/openapi/tao/computers"
	taipkg "github.com/yaoapp/yao/tai"
)

// Attach registers /v1/tao/* and starts the Computer lease sweeper.
func Attach(group *gin.RouterGroup, oauth types.OAuth) {
	computers := modulecomputer.New().WithTokens(oauthpkg.OAuth)
	go computers.RunLeaseSweeper(context.Background(), time.Minute)
	if config.Conf.Tai.Server.Enabled == "true" {
		computers.WithTaiNodeProvider(taipkg.ComputerProvider{})
	}

	service := taocomputer.NewService(taocomputer.NewClient(""), computers).
		WithTai(modulecomputer.NewSSHTaiController()).
		WithServerURL(taipkg.ServerURL()).
		WithProducts(taocomputer.ModelProductWriter{})
	if controlKey, err := taocomputer.LoadControlKey(); err != nil {
		log.Printf("[Tai] control key is not available: %v", err)
	} else {
		service.WithControlKey(controlKey)
	}
	keys := taocomputer.ModelKeyResolver{}

	taocomputers.Attach(group.Group("/computers"), oauth, service, keys)
	taoaccount.Attach(group.Group("/account"), oauth)
	taocomputers.AttachCatalog(group.Group("/catalog"), oauth, service, keys)
}
