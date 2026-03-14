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
	openapiComputer "github.com/yaoapp/yao/openapi/computer"
	"github.com/yaoapp/yao/openapi/dsl"
	"github.com/yaoapp/yao/openapi/file"
	"github.com/yaoapp/yao/openapi/hello"
	openintegrations "github.com/yaoapp/yao/openapi/integrations"
	"github.com/yaoapp/yao/openapi/job"
	"github.com/yaoapp/yao/openapi/kb"
	"github.com/yaoapp/yao/openapi/llm"
	"github.com/yaoapp/yao/openapi/mcp"
	"github.com/yaoapp/yao/openapi/messenger"
	"github.com/yaoapp/yao/openapi/nodes"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/acl"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/otp"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/openapi/sandbox"
	openapiTai "github.com/yaoapp/yao/openapi/tai"
	"github.com/yaoapp/yao/openapi/team"
	openapiTrace "github.com/yaoapp/yao/openapi/trace"
	"github.com/yaoapp/yao/openapi/user"
	openapiWorkspace "github.com/yaoapp/yao/openapi/workspace"
	taiapi "github.com/yaoapp/yao/tai/api"
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
		Enabled:    true,
		PathPrefix: config.BaseURL,
		Cache:      oauthConfig.Cache,
		Provider:   oauthConfig.UserProvider,
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

	// Integrations webhook handlers (public, no OAuth - external platforms push here)
	openintegrations.Attach(group.Group("/integrations"))

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

	// OTP handlers (passwordless authentication)
	otp.Attach(group.Group("/otp"), openapi.OAuth)

	// Sandbox handlers (VNC proxy + management CRUD)
	sandbox.SetPathPrefix(baseURL)
	sandboxGroup := group.Group("/sandbox")
	sandbox.Attach(sandboxGroup, openapi.OAuth)
	sandbox.AttachManage(sandboxGroup, openapi.OAuth)

	// Computer option handlers (for InputArea selector)
	openapiComputer.Attach(group.Group("/computer"), openapi.OAuth)

	// Workspace handlers
	openapiWorkspace.Attach(group.Group("/workspace"), openapi.OAuth)

	// Tai nodes handlers
	nodes.Attach(group.Group("/nodes"), openapi.OAuth)

	// Tai forward handlers (proxy + VNC, dispatches tunnel vs local)
	openapiTai.Attach(group)

	// Tai direct registration API (uses /tai-nodes/ prefix to avoid routing conflict with /tai/:taiID/)
	group.POST("/tai-nodes/register", taiapi.HandleRegister)
	group.POST("/tai-nodes/heartbeat", taiapi.HandleHeartbeat)
	group.DELETE("/tai-nodes/register/:tai_id", taiapi.HandleUnregister)

	// Custom handlers (Defined by developer)

}
