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
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/service"
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

	os.Args = append(os.Args, "migrate", "--reset", "--force")
	assert.NotPanics(t, func() {
		main()
	})
}

func TestCommandStart(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
		service.Stop(func() {})
		log.Println("服务已关闭")
	}()
	go func() {
		os.Args = append(os.Args, "start")
		main()
	}()

	// 发送请求
	request := func() (maps.MapStr, error) {
		time.Sleep(time.Microsecond * 2000)
		url := fmt.Sprintf("http://%s:%d/api/user/find/1?select=id,name", "127.0.0.1", config.Conf.Port)
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
	time.Sleep(time.Second * 2)
	times := 0
	for times < 20 { // 2秒超时
		times++
		fmt.Printf("Trying(%d)...", times)
		res, err := request()
		if err != nil {
			fmt.Printf(" %s\n", err.Error())
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
		url := fmt.Sprintf("http://%s:%d/api/user/find/1?select=id,name", "127.0.0.1", config.Conf.Port)
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
	time.Sleep(time.Second * 2)
	times := 0
	for times < 20 { // 2秒超时
		times++
		res, err := request()
		if err != nil {
			fmt.Println("REQUEST ERROR:", err)
			continue
		}
		assert.Equal(t, 1, any.Of(res.Get("id")).CInt())
		assert.Equal(t, "管理员", res.Get("name"))

		// 测试关闭
		service.Stop(func() { log.Println("服务已关闭") })
		time.Sleep(time.Second * 5)
		_, err = request()
		assert.NotNil(t, err)
		return
	}

	assert.True(t, false)
}
