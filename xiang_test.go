package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/kun/utils"
)

func TestCommand(t *testing.T) {

	go main()

	// 发送请求
	request := func() (maps.MapStr, error) {
		time.Sleep(time.Microsecond * 1000)
		url := fmt.Sprintf("http://%s:%d/api/user/info/1?select=id,name", "127.0.0.1", cfg.Service.Port)
		utils.Dump(url)

		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		res := maps.MakeMapStr()
		err = jsoniter.Unmarshal(body, &res)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	// 等待服务启动
	times := 0
	for times < 20 { // 2秒超时
		times++
		res, err := request()
		if err != nil {
			continue
		}
		assert.Equal(t, 1, any.Of(res.Get("id")).CInt())
		assert.Equal(t, "管理员", res.Get("name"))
		return
	}

	assert.True(t, false)
}

func TestCommandMigrate(t *testing.T) {
	assert.NotPanics(t, func() {
		Migrate()
	})
}
