package openapi

import (
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi/agent"
	"github.com/yaoapp/yao/openapi/app"
	"github.com/yaoapp/yao/openapi/captcha"
	"github.com/yaoapp/yao/openapi/chat"
	"github.com/yaoapp/yao/openapi/dsl"
	"github.com/yaoapp/yao/openapi/file"
	"github.com/yaoapp/yao/openapi/hello"
	"github.com/yaoapp/yao/openapi/job"
	"github.com/yaoapp/yao/openapi/kb"
	"github.com/yaoapp/yao/openapi/llm"
	"github.com/yaoapp/yao/openapi/mcp"
	"github.com/yaoapp/yao/openapi/messenger"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/acl"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/team"
	openapiTrace "github.com/yaoapp/yao/openapi/trace"
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

	// Load user configurations
	err = user.Load(appConfig)
	if err != nil {
		return nil, err
	}

	// Load the ACL enforcer
	_, err = acl.Load(&acl.Config{
		Enabled:    true,
		PathPrefix: config.BaseURL,
		Cache:      oauthConfig.Cache,
		Provider:   oauthConfig.UserProvider,
	})
	if err != nil {
		return nil, err
	}

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

	// Models ( LLM Agent )
	group.GET("/models", openapi.OAuth.Guard, agent.GetModels)

	// Get Model Details ( LLM Agent )
	group.GET("/models/:model_name", openapi.OAuth.Guard, agent.GetModelDetails)

	// OAuth handlers
	openapi.attachOAuth(group)

	// Hello World handlers
	hello.Attach(group.Group("/helloworld"), openapi.OAuth)

	// DSL handlers
	dsl.Attach(group.Group("/dsl"), openapi.OAuth)

	// File handlers
	file.Attach(group.Group("/file"), openapi.OAuth)

	// Knowledge Base handlers
	kb.Attach(group.Group("/kb"), openapi.OAuth)

	// Job Management handlers
	job.Attach(group.Group("/job"), openapi.OAuth)

	// Chat handlers
	chat.Attach(group.Group("/chat"), openapi.OAuth)

	// Captcha handlers
	captcha.Attach(group.Group("/captcha"), openapi.OAuth)

	// User handlers
	user.Attach(group.Group("/user"), openapi.OAuth)

	// Team handlers
	team.Attach(group.Group("/team"), openapi.OAuth)

	// Messenger webhook handlers
	messenger.Attach(group.Group("/messenger"), openapi.OAuth)

	// Agent handlers
	agent.Attach(group.Group("/agent"), openapi.OAuth)

	// LLM Provider handlers
	llm.Attach(group.Group("/llm"), openapi.OAuth)

	// MCP Server handlers
	mcp.Attach(group.Group("/mcp"), openapi.OAuth)

	// Trace handlers
	openapiTrace.Attach(group.Group("/trace"), openapi.OAuth)

	// App handlers (menu, etc.)
	app.Attach(group.Group("/app"), openapi.OAuth)

	// Custom handlers (Defined by developer)

}
