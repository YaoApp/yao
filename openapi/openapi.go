package openapi

import (
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi/dsl"
	"github.com/yaoapp/yao/openapi/hello"
	"github.com/yaoapp/yao/openapi/kb"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/types"
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

	// Knowledge Base handlers
	kb.Attach(group.Group("/kb"), openapi.OAuth)

	// Custom handlers (Defined by developer)
}
