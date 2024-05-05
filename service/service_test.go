package service

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/test"
)

func TestStartStop(t *testing.T) {

	gin.SetMode(gin.ReleaseMode)

	cfg := config.Conf
	cfg.Port = 0
	err := engine.Load(cfg, engine.LoadOption{})
	if err != nil {
		t.Fatal(err)
	}

	srv, err := Start(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer Stop(srv)

	<-srv.Event()
	if !srv.Ready() {
		t.Fatal("server not ready")
	}

	port, err := srv.Port()
	if err != nil {
		t.Fatal(err)
	}

	if port <= 0 {
		t.Fatal("invalid port")
	}

	// API Server
	req := test.NewRequest(port).Route("/api/__yao/app/setting")
	res, err := req.Get()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 200, res.Status())
	data, err := res.Map()
	if err != nil {
		t.Fatal(err)
	}
	// assert.Equal(t, "Demo Application", data["name"])
	assert.True(t, len(data["name"].(string)) > 0)

	// Public
	req = test.NewRequest(port).Route("/")
	res, err = req.Get()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 200, res.Status())
	assert.Equal(t, "Hello World\n", res.Body())

	// XGEN
	req = test.NewRequest(port).Route("/admin/")
	res, err = req.Get()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 200, res.Status())
	assert.Contains(t, res.Body(), "ROOT /admin/")
}
