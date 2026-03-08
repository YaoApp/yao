package service

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/test"
)

func TestStartStop(t *testing.T) {

	gin.SetMode(gin.ReleaseMode)

	cfg := config.Conf
	cfg.Port = 0
	_, err := engine.Load(cfg, engine.LoadOption{})
	if err != nil {
		t.Fatal(err)
	}

	// Temporarily disable OpenAPI for this test
	savedOpenAPIServer := openapi.Server
	openapi.Server = nil
	defer func() { openapi.Server = savedOpenAPIServer }()

	srv, err := Start(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	<-srv.Event()

	// API Server
	req := test.NewRequest(cfg.Port).Route("/api/__yao/app/setting")
	res, err := req.Get()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 200, res.Status())
	data, err := res.Map()
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, len(data["name"].(string)) > 0)

	// Public
	req = test.NewRequest(cfg.Port).Route("/")
	res, err = req.Get()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 200, res.Status())
	assert.Equal(t, "Hello World\n", res.Body())

	// XGEN
	req = test.NewRequest(cfg.Port).Route("/admin/")
	res, err = req.Get()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 200, res.Status())
	assert.Contains(t, res.Body(), "ROOT /admin/")
}
