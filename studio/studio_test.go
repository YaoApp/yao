package studio

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/network"
)

func TestLoad(t *testing.T) {
	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, dfs)

	res, err := gou.Yao.Engine.Call(map[string]interface{}{}, "__yao.studio.table", "Ping")
	assert.Nil(t, err)
	assert.Equal(t, "PONG", res)
}

func TestStartStop(t *testing.T) {
	var err error
	go func() { err = Start(config.Conf) }()
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(500 * time.Millisecond)
	Stop()
	time.Sleep(100 * time.Millisecond)
}

func TestStartStopError(t *testing.T) {
	var err error
	go func() { err = Start(config.Conf) }()
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(500 * time.Millisecond)
	go func() { err = Start(config.Conf) }()
	time.Sleep(100 * time.Millisecond)
	assert.NotNil(t, err)

	Stop()
	time.Sleep(100 * time.Millisecond)
}

func TestGetAPI(t *testing.T) {

	Load(config.Conf)

	var err error
	go func() { err = Start(config.Conf) }()
	if err != nil {
		t.Fatal(err)
	}
	defer Stop()
	time.Sleep(500 * time.Millisecond)

	url := fmt.Sprintf("http://127.0.0.1:%d/dsl/readfile?name=/models/user.json", config.Conf.Studio.Port)
	res := network.RequestGet(url, nil, nil)
	assert.Equal(t, "用户", res.Data.(map[string]interface{})["name"])

	url = fmt.Sprintf("http://127.0.0.1:%d/dsl/readdir?name=/models", config.Conf.Studio.Port)
	res = network.RequestGet(url, nil, nil)
	assert.Equal(t, 11, len(res.Data.([]interface{})))

	url = fmt.Sprintf("http://127.0.0.1:%d/dsl/readdir?name=/models&recursive=1", config.Conf.Studio.Port)
	res = network.RequestGet(url, nil, nil)
	assert.Equal(t, 12, len(res.Data.([]interface{})))

}
