package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/kun/utils"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/global"
)

func TestCommandVersion(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = append(os.Args, "version")
	assert.NotPanics(t, func() {
		main()
	})
}

func TestCommandMigrate(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = append(os.Args, "migrate")
	assert.NotPanics(t, func() {
		main()
	})
}

func TestCommandStart(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
		global.ServiceStop(func() {})
		log.Println("服务已关闭")
	}()
	go func() {
		os.Args = append(os.Args, "start")
		main()
	}()

	// 发送请求
	request := func() (maps.MapStr, error) {
		time.Sleep(time.Microsecond * 2000)
		url := fmt.Sprintf("http://%s:%d/api/user/find/1?select=id,name", "local.iqka.com", config.Conf.Service.Port)
		// utils.Dump(url)
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

func TestCommandStop(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()
	go func() {
		os.Args = append(os.Args, "start")
		main()
	}()

	// 发送请求
	request := func() (maps.MapStr, error) {
		time.Sleep(time.Microsecond * 2000)
		url := fmt.Sprintf("http://%s:%d/api/user/find/1?select=id,name", "local.iqka.com", config.Conf.Service.Port)
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

		// 测试关闭
		global.ServiceStop(func() { log.Println("服务已关闭") })
		time.Sleep(time.Second * 2)
		_, err = request()
		assert.NotNil(t, err)
		return
	}

	assert.True(t, false)
}
