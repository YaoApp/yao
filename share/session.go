package share

import (
	"context"
	"fmt"
	"io"
	"log"

	klog "github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/network"

	"github.com/buraksezer/olric"
	config_olric "github.com/buraksezer/olric/config"
	"github.com/yaoapp/gou/session"
)

var sessServer *olric.Olric

// SessionPort Session 端口
var SessionPort int

// SessionMemberPort Session Member Port
var SessionMemberPort int

func init() {
	if config.Conf.Session.Store == "server" {
		SessionPort = network.FreePort()
		SessionMemberPort = network.FreePort()
		klog.Trace("Session port: %d, Member Port: %d", SessionPort, SessionMemberPort)
	}
}

// SessionConnect Connect redis server
func SessionConnect() {
	args := []string{}
	if config.Conf.Session.Port == "" {
		config.Conf.Session.Port = "6379"
	}

	if config.Conf.Session.DB == "" {
		config.Conf.Session.DB = "1"
	}

	args = append(args, config.Conf.Session.Port, config.Conf.Session.DB, config.Conf.Session.Password)
	rdb, err := session.NewRedis(config.Conf.Host, args...)
	if err != nil {
		panic(err)
	}

	session.Register("redis", rdb)
	session.Name = "redis"
	klog.Trace("Session Store:REDIS HOST:%s PORT:%s DB:%s", config.Conf.Session.Host, config.Conf.Session.Port, config.Conf.Session.DB)
}

// SessionServerStop 关闭会话服务器
func SessionServerStop() {
	if sessServer != nil {
		sessServer.Shutdown(context.Background())
	}
}

// SessionServerStart 启动会话服务器
func SessionServerStart() {

	// SessionPort := 0
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
	m.BindAddr = "127.0.0.1"
	m.BindPort = SessionMemberPort
	// m.AdvertiseAddr = "127.0.0.1"
	// m.AdvertisePort = SessionPort
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
		// fmt.Println("Session Port IS:", SessionPort) // DEBUG
		err = sessServer.Start() // Call Start at background. It's a blocker call.
		if err != nil {
			fmt.Println(err)
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
