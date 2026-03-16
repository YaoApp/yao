package service

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/gou/server/http"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi"
	servicelog "github.com/yaoapp/yao/service/log"
	"github.com/yaoapp/yao/share"
)

// Router holds the active gin.Engine so the gRPC API proxy can forward
// requests internally without an HTTP round-trip.
var Router *gin.Engine

// ServerHooks allows the caller to inject gRPC (or other) server lifecycle
// without creating import cycles.
type ServerHooks struct {
	Start func(cfg config.Config) error // called before HTTP starts; nil = skip
	Stop  func()                        // called on shutdown; nil = skip
	Addrs func() []string               // returns listen addresses; nil = skip
}

// Service manages HTTP and optional gRPC servers as a single unit.
type Service struct {
	http  *http.Server
	hooks ServerHooks
}

// Start launches optional hook servers (e.g. gRPC) and the HTTP server.
// Returns a Service handle for shutdown coordination.
func Start(cfg config.Config, hooks ...ServerHooks) (*Service, error) {

	if cfg.AllowFrom == nil {
		cfg.AllowFrom = []string{}
	}

	err := prepare()
	if err != nil {
		return nil, err
	}

	var h ServerHooks
	if len(hooks) > 0 {
		h = hooks[0]
	}

	// Start hook server (gRPC, etc.)
	if h.Start != nil {
		if err := h.Start(cfg); err != nil {
			return nil, err
		}
	}

	router := gin.New()
	Router = router
	router.Use(Middlewares...)

	var apiRoot string
	if openapi.Server != nil {
		apiRoot = openapi.Server.Config.BaseURL
		api.SetGuards(OpenAPIGuards())
		router.Any(apiRoot+"/api/*path", DynamicAPIHandler)
		api.SetRoutes(router, apiRoot, cfg.AllowFrom...)
		api.BuildRouteTable()
		openapi.Server.Attach(router)
	} else {
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

	// Start HTTP in background; wait for the first event to confirm
	// the port is bound before returning.
	go func() {
		srv.Start()
	}()

	// Block until HTTP reports READY or ERROR
	ev := <-srv.Event()
	if ev != http.READY {
		if h.Stop != nil {
			h.Stop()
		}
		return nil, fmt.Errorf("HTTP server failed to start on %s:%d", cfg.Host, cfg.Port)
	}

	return &Service{http: srv, hooks: h}, nil
}

// Event returns the HTTP server event channel (READY, CLOSED, ERROR).
func (s *Service) Event() chan uint8 {
	return s.http.Event()
}

// Stop shuts down hook servers (gRPC, etc.) then signals the HTTP server to close.
func (s *Service) Stop() {
	if s.hooks.Stop != nil {
		s.hooks.Stop()
	}
	s.http.Stop()
}

// HookAddrs returns the hook server listen addresses (e.g. gRPC addresses).
func (s *Service) HookAddrs() []string {
	if s.hooks.Addrs != nil {
		return s.hooks.Addrs()
	}
	return nil
}

// Watch starts file watching in development mode. Blocking; run in a goroutine.
func (s *Service) Watch(done chan uint8) {
	watch(s, done)
}

// Restart the HTTP server with a fresh router (hook servers stay running).
func Restart(svc *Service, cfg config.Config) error {
	router := gin.New()
	Router = router
	router.Use(Middlewares...)

	if openapi.Server != nil {
		baseURL := openapi.Server.Config.BaseURL
		api.SetGuards(OpenAPIGuards())
		router.Any(baseURL+"/api/*path", DynamicAPIHandler)
		api.SetRoutes(router, baseURL, cfg.AllowFrom...)
		api.BuildRouteTable()
		openapi.Server.Attach(router)
	} else {
		api.SetGuards(Guards)
		api.SetRoutes(router, "/api", cfg.AllowFrom...)
	}

	svc.http.Reset(router)
	return svc.http.Restart()
}

func prepare() error {
	servicelog.InitAccessLog(config.Conf.Root)

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
