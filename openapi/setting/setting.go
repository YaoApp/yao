package setting

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	gouStore "github.com/yaoapp/gou/store"
	"github.com/yaoapp/kun/log"
	oauth "github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

const ownerCachePrefix = "setting:owner:"
const ownerCacheTTL = 5 * time.Minute

func getCache() gouStore.Store {
	c, _ := gouStore.Get("__yao.cache")
	return c
}

// Attach registers all /setting/* routes under the given group.
// Currently only System Info routes are wired; other groups will be
// added incrementally.
func Attach(group *gin.RouterGroup, oauth oauthTypes.OAuth) {
	group.Use(oauth.Guard)

	sys := group.Group("/system")
	sys.GET("", handleSystemInfo)
	sys.POST("/check-update", handleSystemCheckUpdate)

	cloud := group.Group("/cloud")
	cloud.GET("", handleCloudGet)
	cloud.PUT("", handleCloudUpdate)
	cloud.POST("/test", handleCloudTest)
	cloud.POST("/refresh", handleCloudRefresh)

	llm := group.Group("/llm")
	llm.GET("", handleLLMGet)
	llm.PUT("/roles", handleLLMRoles)
	llm.POST("/test", handleLLMTest)
	llm.POST("/providers", handleLLMProviderCreate)
	llm.PUT("/providers/:key", handleLLMProviderUpdate)
	llm.DELETE("/providers/:key", handleLLMProviderDelete)
	llm.POST("/providers/:key/test", handleLLMProviderTest)

	search := group.Group("/search")
	search.GET("", handleSearchGet)
	search.PUT("/providers/:key", handleSearchProviderUpdate)
	search.PUT("/providers/:key/toggle", handleSearchProviderToggle)
	search.POST("/providers/:key/test", handleSearchProviderTest)
	search.PUT("/tool-assignment", handleSearchToolAssignment)

	smtpG := group.Group("/smtp")
	smtpG.GET("", handleSmtpGet)
	smtpG.PUT("", handleSmtpUpdate)
	smtpG.PUT("/toggle", handleSmtpToggle)
	smtpG.POST("/test", handleSmtpTest)

	mcpG := group.Group("/mcp")
	mcpG.GET("/servers", handleMCPList)
	mcpG.POST("/servers", handleMCPCreate)
	mcpG.PUT("/servers/:id", handleMCPUpdate)
	mcpG.DELETE("/servers/:id", handleMCPDelete)
	mcpG.POST("/test", handleMCPTest)

	sb := group.Group("/sandbox")
	sb.GET("", handleSandboxGet)
	sb.PUT("/registry", handleSandboxRegistry)
	sb.POST("/nodes/:nodeId/images/:imageId/pull", handleSandboxPull)
	sb.POST("/nodes/:nodeId/images/pull-all", handleSandboxPullAll)
	sb.DELETE("/nodes/:nodeId/images/:imageId", handleSandboxImageDelete)
	sb.POST("/nodes/:nodeId/check-docker", handleSandboxCheckDocker)

	group.GET("/setup-status", handleSetupStatus)
	group.GET("/setup-status/assistant/:id", handleAssistantSetupStatus)

	pref := group.Group("/preference")
	pref.GET("", handlePreferenceGet)
	pref.PUT("", handlePreferenceUpdate)
}

// requireOwner checks that the current user is the team owner.
// Non-team context (TeamID == ""): always allowed — user is managing their own data.
// Team context: checks cache first, then queries the member table is_owner field.
// Use this as a guard for any write operation across all /setting/* groups.
func requireOwner(c *gin.Context, info *oauthTypes.AuthorizedInfo) error {
	if info == nil || info.UserID == "" {
		return fmt.Errorf("authentication required")
	}
	if info.TeamID == "" {
		return nil
	}

	cacheKey := ownerCachePrefix + info.TeamID + ":" + info.UserID

	if cache := getCache(); cache != nil {
		if val, ok := cache.Get(cacheKey); ok {
			if isOwner, ok := val.(bool); ok {
				if isOwner {
					return nil
				}
				return fmt.Errorf("access denied: only team owner can modify settings")
			}
		}
	}

	if oauth.OAuth == nil {
		return fmt.Errorf("service not initialized")
	}
	provider, err := oauth.OAuth.GetUserProvider()
	if err != nil {
		return fmt.Errorf("service not available")
	}

	member, err := provider.GetMember(c.Request.Context(), info.TeamID, info.UserID)
	if err != nil {
		log.Error("[setting] GetMember failed: %v", err)
		return fmt.Errorf("access denied")
	}

	isOwner := checkIsOwner(member["is_owner"])
	if cache := getCache(); cache != nil {
		cache.Set(cacheKey, isOwner, ownerCacheTTL)
	}

	if isOwner {
		return nil
	}
	return fmt.Errorf("access denied: only team owner can modify settings")
}

func checkIsOwner(val interface{}) bool {
	switch v := val.(type) {
	case bool:
		return v
	case int:
		return v == 1
	case int64:
		return v == 1
	case float64:
		return v == 1
	}
	return false
}

// guardOwner is a convenience wrapper: calls requireOwner and writes 403 on failure.
// Returns true if the request should continue, false if it was aborted.
func guardOwner(c *gin.Context) bool {
	info := authorized.GetInfo(c)
	if err := requireOwner(c, info); err != nil {
		respondError(c, http.StatusForbidden, err.Error())
		return false
	}
	return true
}

// respondError is a thin helper that writes a JSON error via the shared
// response package.
func respondError(c *gin.Context, status int, msg string) {
	response.RespondWithError(c, status, &response.ErrorResponse{
		Code:             "server_error",
		ErrorDescription: msg,
	})
}
