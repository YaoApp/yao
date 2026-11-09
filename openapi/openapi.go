package openapi

import (
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi/app"
	"github.com/yaoapp/yao/openapi/captcha"
	"github.com/yaoapp/yao/openapi/dsl"
	openapiEvent "github.com/yaoapp/yao/openapi/event"
	"github.com/yaoapp/yao/openapi/file"
	"github.com/yaoapp/yao/openapi/hello"
	openintegrations "github.com/yaoapp/yao/openapi/integrations"
	"github.com/yaoapp/yao/openapi/job"
	"github.com/yaoapp/yao/openapi/mcp"
	"github.com/yaoapp/yao/openapi/messenger"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/acl"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/otp"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/openapi/team"
	"github.com/yaoapp/yao/openapi/testmode"
	"github.com/yaoapp/yao/openapi/user"
)

// Server is the OpenAPI server
var Server *OpenAPI = nil

// OpenAPI is the OpenAPI server
type OpenAPI struct {
	Config *Config     // OpenAPI configuration
	OAuth  types.OAuth // OAuth service interface
}

// Load loads the OpenAPI server from the configuration
func Load(appConfig config.Config) (*OpenAPI, error) {

	var configPath string = filepath.Join("openapi", "openapi.yao")
	var configRaw, err = application.App.Read(configPath)
	if err != nil {
		return nil, err
	}

	// Parse the configuration
	var config Config
	err = application.Parse(configPath, configRaw, &config)
	if err != nil {
		return nil, err
	}

	// Convert the configuration to an OAuth configuration
	oauthConfig, err := config.OAuthConfig(appConfig)
	if err != nil {
		return nil, err
	}

	// Create the OAuth service
	oauthService, err := oauth.NewService(oauthConfig)
	if err != nil {
		return nil, err
	}

	// Set the secure cookie configuration for the response package
	// This determines whether to use __Host- prefix and Secure flag for cookies
	response.SetSecureCookieEnabled(oauthConfig.Security.SecureCookie)

	// Load user configurations
	err = user.Load(appConfig)
	if err != nil {
		return nil, err
	}

	// Load the ACL enforcer
	_, err = acl.Load(&acl.Config{
		Enabled:        true,
		PathPrefix:     config.BaseURL,
		Cache:          oauthConfig.Cache,
		Provider:       oauthConfig.UserProvider,
		ClientProvider: oauthConfig.ClientProvider,
	})
	if err != nil {
		return nil, err
	}

	// Initialize OTP service (shares the OAuth store)
	otp.NewService(oauthService.GetStore(), oauthService.GetKeyPrefix())

	// Create the OpenAPI server
	Server = &OpenAPI{Config: &config, OAuth: oauthService}
	return Server, nil
}

// Attach attaches the OpenAPI server to the router
func (openapi *OpenAPI) Attach(router *gin.Engine) {

	// Ignore if the OpenAPI server is not configured
	if openapi.Config == nil {
		return
	}

	// Basic Groups
	baseURL := openapi.Config.BaseURL
	group := router.Group(baseURL)

	// Well-known handlers
	openapi.attachWellKnown(router)

	// OAuth handlers
	openapi.attachOAuth(group)

	// Hello World handlers
	hello.Attach(group.Group("/helloworld"), openapi.OAuth)

	// DSL handlers
	dsl.Attach(group.Group("/dsl"), openapi.OAuth)

	// File handlers
	file.Attach(group.Group("/file"), openapi.OAuth)

	// Job Management handlers
	job.Attach(group.Group("/job"), openapi.OAuth)

	// Captcha handlers
	captcha.Attach(group.Group("/captcha"), openapi.OAuth)

	// User handlers
	user.Attach(group.Group("/user"), openapi.OAuth)

	// Team handlers
	team.Attach(group.Group("/team"), openapi.OAuth)

	// Messenger webhook handlers
	messenger.Attach(group.Group("/messenger"), openapi.OAuth)

	// Integrations webhook handlers (public, no OAuth - external platforms push here)
	openintegrations.Attach(group.Group("/integrations"))

	// System event WebSocket (top-level, not under /agent)
	openapiEvent.Attach(group.Group("/events"), openapi.OAuth)

	// MCP Server handlers
	mcp.Attach(group.Group("/mcp"), openapi.OAuth)

	// App handlers (menu, etc.)
	app.Attach(group.Group("/app"), openapi.OAuth)

	// OTP handlers (passwordless authentication)
	otp.Attach(group.Group("/otp"), openapi.OAuth)

	// Test mode endpoints (public, no OAuth guard)
	// Blocked unconditionally in production mode regardless of TestMode flag.
	if config.Conf.TestMode && config.Conf.Mode != "production" {
		testmode.Attach(group.Group("/test"))
	}

	// Custom handlers (Defined by developer)

}
