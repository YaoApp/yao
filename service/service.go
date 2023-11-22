package service

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/gou/server/http"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/neo"
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
	api.SetGuards(Guards)
	api.SetRoutes(router, "/api", cfg.AllowFrom...)
	srv := http.New(router, http.Option{
		Host:    cfg.Host,
		Port:    cfg.Port,
		Root:    "/api",
		Allows:  cfg.AllowFrom,
		Timeout: 5 * time.Second,
	})

	// Neo API
	if neo.Neo != nil {
		neo.Neo.API(router, "/api/__yao/neo")
	}

	go func() {
		err = srv.Start()
	}()

	return srv, nil
}

// Restart the yao service
func Restart(srv *http.Server, cfg config.Config) error {
	router := gin.New()
	router.Use(Middlewares...)
	api.SetGuards(Guards)
	api.SetRoutes(router, "/api", cfg.AllowFrom...)
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
