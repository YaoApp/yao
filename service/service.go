package service

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/gou/server/http"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/share"
)

// Start the yao service
func Start(cfg config.Config) (*http.Server, error) {

	if cfg.AllowFrom == nil {
		cfg.AllowFrom = []string{}
	}

	err := prepare()
	if err != nil {
		return nil, err
	}

	router := gin.New()
	router.Use(Middlewares...)

	var apiRoot string
	if openapi.Server != nil {
		// OpenAPI mode: use OAuth guards and dynamic routing
		apiRoot = openapi.Server.Config.BaseURL
		api.SetGuards(OpenAPIGuards())

		// Developer APIs: use dynamic proxy (supports hot-reload)
		router.Any(apiRoot+"/api/*path", DynamicAPIHandler)

		// Widgets and system APIs: static registration
		api.SetRoutes(router, apiRoot, cfg.AllowFrom...)

		// Build route table for dynamic lookup
		api.BuildRouteTable()

		// Attach OpenAPI built-in features
		openapi.Server.Attach(router)
	} else {
		// Traditional mode: unchanged
		apiRoot = "/api"
		api.SetGuards(Guards)
		api.SetRoutes(router, "/api", cfg.AllowFrom...)
	}

	srv := http.New(router, http.Option{
		Host:    cfg.Host,
		Port:    cfg.Port,
		Root:    apiRoot,
		Allows:  cfg.AllowFrom,
		Timeout: 5 * time.Second,
	})

	go func() {
		err = srv.Start()
	}()

	return srv, nil
}

// Restart the yao service
func Restart(srv *http.Server, cfg config.Config) error {
	router := gin.New()
	router.Use(Middlewares...)

	if openapi.Server != nil {
		// OpenAPI mode
		baseURL := openapi.Server.Config.BaseURL
		api.SetGuards(OpenAPIGuards())
		router.Any(baseURL+"/api/*path", DynamicAPIHandler)
		api.SetRoutes(router, baseURL, cfg.AllowFrom...)
		api.BuildRouteTable()
		openapi.Server.Attach(router)
	} else {
		// Traditional mode: unchanged
		api.SetGuards(Guards)
		api.SetRoutes(router, "/api", cfg.AllowFrom...)
	}

	srv.Reset(router)
	return srv.Restart()
}

// Stop the yao service
func Stop(srv *http.Server) error {
	err := srv.Stop()
	if err != nil {
		return err
	}
	<-srv.Event()
	return nil
}

func prepare() error {

	// Session server
	err := share.SessionStart()
	if err != nil {
		return err
	}

	err = SetupStatic()
	if err != nil {
		return err
	}

	return nil
}
