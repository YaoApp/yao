package openapi

import (
	"path/filepath"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/config"
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
func Load(appConfig *config.Config) (*OpenAPI, error) {

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
