package share

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/buraksezer/olric"
	"github.com/buraksezer/olric/client"
	config_olric "github.com/buraksezer/olric/config"
	"github.com/buraksezer/olric/serializer"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xiang/config"
)

// SessionConnect 加载会话信息
func SessionConnect() {
	if config.Conf.Session.Hosting {
		SessionServer()
		return
	}

	var clientConfig = &client.Config{
		Servers:    []string{fmt.Sprintf("%s:%d", config.Conf.Session.Host, config.Conf.Session.Port)},
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

// SessionServer 启动会话服务器
func SessionServer() {

	c := config_olric.New("local")
	c.BindAddr = config.Conf.Session.Host
	c.BindPort = config.Conf.Session.Port

	c.Logger.SetOutput(ioutil.Discard) // 暂时关闭日志
	ctx, cancel := context.WithCancel(context.Background())
	c.Started = func() {
		defer cancel()
		// log.Println("[INFO] Olric is ready to accept connections")
	}

	db, err := olric.New(c)
	if err != nil {
		log.Fatalf("Failed to create Olric instance: %v", err)
	}

	go func() {
		err = db.Start() // Call Start at background. It's a blocker call.
		if err != nil {
			log.Fatalf("olric.Start returned an error: %v", err)
		}
	}()

	<-ctx.Done()
	dm, err := db.NewDMap("local-session")
	if err != nil {
		log.Fatalf("olric.NewDMap returned an error: %v", err)
	}

	session.MemoryUse(session.ServerDMap{DMap: dm})
}
