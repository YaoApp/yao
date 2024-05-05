package service

import (
	"testing"
	"time"

	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
)

func TestWatch(t *testing.T) {
	err := engine.Load(config.Conf, engine.LoadOption{})
	if err != nil {
		t.Fatal(err)
	}

	srv, err := Start(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
	defer Stop(srv)

	done := make(chan uint8, 1)
	go Watch(srv, done)

	select {
	case <-time.After(200 * time.Millisecond):
		done <- 1
		return
	}
}
