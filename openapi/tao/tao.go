// Package tao registers the Tao cloud purchase routes.
package tao

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/config"
	modulecomputer "github.com/yaoapp/yao/modules/computer"
	"github.com/yaoapp/yao/modules/product"
	taocomputer "github.com/yaoapp/yao/modules/tao/computer"
	oauthpkg "github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/types"
	taoaccount "github.com/yaoapp/yao/openapi/tao/account"
	taocomputers "github.com/yaoapp/yao/openapi/tao/computers"
	taipkg "github.com/yaoapp/yao/tai"
)

// Attach registers /v1/tao/* and starts the Computer lease sweeper.
func Attach(group *gin.RouterGroup, oauth types.OAuth, products *product.Service) {
	computers := modulecomputer.New().WithTokens(oauthpkg.OAuth)
	go computers.RunLeaseSweeper(context.Background(), time.Minute)
	if config.Conf.Tai.Server.Enabled == "true" {
		computers.WithTaiNodeProvider(taipkg.ComputerProvider{})
	}

	serverURL, serverURLErr := taipkg.CloudServerURL()
	if serverURLErr != nil {
		log.Printf("[Tai] cloud-reachable server URL is not configured: %v", serverURLErr)
	}
	service := taocomputer.NewService(taocomputer.NewClient(""), computers).
		WithTai(modulecomputer.NewSSHTaiController()).
		WithServerURL(serverURL).
		WithProducts(taocomputer.ModelProductWriter{})
	if controlKey, err := taocomputer.LoadControlKey(); err != nil {
		log.Printf("[Tai] control key is not available: %v", err)
	} else {
		service.WithControlKey(controlKey)
	}
	keys := taocomputer.ModelKeyResolver{}
	service.WithCloudTasks(taocomputer.CloudTaskModelStore()).WithTaskKeyResolver(keys)
	service.StartCloudWorker(context.Background())
	products.Register(newTaoProductProvider(service, keys))

	taocomputers.Attach(group.Group("/computers"), oauth, service, keys)
	taoaccount.Attach(group.Group("/account"), oauth)
	taocomputers.AttachCatalog(group.Group("/catalog"), oauth, service, keys)
}
