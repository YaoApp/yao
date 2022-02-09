package share

import (
	"context"
	"fmt"
	"io"
	"log"

	klog "github.com/yaoapp/kun/log"

	"github.com/buraksezer/olric"
	"github.com/buraksezer/olric/client"
	config_olric "github.com/buraksezer/olric/config"
	"github.com/buraksezer/olric/serializer"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/network"
)

var sessServer *olric.Olric

// SessionPort Session 端口
var SessionPort int

func init() {
	SessionPort = network.FreePort()
	klog.Trace("session port: %d", SessionPort)
}

// SessionConnect 加载会话信息
func SessionConnect(conf config.SessionConfig) {

	var clientConfig = &client.Config{
		Servers:    []string{fmt.Sprintf("%s:%d", "127.0.0.1", SessionPort)},
		Serializer: serializer.NewMsgpackSerializer(),
		Client:     config_olric.NewClient(),
	}

	c, err := client.New(clientConfig)
	if err != nil {
		exception.New("会话服务器连接失败 %s", 500, err.Error()).Throw()
	}

	dm := c.NewDMap("local-session")
	session.MemoryUse(session.ClientDMap{DMap: dm})
}

// SessionServerStop 关闭会话服务器
func SessionServerStop() {
	if sessServer != nil {
		sessServer.Shutdown(context.Background())
	}
}

// SessionServerStart 启动会话服务器
func SessionServerStart() {

	c := &config_olric.Config{
		BindAddr:          "127.0.0.1",
		BindPort:          SessionPort,
		ReadRepair:        false,
		ReplicaCount:      1,
		WriteQuorum:       1,
		ReadQuorum:        1,
		MemberCountQuorum: 1,
		Peers:             []string{},
		DMaps:             &config_olric.DMaps{},
		StorageEngines:    config_olric.NewStorageEngine(),
		Logger:            &log.Logger{},
	}

	m, err := config_olric.NewMemberlistConfig("local")
	if err != nil {
		panic(fmt.Sprintf("unable to create a new memberlist config: %v", err))
	}
	// m.BindAddr = config.Conf.Session.Host
	m.BindPort = config.Conf.Session.Port
	m.AdvertisePort = config.Conf.Session.Port
	c.MemberlistConfig = m

	// c.MemberlistConfig.BindAddr = config.Conf.Session.Host
	// c.MemberlistConfig.BindPort = 3308

	// c := config_olric.New("local")
	// c.BindAddr = config.Conf.Session.Host
	// c.BindPort = config.Conf.Session.Port

	c.Logger.SetOutput(io.Discard) // 暂时关闭日志
	ctx, cancel := context.WithCancel(context.Background())
	c.Started = func() {
		defer cancel()
		klog.Trace("[INFO] Olric is ready to accept connections")
	}

	sessServer, err = olric.New(c)

	if err != nil {
		klog.Error("Failed to create Olric instance: %v", err)
	}

	go func() {
		err = sessServer.Start() // Call Start at background. It's a blocker call.
		if err != nil {
			klog.Panic("olric.Start returned an error: %v", err)
		}
	}()

	<-ctx.Done()
	dm, err := sessServer.NewDMap("local-session")
	if err != nil {
		klog.Panic("olric.NewDMap returned an error: %v", err)
	}

	session.MemoryUse(session.ServerDMap{DMap: dm})
}
